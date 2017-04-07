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
