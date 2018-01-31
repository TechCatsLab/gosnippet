## sync.WaitGroup

```go
package sync

import (
	"internal/race"
	"sync/atomic"
	"unsafe"
)

type WaitGroup struct {
	noCopy noCopy

    // 64-bit 值， 高 32-bit 为计数器，低 32-bit 为等待者(执行 Wait 的协程数)计数器
    // 64-bit 原子操作，需要 64-bit(8-byte) 对齐，但是 32 位编译器不能保证，因此，多余 4 byte 用于保证 8-byte 对齐
    // 64-bit 对齐含义为： 变量地址 % 8 == 0
	state1 [12]byte
	sema   uint32
}

func (wg *WaitGroup) state() *uint64 {
    // 64-bit 对齐，直接返回地址
	if uintptr(unsafe.Pointer(&wg.state1))%8 == 0 {
		return (*uint64)(unsafe.Pointer(&wg.state1))
	} else {
	    // 32-bit 对齐，对齐到 64-bit 再返回
		return (*uint64)(unsafe.Pointer(&wg.state1[4]))
	}
}

func (wg *WaitGroup) Add(delta int) {
	statep := wg.state()
	if race.Enabled {
		_ = *statep 
		if delta < 0 {
			race.ReleaseMerge(unsafe.Pointer(wg))
		}
		race.Disable()
		defer race.Enable()
	}
	
	// 等待计数加 delta，注意移位操作
	state := atomic.AddUint64(statep, uint64(delta)<<32)
	v := int32(state >> 32)     // 可以为负值，所以使用 int32
	w := uint32(state)          // 不可为负值，使用 uint32
	
	if race.Enabled {
		if delta > 0 && v == int32(delta) {
			// v == int32(delta): 首次执行 Add；使用信号量，保证与 Wait 操作的同步
			race.Read(unsafe.Pointer(&wg.sema))
		}
	}
	
	// 计数为负，出现错误
	if v < 0 {
		panic("sync: negative WaitGroup counter")
	}
	
	// w != 0：Wait 已触发至少一次
	// v == int32(delta)：首次 Add
	// Wait 与 Add 并发执行造成，错误！
	if w != 0 && delta > 0 && v == int32(delta) {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	}
	
	// 计数器大于 0
	// 计数器与等待者计数均为 0，直接退出
	if v > 0 || w == 0 {
		return
	}
	
	// v == 0 && w > 0
	// v < 0 早已 panic，因此，此处 v == 0
	if *statep != state {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	}
	
	// 将等待者计数清零（执行到此处时，计数器已经为 0）
	*statep = 0
	
	// 释放全部等待者
	for ; w != 0; w-- {
		runtime_Semrelease(&wg.sema)
	}
}

func (wg *WaitGroup) Done() {
    // 计数器减一
	wg.Add(-1)
}

func (wg *WaitGroup) Wait() {
	statep := wg.state()
	if race.Enabled {
		_ = *statep
		race.Disable()
	}
	
	for {
		state := atomic.LoadUint64(statep)
		v := int32(state >> 32)
		w := uint32(state)
		if v == 0 {
			// 计数器为 0，不需要等待
			if race.Enabled {
				race.Enable()
				race.Acquire(unsafe.Pointer(wg))
			}
			return
		}
		
		// 等待计数加 1，注意，如果 Add 同步执行，那么这里执行不成功，直接进入下次循环
		if atomic.CompareAndSwapUint64(statep, state, state+1) {
		    // 首次等待要与首次 Add 同步执行，配合 Add 中逻辑
			if race.Enabled && w == 0 {
				race.Write(unsafe.Pointer(&wg.sema))
			}
			
			// 获取等待信号量，阻塞在此处
			runtime_Semacquire(&wg.sema)
			
			// 等待完成，恢复执行；
			// 此时，不可以重用
			if *statep != 0 {
				panic("sync: WaitGroup is reused before previous Wait has returned")
			}
			
			if race.Enabled {
				race.Enable()
				race.Acquire(unsafe.Pointer(wg))
			}
			return
		}
	}
}

```
