## Semaphore (runtime/sema.go)
```go
// uintptr is an integer type that is large enough to hold the bit pattern of
// any pointer.
type uintptr uintptr

// 使用指针隐藏底层实现
type mutex struct {
	key uintptr
}

type semaRoot struct {
	lock  mutex
	head  *sudog
	tail  *sudog
	nwait uint32        // 等待计数
}

// 参照 semroot 实现
const semTabSize = 251

var semtable [semTabSize]struct {
	root semaRoot
	// 对齐到 cacheLine，优化技巧！！
	// 注意不要滥用，对非核心数据不要这样使用
	pad  [sys.CacheLineSize - unsafe.Sizeof(semaRoot{})]byte
}

// 获取 semaRoot 地址
func semroot(addr *uint32) *semaRoot {
	return &semtable[(uintptr(unsafe.Pointer(addr))>>3)%semTabSize].root
}

// 是否可以获取
func cansemacquire(addr *uint32) bool {
	for {
		v := atomic.Load(addr)
		
		// 如果为 0，不能获取信号量，返回
		if v == 0 {
			return false
		}
		
		// 自旋至获取到信号量，返回
		// Cas: Compare And Switch
		if atomic.Cas(addr, v, v-1) {
			return true
		}
	}
}

type semaProfileFlags int
const (
	semaBlockProfile semaProfileFlags = 1 << iota
	semaMutexProfile
)

// 函数封装
// 通过参数变化调用同一函数，隐藏实现细节，这样的技巧必须掌握
// 好处是，对外的接口不需要任何变更，底层实现可随时调整，不影响其他代码
func sync_runtime_Semacquire(addr *uint32) {
	semacquire(addr, semaBlockProfile)
}

func net_runtime_Semacquire(addr *uint32) {
	semacquire(addr, semaBlockProfile)
}

func sync_runtime_Semrelease(addr *uint32) {
	semrelease(addr)
}

func sync_runtime_SemacquireMutex(addr *uint32) {
	semacquire(addr, semaBlockProfile|semaMutexProfile)
}

// 获取信号量
func semacquire(addr *uint32, profile semaProfileFlags) {
    // 获取当前 goroutine
	gp := getg()
	
	// 当前执行 goroutine 不是当前栈运行 goroutine
	if gp != gp.m.curg {
		throw("semacquire not on the G stack")
	}

	// Easy case.
	if cansemacquire(addr) {
		return
	}

	s := acquireSudog()
	root := semroot(addr)
	t0 := int64(0)
	
	// 时间统计，不要深究
	s.releasetime = 0
	s.acquiretime = 0
	if profile&semaBlockProfile != 0 && blockprofilerate > 0 {
		t0 = cputicks()
		s.releasetime = -1
	}
	if profile&semaMutexProfile != 0 && mutexprofilerate > 0 {
		if t0 == 0 {
			t0 = cputicks()
		}
		s.acquiretime = t0
	}
	
	// 进入获取信号量过程
	for {
		lock(&root.lock)                    // 加锁保护
		atomic.Xadd(&root.nwait, 1)         // 等待计数加一，将调用者计数
		
		// 检查是否已有信号量释放，快速获取
		if cansemacquire(addr) {
			atomic.Xadd(&root.nwait, -1)    // 等待计数减一
			unlock(&root.lock)              // 解锁
			break
		}
		
		// 没有释放的信号量，进入队列
		root.queue(addr, s)
		
		// 将当前 goroutine 放入等待队列，并解锁
		goparkunlock(&root.lock, "semacquire", traceEvGoBlockSync, 4)
		
		// 触发调度后，检查是否有信号量释放
		if cansemacquire(addr) {
			break
		}
	}
	
	if s.releasetime > 0 {
		blockevent(s.releasetime-t0, 3)
	}
	releaseSudog(s)
}

func semrelease(addr *uint32) {
	root := semroot(addr)
	
	// 释放信号量
	atomic.Xadd(addr, 1)

    // 没有等待者，直接返回
	if atomic.Load(&root.nwait) == 0 {
		return
	}

	// 尝试激活一个等待者
	lock(&root.lock)
	
	// 等待者已经被激活(参考 cansemacquire)
	if atomic.Load(&root.nwait) == 0 {
		unlock(&root.lock)
		return
	}
	
	// 遍历等待队列
	s := root.head
	for ; s != nil; s = s.next {
	    // 找到第一个
		if s.elem == unsafe.Pointer(addr) {
			atomic.Xadd(&root.nwait, -1)        // 等待计数减一
			root.dequeue(s)                     // 出队
			break
		}
	}
	
	if s != nil {
		if s.acquiretime != 0 {
			t0 := cputicks()
			for x := root.head; x != nil; x = x.next {
				if x.elem == unsafe.Pointer(addr) {
					x.acquiretime = t0
				}
			}
			mutexevent(t0-s.acquiretime, 3)
		}
	}
	unlock(&root.lock)                          // 处理完成，尽快解锁！
	
	if s != nil {
		readyWithTime(s, 5)
	}
}
```

## Notify List
```go
type notifyList struct {
	wait uint32         // 下个等待者的 ticket
	notify uint32       // 下个唤醒的 ticket

	lock mutex          // 等待队列相关结构
	head *sudog
	tail *sudog
}

func less(a, b uint32) bool {
	return int32(a-b) < 0
}

func notifyListAdd(l *notifyList) uint32 {
    // 增加计数，并返回原有计数
    // 获取当前等待 ticket
    // 并调整下个 ticket
	return atomic.Xadd(&l.wait, 1) - 1
}

func notifyListWait(l *notifyList, t uint32) {
	lock(&l.lock)

    // 如果已经通知过 ticket，不需要等待
	if less(t, l.notify) {
		unlock(&l.lock)
		return
	}

    // 没有通知到，入队
	s := acquireSudog()
	s.g = getg()
	s.ticket = t            // 设置 ticket
	s.releasetime = 0
	t0 := int64(0)
	if blockprofilerate > 0 {
		t0 = cputicks()
		s.releasetime = -1
	}
	
	// 入队尾
	if l.tail == nil {
		l.head = s
	} else {
		l.tail.next = s
	}
	l.tail = s
	
	// 当前 goroutine 进入等待状态，并解锁
	goparkunlock(&l.lock, "semacquire", traceEvGoBlockCond, 3)
	
	// 恢复调度后执行
	if t0 != 0 {
		blockevent(s.releasetime-t0, 2)
	}
	releaseSudog(s)
}

func notifyListNotifyAll(l *notifyList) {
	// 不需要通知
	if atomic.Load(&l.wait) == atomic.Load(&l.notify) {
		return
	}

    // 注意这里的操作：
    // 加锁，处理链表，解锁后，链表内容不会发生变化！
	lock(&l.lock)
	s := l.head
	l.head = nil
	l.tail = nil

	// 修改 notify ticket 为最大号
	atomic.Store(&l.notify, atomic.Load(&l.wait))
	unlock(&l.lock)

	// 激活全部等待者
	for s != nil {
		next := s.next
		s.next = nil
		readyWithTime(s, 4)
		s = next
	}
}

func notifyListNotifyOne(l *notifyList) {
	if atomic.Load(&l.wait) == atomic.Load(&l.notify) {
		return
	}

	lock(&l.lock)

	// 加锁后再次检查
	t := l.notify
	if t == atomic.Load(&l.wait) {
		unlock(&l.lock)
		return
	}

    // 通知序号加一，用于下次通知
	atomic.Store(&l.notify, t+1)
	
	// 遍历等待队列，找到 ticket == t 的节点
	for p, s := (*sudog)(nil), l.head; s != nil; p, s = s, s.next {
		if s.ticket == t {
		    // s 出队
			n := s.next
			if p != nil {
				p.next = n
			} else {
				l.head = n
			}
			
			if n == nil {
				l.tail = p
			}
			unlock(&l.lock)     // 不需要锁时，尽早释放！
			
			s.next = nil
			readyWithTime(s, 4)
			return
		}
	}
	unlock(&l.lock)
}
```
