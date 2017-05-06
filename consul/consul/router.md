## Router
### 核心结构
```go
type RouterSerfCluster interface {
	NumNodes() int
	Members() []serf.Member
	GetCoordinate() (*coordinate.Coordinate, error)
	GetCachedCoordinate(name string) (*coordinate.Coordinate, bool)
}

type managerInfo struct {
	manager *Manager
	shutdownCh chan struct{}            // 关闭通知 chan
}

type areaInfo struct {
	cluster RouterSerfCluster
	pinger Pinger
	managers map[string]*managerInfo    // DC -> managerInfo
}

type Router struct {
	logger *log.Logger
	localDatacenter string              // 本地 DC
	areas map[types.AreaID]*areaInfo    // DC 映射
	managers map[string][]*Manager      // DC 管理者映射
	routeFn func(datacenter string) (*Manager, *agent.Server, bool) // 路由 hook
	sync.RWMutex
}
```

### 方法
#### NewRouter
```go
func NewRouter(logger *log.Logger, shutdownCh chan struct{}, localDatacenter string) *Router {
	router := &Router{
		logger:          logger,
		localDatacenter: localDatacenter,
		areas:           make(map[types.AreaID]*areaInfo),
		managers:        make(map[string][]*Manager),
	}

	// 默认使用直接路由方式
	router.routeFn = router.findDirectRoute

	go func() {
		<-shutdownCh            // 等待系统关闭
		router.Lock()
		defer router.Unlock()

		for _, area := range router.areas {
			for _, info := range area.managers {
				close(info.shutdownCh)  // 关闭通知
			}
		}

        // 释放内存，否则可造成内存泄漏
        // 使用带 GC 编程语言时，一定要注意这类写法
		router.areas = nil
		router.managers = nil
	}()

	return router
}
```

#### AddArea
```go
func (r *Router) AddArea(areaID types.AreaID, cluster RouterSerfCluster, pinger Pinger) error {
	r.Lock()
	defer r.Unlock()

    // 已存在，报错
	if _, ok := r.areas[areaID]; ok {
		return fmt.Errorf("area ID %q already exists", areaID)
	}

    // 记录新区域信息
	area := &areaInfo{
		cluster:  cluster,
		pinger:   pinger,
		managers: make(map[string]*managerInfo),
	}
	r.areas[areaID] = area

	for _, m := range cluster.Members() {
		ok, parts := agent.IsConsulServer(m)
		if !ok {
			r.logger.Printf("[WARN]: consul: Non-server %q in server-only area %q",
				m.Name, areaID)
			continue
		}

        // 如果是 consul server，添加
		if err := r.addServer(area, parts); err != nil {
			return fmt.Errorf("failed to add server %q to area %q: %v", m.Name, areaID, err)
		}
	}

	return nil
}

func (r *Router) addServer(area *areaInfo, s *agent.Server) error {
	info, ok := area.managers[s.Datacenter]
	
	// 首次添加 DC，创建 Manager
	if !ok {
		shutdownCh := make(chan struct{})
		manager := New(r.logger, shutdownCh, area.cluster, area.pinger)
		info = &managerInfo{
			manager:    manager,
			shutdownCh: shutdownCh,
		}
		area.managers[s.Datacenter] = info

		managers := r.managers[s.Datacenter]
		r.managers[s.Datacenter] = append(managers, manager)
		go manager.Start()
	}

	info.manager.AddServer(s)
	return nil
}
```

#### RemoveArea
```go
func (r *Router) RemoveArea(areaID types.AreaID) error {
	r.Lock()
	defer r.Unlock()

	area, ok := r.areas[areaID]
	if !ok {
		return fmt.Errorf("area ID %q does not exist", areaID)
	}

	// 移除全部 manager
	for datacenter, info := range area.managers {
		r.removeManagerFromIndex(datacenter, info.manager)
		close(info.shutdownCh)
	}

	delete(r.areas, areaID)
	return nil
}

func (r *Router) removeManagerFromIndex(datacenter string, manager *Manager) {
	managers := r.managers[datacenter]
	for i := 0; i < len(managers); i++ {
	    // 移除 manager
		if managers[i] == manager {
		    // slice 操作
			r.managers[datacenter] = append(managers[:i], managers[i+1:]...)
			
			// 数据中心已无管理者，移除该映射
			if len(r.managers[datacenter]) == 0 {
				delete(r.managers, datacenter)
			}
			return
		}
	}
	
	// 没有找到，报错
	panic("managers index out of sync")
}
```

#### AddServer
```go
func (r *Router) AddServer(areaID types.AreaID, s *agent.Server) error {
	r.Lock()
	defer r.Unlock()

	area, ok := r.areas[areaID]
	if !ok {
		return fmt.Errorf("area ID %q does not exist", areaID)
	}
	return r.addServer(area, s)
}
```

#### RemoveServer
```go
func (r *Router) RemoveServer(areaID types.AreaID, s *agent.Server) error {
	r.Lock()
	defer r.Unlock()

	area, ok := r.areas[areaID]
	if !ok {
		return fmt.Errorf("area ID %q does not exist", areaID)
	}

    // 数据中心已不存在
	info, ok := area.managers[s.Datacenter]
	if !ok {
		return nil
	}
	
	// 移除服务器
	info.manager.RemoveServer(s)

    // manager 已不包含任何服务器，则从 router 中移除
	if num := info.manager.NumServers(); num == 0 {
		r.removeManagerFromIndex(s.Datacenter, info.manager)
		close(info.shutdownCh)              // 服务关闭通知
		delete(area.managers, s.Datacenter) // 移除该数据中心
	}

	return nil
}
```

#### FailServer
```go
func (r *Router) FailServer(areaID types.AreaID, s *agent.Server) error {
	r.RLock()
	defer r.RUnlock()

	area, ok := r.areas[areaID]
	if !ok {
		return fmt.Errorf("area ID %q does not exist", areaID)
	}

	info, ok := area.managers[s.Datacenter]
	if !ok {
		return nil
	}

	info.manager.NotifyFailedServer(s)
	return nil
}
```

#### FindRoute
```go
func (r *Router) FindRoute(datacenter string) (*Manager, *agent.Server, bool) {
	return r.routeFn(datacenter)
}

func (r *Router) findDirectRoute(datacenter string) (*Manager, *agent.Server, bool) {
	r.RLock()
	defer r.RUnlock()

	managers, ok := r.managers[datacenter]
	if !ok {
		return nil, nil, false
	}

	for _, manager := range managers {
		if manager.IsOffline() {
			continue
		}

		if s := manager.FindServer(); s != nil {
			return manager, s, true
		}
	}

	return nil, nil, false
}
```

#### GetDatacenters
```go
func (r *Router) GetDatacenters() []string {
	r.RLock()
	defer r.RUnlock()

	dcs := make([]string, 0, len(r.managers))
	for dc, _ := range r.managers {
		dcs = append(dcs, dc)
	}

	sort.Strings(dcs)
	return dcs
}
```
