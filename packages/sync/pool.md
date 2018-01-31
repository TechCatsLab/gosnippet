## sync.pool

```
type Pool struct {
	noCopy noCopy

	local     unsafe.Pointer
	localSize uintptr
    //当pool中无可用对象时，调用New函数产生对象值直接返回给调用方，所以其产生的对象值永远不会被放置到池中
	New func() interface{}
}

//sync.pool为每个P（golang的调度模型介绍中有介绍）都分配了一个子池。
//每个子池里面有一个私有对象和共享列表对象，私有对象是只有对应的P能够访问，
//因为一个P同一时间只能执行一个goroutine，因此对私有对象存取操作是不需要加锁的。
//共享列表是和其他P分享的，因此操作共享列表是需要加锁的。
type poolLocal struct {
	private interface{}
	shared  []interface{}
	Mutex
	pad     [128]byte
}


func fastrand() uint32

var poolRaceHash [128]uint64

func poolRaceAddr(x interface{}) unsafe.Pointer {
	ptr := uintptr((*[2]unsafe.Pointer)(unsafe.Pointer(&x))[1])
	h := uint32((uint64(uint32(ptr)) * 0x85ebca6b) >> 16)
	return unsafe.Pointer(&poolRaceHash[h%uint32(len(poolRaceHash))])
}


//固定到某个P，如果私有对象为空则放到私有对象；
//否则加入到该P子池的共享列表中（需要加锁）。
//可以看到一次put操作最少0次加锁，最多1次加锁。
func (p *Pool) Put(x interface{}) {
	if x == nil {
		return
	}
	if race.Enabled {
		if fastrand()%4 == 0 {
			return
		}
		race.ReleaseMerge(poolRaceAddr(x))
		race.Disable()
	}
	l := p.pin()
	if l.private == nil {
		l.private = x
		x = nil
	}
	runtime_procUnpin()
	if x != nil {
		l.Lock()
		l.shared = append(l.shared, x)
		l.Unlock()
	}
	if race.Enabled {
		race.Enable()
	}
}

//固定到某个P，尝试从私有对象获取，如果私有对象非空则返回该对象，并把私有对象置空；
//如果私有对象是空的时候，就去当前子池的共享列表获取（需要加锁）；
//如果当前子池的共享列表也是空的，那么就尝试去其他P的子池的共享列表偷取一个（需要加锁）；
//如果其他子池都是空的，最后就用用户指定的New函数产生一个新的对象返回。
//可以看到一次get操作最少0次加锁，最大N（N等于MAXPROCS）次加锁。
func (p *Pool) Get() interface{} {
	if race.Enabled {
		race.Disable()
	}
	l := p.pin()
	x := l.private
	l.private = nil
	runtime_procUnpin()
	if x == nil {
		l.Lock()
		last := len(l.shared) - 1
		if last >= 0 {
			x = l.shared[last]
			l.shared = l.shared[:last]
		}
		l.Unlock()
		if x == nil {
			x = p.getSlow()
		}
	}
	if race.Enabled {
		race.Enable()
		if x != nil {
			race.Acquire(poolRaceAddr(x))
		}
	}
	if x == nil && p.New != nil {
		x = p.New()
	}
	return x
}

func (p *Pool) getSlow() (x interface{}) {
	size := atomic.LoadUintptr(&p.localSize)
	local := p.local

	pid := runtime_procPin()
	runtime_procUnpin()
	for i := 0; i < int(size); i++ {
		l := indexLocal(local, (pid+i+1)%int(size))
		l.Lock()
		last := len(l.shared) - 1
		if last >= 0 {
			x = l.shared[last]
			l.shared = l.shared[:last]
			l.Unlock()
			break
		}
		l.Unlock()
	}
	return x
}

func (p *Pool) pin() *poolLocal {
	pid := runtime_procPin()

	s := atomic.LoadUintptr(&p.localSize)
	l := p.local
	if uintptr(pid) < s {
		return indexLocal(l, pid)
	}
	return p.pinSlow()
}

func (p *Pool) pinSlow() *poolLocal {
	runtime_procUnpin()
	allPoolsMu.Lock()
	defer allPoolsMu.Unlock()
	pid := runtime_procPin()

	s := p.localSize
	l := p.local
	if uintptr(pid) < s {
		return indexLocal(l, pid)
	}
	if p.local == nil {
		allPools = append(allPools, p)
	}

	size := runtime.GOMAXPROCS(0)
	local := make([]poolLocal, size)
	atomic.StorePointer(&p.local, unsafe.Pointer(&local[0]))
	atomic.StoreUintptr(&p.localSize, uintptr(size))
	return &local[pid]
}

//在系统自动GC的时候，触发pool.go中的poolCleanup函数,把Pool中所有goroutine创建的对象都进行销毁
func poolCleanup() {
	for i, p := range allPools {
		allPools[i] = nil
		for i := 0; i < int(p.localSize); i++ {
			l := indexLocal(p.local, i)
			l.private = nil
			for j := range l.shared {
				l.shared[j] = nil
			}
			l.shared = nil
		}
		p.local = nil
		p.localSize = 0
	}
	allPools = []*Pool{}
}

var (
	allPoolsMu Mutex
	allPools   []*Pool
)

//在init中注册了一个poolCleanup函数，
//它会清除所有的pool里面的所有缓存的对象，该函数注册进去之后会在每次gc之前都会调用
func init() {
	runtime_registerPoolCleanup(poolCleanup)
}

func indexLocal(l unsafe.Pointer, i int) *poolLocal {
	return &(*[1000000]poolLocal)(l)[i]
}

// Implemented in runtime.
func runtime_registerPoolCleanup(cleanup func())
func runtime_procPin() int
func runtime_procUnpin()

```