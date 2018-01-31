## sync.Cond
```go
package sync

import (
	"sync/atomic"
	"unsafe"
)

type Cond struct {
	noCopy noCopy

	L Locker                // 锁接口

	notify  notifyList      // 通知列表
	checker copyChecker
}

// Cond 必须配合锁使用
func NewCond(l Locker) *Cond {
	return &Cond{L: l}
}

func (c *Cond) Wait() {
	c.checker.check()                       // 确保没有被复制
	t := runtime_notifyListAdd(&c.notify)   // 将自己加入通知队列
	c.L.Unlock()                            // 解锁，使用 Wait 前，需要加锁，这里释放，可以让其他含 Signal, Broadcast 的协程使用
	runtime_notifyListWait(&c.notify, t)    // 是否有通知
	c.L.Lock()                              // 加锁，使外部解锁代码不至于出错
}

func (c *Cond) Signal() {
	c.checker.check()
	runtime_notifyListNotifyOne(&c.notify)  // 释放一个等待事件
}

func (c *Cond) Broadcast() {
	c.checker.check()
	runtime_notifyListNotifyAll(&c.notify)
}

type copyChecker uintptr

func (c *copyChecker) check() {
	if uintptr(*c) != uintptr(unsafe.Pointer(c)) &&
		!atomic.CompareAndSwapUintptr((*uintptr)(c), 0, uintptr(unsafe.Pointer(c))) &&
		uintptr(*c) != uintptr(unsafe.Pointer(c)) {
		panic("sync.Cond is copied")
	}
}

type noCopy struct{}

func (*noCopy) Lock() {}
```
