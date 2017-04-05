## sync.Mutex

```go
package sync

import (
	"internal/race"
	"sync/atomic"
	"unsafe"
)

func throw(string)  // 运行时，抛出异常

type Mutex struct {
	state int32     // 初始为 0，未加锁
	sema  uint32
}

// 通用锁接口
type Locker interface {
	Lock()
	Unlock()
}

// 注意理解 iota，参考代码：iota.go
// mutexWoken = 1 << iota， 此时 iota = 1
// mutexWaiterShift 强制赋值 iota，此时 iota = 2
// mutexWaiterShift 含义为：从该位起，计数等待者数量；注意这类写法！
const (
	mutexLocked = 1 << iota // 加锁状态: 1
	mutexWoken              // 唤醒： 2
	mutexWaiterShift = iota // 等待者移位：2
)

func (m *Mutex) Lock() {
    // 尝试加锁，如果成功，直接返回
    // 注意原子操作快速方式获取锁的优化！！！
	if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
		if race.Enabled {
			race.Acquire(unsafe.Pointer(m))
		}
		return
	}

    // 快速通道未成功获取锁，进入自旋
    // 自旋逻辑为：CPU 空转时间比协程调度时间还要短
	awoke := false
	iter := 0       // 自旋次数
	for {
		old := m.state      // 获取当前 state 值
		new := old | mutexLocked    // 设置当前 state 值获得锁时状态值
		
		// 当前 mutex 处于加锁状态 
		if old&mutexLocked != 0 {
		    // 自旋次数没有超限；先尝试通过自旋方式获取锁，如果不成功，再等待
		    // 注意此类保护，防止无限自旋，反而消耗更多 CPU 资源
			if runtime_canSpin(iter) {
			    // !awoke: 自己未被唤醒
			    // old & mutexWoken == 0: 锁当前不处于唤醒状态
			    // old>>mutexWaiterShift != 0: 有其他等待者
			    // atomic: 成功标记锁为唤醒状态
				if !awoke && old&mutexWoken == 0 && old>>mutexWaiterShift != 0 &&
					atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) {
					awoke = true
				}
				runtime_doSpin()    // 执行自旋操作
				iter++
				continue
			}
			
			// 自旋完毕，加锁依然不成功，等待者计数加 1
			new = old + 1<<mutexWaiterShift
		}
		
		// 自旋过程中，如果自己被标记为唤醒状态，需要检查 mutex 状态
		// 自选过程中，不能保证 awoke 为 true
		// 如果只有一个协程在获取锁，awoke 为 false
		if awoke {
			if new&mutexWoken == 0 {
				throw("sync: inconsistent mutex state")
			}
			new &^= mutexWoken  // 清除 mutexWoken 标记
		}
		
		// 加锁
		if atomic.CompareAndSwapInt32(&m.state, old, new) {
		    // 加锁成功时，锁已被释放，则退出循环
			if old&mutexLocked == 0 {
				break
			}
			runtime_SemacquireMutex(&m.sema)
			awoke = true
			iter = 0
		}
	}

	if race.Enabled {
		race.Acquire(unsafe.Pointer(m))
	}
}

func (m *Mutex) Unlock() {
	if race.Enabled {
		_ = m.state
		race.Release(unsafe.Pointer(m))
	}

    // 如果未加锁，抛出异常
	new := atomic.AddInt32(&m.state, -mutexLocked)
	if (new+mutexLocked)&mutexLocked == 0 {
		throw("sync: unlock of unlocked mutex")
	}

	old := new
	for {
	    // 只有一个等待者
	    // 有多个等待者，并且已有唤醒者
		if old>>mutexWaiterShift == 0 || old&(mutexLocked|mutexWoken) != 0 {
			return
		}
		
		// 有多个等待者，且没有唤醒者，此时，old 一定为 4 的整数倍
		// 减少一个等待者，并标识唤醒位
		new = (old - 1<<mutexWaiterShift) | mutexWoken
		
		// 释放锁成功，退出；
		// 不成功，再次循环;不成功是由于加锁部分也在执行
		if atomic.CompareAndSwapInt32(&m.state, old, new) {
			runtime_Semrelease(&m.sema)
			return
		}
		old = m.state
	}
}
```
