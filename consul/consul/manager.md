## Server Manager
### 核心结构
```go
type ManagerSerfCluster interface {
	NumNodes() int
}

type Pinger interface {
	PingConsulServer(s *agent.Server) (bool, error)
}

type serverList struct {
	servers []*agent.Server
}

type Manager struct {
	listValue atomic.Value      // 存储服务器链表
	listLock  sync.Mutex

	rebalanceTimer *time.Timer

	shutdownCh chan struct{}

	logger *log.Logger

	clusterInfo ManagerSerfCluster

	connPoolPinger Pinger

	notifyFailedBarrier int32

	offline int32
}
```

### 方法
#### New
```go
func New(logger *log.Logger, shutdownCh chan struct{}, clusterInfo ManagerSerfCluster, connPoolPinger Pinger) (m *Manager) {
	m = new(Manager)
	m.logger = logger
	m.clusterInfo = clusterInfo
	m.connPoolPinger = connPoolPinger
	m.rebalanceTimer = time.NewTimer(clientRPCMinReuseDuration)
	m.shutdownCh = shutdownCh
	
	// 初始设置为 1
	atomic.StoreInt32(&m.offline, 1)

	l := serverList{}
	l.servers = make([]*agent.Server, 0)
	m.saveServerList(l)
	return m
}
```

#### AddServer
```go
func (m *Manager) AddServer(s *agent.Server) {
	m.listLock.Lock()
	defer m.listLock.Unlock()
	l := m.getServerList()

	found := false
	for idx, existing := range l.servers {
	    // 服务器已存在
		if existing.Name == s.Name {
			newServers := make([]*agent.Server, len(l.servers))
			copy(newServers, l.servers)

			// 更新信息
			newServers[idx] = s

			l.servers = newServers
			found = true
			break
		}
	}

	// 新服务器注册
	if !found {
		newServers := make([]*agent.Server, len(l.servers), len(l.servers)+1)
		copy(newServers, l.servers)
		newServers = append(newServers, s)
		l.servers = newServers
	}

	// 添加新服务器后，设置 offline 为 0 (false)
	atomic.StoreInt32(&m.offline, 0)

    // 保存至 manager
	m.saveServerList(l)
}

// 注意 atomic.Value 使用方式
func (m *Manager) getServerList() serverList {
	return m.listValue.Load().(serverList)
}

func (m *Manager) saveServerList(l serverList) {
	m.listValue.Store(l)
}
```

#### IsOffline
```go
func (m *Manager) IsOffline() bool {
	offline := atomic.LoadInt32(&m.offline)
	return offline == 1
}
```

#### FindServer
```go
func (m *Manager) FindServer() *agent.Server {
	l := m.getServerList()
	numServers := len(l.servers)
	if numServers == 0 {
		m.logger.Printf("[WARN] manager: No servers available")
		return nil
	} else {
		// 返回第一个
		// 因为有 shuffleServers 存在，所以没问题
		return l.servers[0]
	}
}

// 随机调整服务器顺序
func (l *serverList) shuffleServers() {
	for i := len(l.servers) - 1; i > 0; i-- {
		j := rand.Int31n(int32(i + 1))
		l.servers[i], l.servers[j] = l.servers[j], l.servers[i]
	}
}
```

#### NotifyFailedServer
```go
func (m *Manager) NotifyFailedServer(s *agent.Server) {
	l := m.getServerList()

	// 服务器多余 1 个
	// 目前使用的是链表首服务器
	// barrier 为 0
	if len(l.servers) > 1 && l.servers[0] == s &&
		atomic.CompareAndSwapInt32(&m.notifyFailedBarrier, 0, 1) {
		// 释放 barrier
		defer atomic.StoreInt32(&m.notifyFailedBarrier, 0)

		m.listLock.Lock()
		defer m.listLock.Unlock()
		l = m.getServerList()

        // 需要二次验证
		if len(l.servers) > 1 && l.servers[0] == s {
			l.servers = l.cycleServer() // 将链表头移至表尾
			m.saveServerList(l)
		}
	}
}
```

#### RebalanceServers
```go
func (m *Manager) RebalanceServers() {
	l := m.getServerList()

    // 调整服务器顺序，更容易获取新服务器
	l.shuffleServers()

	var foundHealthyServer bool
	for i := 0; i < len(l.servers); i++ {
		selectedServer := l.servers[0]  // 使用表头服务器

        // 验证服务器是否健康
		ok, err := m.connPoolPinger.PingConsulServer(selectedServer)
		if ok {
			foundHealthyServer = true
			break
		}
		m.logger.Printf(`[DEBUG] manager: pinging server "%s" failed: %s`, selectedServer.String(), err)

        // 当前服务器不健康，移至表尾
		l.cycleServer()
	}

	// 设置服务器状态
	if foundHealthyServer {
		atomic.StoreInt32(&m.offline, 0)
	} else {
		atomic.StoreInt32(&m.offline, 1)
		m.logger.Printf("[DEBUG] manager: No healthy servers during rebalance, aborting")
		return
	}

	if m.reconcileServerList(&l) {
		m.logger.Printf("[DEBUG] manager: Rebalanced %d servers, next active server is %s", len(l.servers), l.servers[0].String())
	} else {
	}

	return
}
```

#### Start
```go
func (m *Manager) Start() {
	for {
		select {
		case <-m.rebalanceTimer.C:              // 定时选择服务器
			m.RebalanceServers()
			m.refreshServerRebalanceTimer()     // 重置 Timer

		case <-m.shutdownCh:                    // Consul Server 关闭通知
			m.logger.Printf("[INFO] manager: shutting down")
			return
		}
	}
}
```
