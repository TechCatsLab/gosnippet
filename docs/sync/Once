## sync.Once

```go
package sync

import (
	"sync/atomic"
)

type Once struct {
	m    Mutex       // 确保函数只执行一次
	done uint32      // 执行次数计数，初始为 0
}

func (o *Once) Do(f func()) {
    // 快速通道，如果为 1，则已经执行
	if atomic.LoadUint32(&o.done) == 1 {
		return
	}
	
	// 慢速，确保函数只执行一次
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		defer atomic.StoreUint32(&o.done, 1)
		f()
	}
}
```