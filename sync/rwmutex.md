## sync.RWMutex

```go
package sync

import (
	"internal/race"
	"sync/atomic"
	"unsafe"
)

type RWMutex struct {
	w           Mutex  // 写操作保护
	writerSem   uint32 // 写者信号量
	readerSem   uint32 // 读者信号量
	readerCount int32  // pending 状态读者数量
	readerWait  int32  // 正在执行的读者数量
}

const rwmutexMaxReaders = 1 << 30   // 读者最大数量

func (rw *RWMutex) RLock() {
	if race.Enabled {
		_ = rw.w.state
		race.Disable()
	}
	// 写锁正在执行，等待
	if atomic.AddInt32(&rw.readerCount, 1) < 0 {
	    // 等待写锁结束
		runtime_Semacquire(&rw.readerSem)
	}
	
	if race.Enabled {
		race.Enable()
		race.Acquire(unsafe.Pointer(&rw.readerSem)) // 获取读者信号量
	}
}

func (rw *RWMutex) RUnlock() {
	if race.Enabled {
		_ = rw.w.state
		race.ReleaseMerge(unsafe.Pointer(&rw.writerSem))    // 获取写信号量
		race.Disable()
	}
	
	// 释放一个读者
	if r := atomic.AddInt32(&rw.readerCount, -1); r < 0 {
	    // r + 1 == 0: 没有读者
	    // r + 1 == -rwmutexMaxReaders: 写锁 pending，没有读者
		if r+1 == 0 || r+1 == -rwmutexMaxReaders {
			race.Enable()
			throw("sync: RUnlock of unlocked RWMutex")
		}
		
		// 写锁 pending，且为最后一个读者
		if atomic.AddInt32(&rw.readerWait, -1) == 0 {
			// 释放写锁信号量
			runtime_Semrelease(&rw.writerSem)
		}
	}
	if race.Enabled {
		race.Enable()
	}
}

func (rw *RWMutex) Lock() {
	if race.Enabled {
		_ = rw.w.state
		race.Disable()
	}
	
	// 解决多写者间竞争 
	rw.w.Lock()
	
	// 通知读者，写锁正在进行中
	r := atomic.AddInt32(&rw.readerCount, -rwmutexMaxReaders) + rwmutexMaxReaders
	
	// 等待读锁完成
	// atomic.AddInt32(&rw.readerWait, r): 设置当前正在执行的读者
	if r != 0 && atomic.AddInt32(&rw.readerWait, r) != 0 {
		runtime_Semacquire(&rw.writerSem)
	}
	if race.Enabled {
		race.Enable()
		race.Acquire(unsafe.Pointer(&rw.readerSem)) // 获取 reader 信号量
		race.Acquire(unsafe.Pointer(&rw.writerSem)) // 获取 writer 信号量
	}
}

func (rw *RWMutex) Unlock() {
	if race.Enabled {
		_ = rw.w.state
		race.Release(unsafe.Pointer(&rw.readerSem)) // 释放读信号量
		race.Release(unsafe.Pointer(&rw.writerSem)) // 释放写信号量
		race.Disable()
	}

	// 通知读者没有写者
	r := atomic.AddInt32(&rw.readerCount, rwmutexMaxReaders)
	
	// 释放未加锁的写锁
	if r >= rwmutexMaxReaders {
		race.Enable()
		throw("sync: Unlock of unlocked RWMutex")
	}
	
	// 释放全部读者信号量 
	for i := 0; i < int(r); i++ {
		runtime_Semrelease(&rw.readerSem)
	}
	
	// 解锁，其他写着可以进行
	rw.w.Unlock()
	if race.Enabled {
		race.Enable()
	}
}

func (rw *RWMutex) RLocker() Locker {
	return (*rlocker)(rw)
}

type rlocker RWMutex

func (r *rlocker) Lock()   { (*RWMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*RWMutex)(r).RUnlock() }

```
