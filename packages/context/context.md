## 什么是context

从go1.7开始，golang.org/x/net/context包正式作为context包进入了标准库。那么，这个包到底是做什么的呢？根据官方的文档说明：

```
Package context defines the Context type, which carries deadlines,
cancelation signals, and other request-scoped values across API
boundaries and between processes.
```

也就是说，通过context，我们可以方便地对同一个请求所产生地goroutine进行约束管理，可以设定超时、deadline，甚至是取消这个请求相关的所有goroutine。

## 如何使用context

先来看看context的代码示例

```go
package main

import (
    "context"
    "log"
    "net/http"
    _ "net/http/pprof"
    "time"
)

func main() {
    go http.ListenAndServe(":8989", nil)
    ctx, cancel := context.WithCancel(context.Background())
    go func() {
        time.Sleep(3 * time.Second)
        cancel()
    }()
    log.Println(Add(ctx))
    select {}
}

func C(ctx context.Context) string {
    select {
    case <-ctx.Done():
        return "C Done"
    }
    return ""
}

func B(ctx context.Context) string {
    ctx, _ = context.WithCancel(ctx)
    go log.Println(C(ctx))
    select {
    case <-ctx.Done():
        return "B Done"
    }
    return ""
}

func A(ctx context.Context) string {
    go log.Println(B(ctx))
    select {
    case <-ctx.Done():
        return "A Done"
    }
    return ""
}
```

## context.Context 源码导读

```go
package context

import (
    "errors"
    "fmt"
    "reflect"
    "sync"
    "time"
)

type Context interface {
    // Context 是否超时
    Deadline() (deadline time.Time, ok bool)

    // Context 是否结束
    Done() <-chan struct{}

    // 获取错误
    Err() error

    // 获取 Key/Value
    Value(key interface{}) interface{}
}

// Context 已经取消
var Canceled = errors.New("context canceled")

// Context 已超时
var DeadlineExceeded error = deadlineExceededError{}

// 自定义错误
type deadlineExceededError struct{}
func (deadlineExceededError) Error() string   { return "context deadline exceeded" }
func (deadlineExceededError) Timeout() bool   { return true }
func (deadlineExceededError) Temporary() bool { return true }

// 空 Context，直接使用 int 类型，注意这种技巧
type emptyCtx int

// 带返回值参数，ok 默认 false
func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
    return
}

// 作为 Context 根，永不结束
func (*emptyCtx) Done() <-chan struct{} {
    return nil
}

// 不报错
func (*emptyCtx) Err() error {
    return nil
}

// 没有携带任何 Key/Value
func (*emptyCtx) Value(key interface{}) interface{} {
    return nil
}

// 只有两个实例：background, todo
func (e *emptyCtx) String() string {
    switch e {
    case background:
        return "context.Background"
    case todo:
        return "context.TODO"
    }
    return "unknown empty Context"
}

var (
    background = new(emptyCtx)      // 地址，全局唯一
    todo       = new(emptyCtx)      // 地址，全局唯一
)

// 获取根 Context : background
func Background() Context {
    return background
}

// 获取根 Context : todo
func TODO() Context {
    return todo
}

// 定义取消函数原型
type CancelFunc func()

// 创建带 cancel 功能的子 Context
// interface 可以适配指针，也是常用技巧
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
    c := newCancelCtx(parent)       // 创建新 Context
    propagateCancel(parent, &c)     // 根据父节点状态，设定当前 Context 状态
    return &c, func() { c.cancel(true, Canceled) }  // 注意，返回的 Context 仍为地址!
}

func newCancelCtx(parent Context) cancelCtx {
    return cancelCtx{
        Context: parent,                // parent 实际类型为一个 Context 指针
        done:    make(chan struct{}),   // 定义 done channel
    }
}

func propagateCancel(parent Context, child canceler) {
    // 父节点为 emptyCtx， 直接返回
    // 配合后面 goroutine，在 goroutine 内部不可能出现 nil channel；
    if parent.Done() == nil {
        return
    }

    // 如果上级节点中有 Cancel Context
    if p, ok := parentCancelCtx(parent); ok {
        p.mu.Lock()         // 加锁，保护 children
        if p.err != nil {
            // 如果上级 Cancel Context 已经取消，那么标识自己为已取消
            // false: 由于是新建的 Cancel Context，所以没有子 Context，不需要执行取消子 Context 动作
            child.cancel(false, p.err)
        } else {
            // 如果上级 Cancel Context 没有取消，在其 children 中添加自己
            if p.children == nil {
                // 是上级 Cancel Context 的第一个子 Cancel Context，新建 children
                p.children = make(map[canceler]struct{})
            }

            // 这里是配合 done，struct{}{} 是 struct{} 一个实例
            p.children[child] = struct{}{}
        }
        p.mu.Unlock()
    } else {
        // 上级节点没有 Cancel Context，启动接收协程，防止死锁；
        // 无缓冲 chan，必须先有接受者，才能发送数据
        go func() {
            select {
            // 父节点已结束，那么自己要停止
            case <-parent.Done():
                child.cancel(false, parent.Err())

            // 自己结束，正常退出
            case <-child.Done():
            }
        }()
    }
}

func parentCancelCtx(parent Context) (*cancelCtx, bool) {
    // 获取上级节点中距离最近的 Cancel Context
    for {
        switch c := parent.(type) {
        case *cancelCtx:
            return c, true
        case *timerCtx:
            return &c.cancelCtx, true
        case *valueCtx:
            parent = c.Context
        default:
            return nil, false
        }
    }
}

func removeChild(parent Context, child canceler) {
    // 从上级节点(parent)中移除一个节点(child)
    p, ok := parentCancelCtx(parent)
    if !ok {
        return
    }
    p.mu.Lock()
    if p.children != nil {
        delete(p.children, child)
    }
    p.mu.Unlock()
}

type canceler interface {
    cancel(removeFromParent bool, err error)
    Done() <-chan struct{}
}

type cancelCtx struct {
    Context

    done chan struct{}

    mu       sync.Mutex             // 默认值为未加锁状态！！所以不需要做额外初始化
    children map[canceler]struct{}
    err      error
}

func (c *cancelCtx) Done() <-chan struct{} {
    return c.done
}

func (c *cancelCtx) Err() error {
    // propagateCancel 中有设置 err，所以需要加锁
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.err
}

func (c *cancelCtx) String() string {
    return fmt.Sprintf("%v.WithCancel", c.Context)
}

func (c *cancelCtx) cancel(removeFromParent bool, err error) {
    // 如果没有错误，不允许 cancel 操作
    if err == nil {
        panic("context: internal error: missing cancel error")
    }

    // 是否首次 cancel，如果已 cancel 过，直接返回
    c.mu.Lock()
    if c.err != nil {
        c.mu.Unlock()
        return
    }
    c.err = err
    close(c.done)       // close 会触发所有对 channel 的读操作返回

    // 取消全部子 Cancel Context
    for child := range c.children {
        child.cancel(false, err)
    }
    c.children = nil
    c.mu.Unlock()

    // 如果是从上级移除，在上级 Cancel Context 中移除自己
    if removeFromParent {
        removeChild(c.Context, c)
    }
}

// 创建 timerCtx
func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
    // 如果上级节点没有超时，直接添加 Cancel 功能即可
    // emptyCtx, cancelCtx Deadline 均返回 false，所以，运行至此，上级一定是 timerCtx
    if cur, ok := parent.Deadline(); ok && cur.Before(deadline) {
        return WithCancel(parent)
    }

    // 创建新 Context，注意，返回仍然是地址!
    c := &timerCtx{
        cancelCtx: newCancelCtx(parent),    // 创建 Cancel Context
        deadline:  deadline,                // 设置超时时刻
    }

    // 上级 Cancel 如果已触发，设置自己为已 cancel
    propagateCancel(parent, c)

    // 如果超时时间已过，设置错误
    d := time.Until(deadline)
    if d <= 0 {
        c.cancel(true, DeadlineExceeded)
        return c, func() { c.cancel(true, Canceled) }
    }

    // 加锁，因为 propagateCancel 可能会设置 err
    c.mu.Lock()
    defer c.mu.Unlock()

    // 无错误时，设置 timer，超时后自动取消
    if c.err == nil {
        c.timer = time.AfterFunc(d, func() {
            c.cancel(true, DeadlineExceeded)
        })
    }
    return c, func() { c.cancel(true, Canceled) }
}

type timerCtx struct {
    cancelCtx               // 匿名子段，包含 cancelCtx 全部成员
    timer *time.Timer

    deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
    return c.deadline, true
}

func (c *timerCtx) String() string {
    return fmt.Sprintf("%v.WithDeadline(%s [%s])", c.cancelCtx.Context, c.deadline, time.Until(c.deadline))
}

func (c *timerCtx) cancel(removeFromParent bool, err error) {
    // 首先调用 cancelCtx 的 cancel 方法
    c.cancelCtx.cancel(false, err)

    // 从上级节点中移除自己
    if removeFromParent {
        removeChild(c.cancelCtx.Context, c)
    }

    // 停止计时器 
    c.mu.Lock()
    if c.timer != nil {
        c.timer.Stop()
        c.timer = nil
    }
    c.mu.Unlock()
}

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
    return WithDeadline(parent, time.Now().Add(timeout))
}

// 返回 valueCtx，注意，一个 valueCtx 只能保存一个 Key/Value 对！！
// parent 可以为任意类型 Context，注意：parent 实际也为地址！！
func WithValue(parent Context, key, val interface{}) Context {
    // 这样的代码保护，要学习
    if key == nil {
        panic("nil key")
    }

    // Key 必须可比较，否则无法获取
    if !reflect.TypeOf(key).Comparable() {
        panic("key is not comparable")
    }

    return &valueCtx{parent, key, val}
}

type valueCtx struct {
    Context
    key, val interface{}
}

func (c *valueCtx) String() string {
    return fmt.Sprintf("%v.WithValue(%#v, %#v)", c.Context, c.key, c.val)
}

// 获取值
func (c *valueCtx) Value(key interface{}) interface{} {
    // 如果本身存储，直接返回
    if c.key == key {
        return c.val
    }

    // 向上遍历
    // 如果 key 不存在，最终会到 emptyCtx，返回 nil
    // 注意，interface 使用，Value 本身是 Context 方法，编译能过；
    // 同时，会根据实际存储的类型，执行对应方法！
    return c.Context.Value(key)
}
```

## context的使用规范
使用 context 的最佳规范：
- 不要把 context 存储在结构体中，而是要显式地进行传递
- 把 context 作为第一个参数，并且一般都把变量命名为 ctx
- 就算是程序允许，也不要传入一个 nil 的 context，如果不知道是否要用 context 的话，用 context.TODO() 来替代
- context.WithValue() 只用来传递请求范围的值，不要用它来传递可选参数
- 就算是被多个不同的 goroutine 使用，context 也是安全的
