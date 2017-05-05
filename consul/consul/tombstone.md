## Tombstone GC
### 结构
```go
type TombstoneGC struct {
	ttl         time.Duration
	granularity time.Duration

	enabled bool                        // 开启控制

	expires map[time.Time]*expireInterval
	expireCh chan uint64

	lock sync.Mutex
}

type expireInterval struct {
	maxIndex uint64
	timer    *time.Timer
}
```

### 方法
#### NewTombstoneGC
```go
func NewTombstoneGC(ttl, granularity time.Duration) (*TombstoneGC, error) {
	// 参数检查
	if ttl <= 0 || granularity <= 0 {
		return nil, fmt.Errorf("Tombstone TTL and granularity must be positive")
	}

	t := &TombstoneGC{
		ttl:         ttl,
		granularity: granularity,
		enabled:     false,                                 // 初始为 false，不启动
		expires:     make(map[time.Time]*expireInterval),
		expireCh:    make(chan uint64, 1),
	}
	return t, nil
}
```

#### 结构方法
```go
// 返回 Expire Chan
func (t *TombstoneGC) ExpireCh() <-chan uint64 {
	return t.expireCh
}

// 设置是否启动
func (t *TombstoneGC) SetEnabled(enabled bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	
	// 状态无变化
	if enabled == t.enabled {
		return
	}

	// 如果关闭，清理全部定时器
	if !enabled {
		for _, exp := range t.expires {
			exp.timer.Stop()
		}
		
		// 重新创建，旧的直接丢弃
		t.expires = make(map[time.Time]*expireInterval)
	}

	t.enabled = enabled
}

func (t *TombstoneGC) Hint(index uint64) {
	expires := t.nextExpires()          // 获取下次出发时刻

	t.lock.Lock()
	defer t.lock.Unlock()
	
	// disable 状态，直接退出
	if !t.enabled {
		return
	}

	// 是否存在该时刻超时定时器
	exp, ok := t.expires[expires]
	if ok {
		// 刷新最大索引
		if index > exp.maxIndex {
			exp.maxIndex = index
		}
		return
	}

	// 创建新条目
	t.expires[expires] = &expireInterval{
		maxIndex: index,
		timer: time.AfterFunc(expires.Sub(time.Now()), func() {
			t.expireTime(expires)
		}),
	}
}

func (t *TombstoneGC) PendingExpiration() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return len(t.expires) > 0
}

func (t *TombstoneGC) nextExpires() time.Time {
	expires := time.Now().Add(t.ttl)    // 添加 TTL (生存时间)
	
	// 补足到时间间隔（粒度）
	remain := expires.UnixNano() % int64(t.granularity) 
	adj := expires.Add(t.granularity - time.Duration(remain))
	return adj
}

// 使过期条目
func (t *TombstoneGC) expireTime(expires time.Time) {
	t.lock.Lock()
	exp := t.expires[expires]
	delete(t.expires, expires)
	t.lock.Unlock()

    // 通知最大条目
	t.expireCh <- exp.maxIndex
}
```
