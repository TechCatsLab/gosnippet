## Server
### 核心结构
```go
type Server struct {
	aclAuthCache *acl.Cache                             // Auth 缓存
	aclCache *aclCache                                  // Non-Auth 缓存

	autopilotPolicy AutopilotPolicy                     // 策略
	autopilotRemoveDeadCh chan struct{}                 // 移除宕机服务器策略
	autopilotShutdownCh chan struct{}                   // 停止 chan
	autopilotWaitGroup sync.WaitGroup                   // 关闭等待组

	clusterHealth     structs.OperatorHealthReply       // 集群健康检测
	clusterHealthLock sync.RWMutex

	config *Config                                      // 配置

	connPool *ConnPool                                  // 到其他 Server 的连接池

	endpoints endpoints                                 // RPC

	eventChLAN chan serf.Event                          // 同一数据中心
	eventChWAN chan serf.Event                          // 跨数据中心

	fsm *consulFSM                                      // Raft 状态机

	localConsuls map[raft.ServerAddress]*agent.Server
	localLock    sync.RWMutex

	logger *log.Logger

	raft          *raft.Raft
	raftLayer     *RaftLayer
	raftStore     *raftboltdb.BoltStore
	raftTransport *raft.NetworkTransport
	raftInmem     *raft.InmemStore

	leaderCh <-chan bool                                // Leader 选举 chan

	reconcileCh chan serf.Member

	router *servers.Router

	rpcListener net.Listener                            // 接受 RPC 连接
	rpcServer   *rpc.Server

	rpcTLS *tls.Config                                  // TLS 连接配置

	serfLAN *serf.Serf
	serfWAN *serf.Serf

	floodLock sync.RWMutex
	floodCh   []chan struct{}

	sessionTimers     map[string]*time.Timer
	sessionTimersLock sync.Mutex

	statsFetcher *StatsFetcher                          // 检测相关

	tombstoneGC *state.TombstoneGC                      // GC 检测

	aclReplicationStatus     structs.ACLReplicationStatus
	aclReplicationStatusLock sync.RWMutex

	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
}

// RPC
type endpoints struct {
	ACL           *ACL
	Catalog       *Catalog
	Coordinate    *Coordinate
	Health        *Health
	Internal      *Internal
	KVS           *KVS
	Operator      *Operator
	PreparedQuery *PreparedQuery
	Session       *Session
	Status        *Status
	Txn           *Txn
}
```

### 方法
#### NewServer
```go
func NewServer(config *Config) (*Server, error) {
	// 检查协议版本号
	if err := config.CheckVersion(); err != nil {
		return nil, err
	}

	// 数据目录检查
	if config.DataDir == "" && !config.DevMode {
		return nil, fmt.Errorf("Config must provide a DataDir")
	}

	// ACL 基本检查
	if err := config.CheckACL(); err != nil {
		return nil, err
	}

	// 日志
	if config.LogOutput == nil {
		config.LogOutput = os.Stderr
	}
	logger := log.New(config.LogOutput, "", log.LstdFlags)

	// TLS outgoing 配置
	tlsConf := config.tlsConfig()
	tlsWrap, err := tlsConf.OutgoingTLSWrapper()
	if err != nil {
		return nil, err
	}

	// TLS incoming 配置
	incomingTLS, err := tlsConf.IncomingTLSConfig()
	if err != nil {
		return nil, err
	}

	// GC
	gc, err := state.NewTombstoneGC(config.TombstoneTTL, config.TombstoneTTLGranularity)
	if err != nil {
		return nil, err
	}

	// 创建关闭 chan
	shutdownCh := make(chan struct{})

	// 创建 Server
	s := &Server{
		autopilotRemoveDeadCh: make(chan struct{}),
		autopilotShutdownCh:   make(chan struct{}),
		config:                config,
		connPool:              NewPool(config.LogOutput, serverRPCCache, serverMaxStreams, tlsWrap),
		eventChLAN:            make(chan serf.Event, 256),
		eventChWAN:            make(chan serf.Event, 256),
		localConsuls:          make(map[raft.ServerAddress]*agent.Server),
		logger:                logger,
		reconcileCh:           make(chan serf.Member, 32),
		router:                servers.NewRouter(logger, shutdownCh, config.Datacenter),
		rpcServer:             rpc.NewServer(),
		rpcTLS:                incomingTLS,
		tombstoneGC:           gc,
		shutdownCh:            make(chan struct{}),
	}

	s.autopilotPolicy = &BasicAutopilot{server: s}

	s.statsFetcher = NewStatsFetcher(logger, s.connPool, s.config.Datacenter)

	s.aclAuthCache, err = acl.NewCache(aclCacheSize, s.aclLocalFault)
	if err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("Failed to create authoritative ACL cache: %v", err)
	}

	var local acl.FaultFunc
	if s.IsACLReplicationEnabled() {
		local = s.aclLocalFault
	}
	if s.aclCache, err = newAclCache(config, logger, s.RPC, local); err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("Failed to create non-authoritative ACL cache: %v", err)
	}

	// RPC 初始化
	if err := s.setupRPC(tlsWrap); err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("Failed to start RPC layer: %v", err)
	}

	// Raft 初始化
	if err := s.setupRaft(); err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("Failed to start Raft: %v", err)
	}

	// LAN Serf
	s.serfLAN, err = s.setupSerf(config.SerfLANConfig,
		s.eventChLAN, serfLANSnapshot, false)
	if err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("Failed to start LAN Serf: %v", err)
	}
	go s.lanEventHandler()

	// WAN Serf
	s.serfWAN, err = s.setupSerf(config.SerfWANConfig,
		s.eventChWAN, serfWANSnapshot, true)
	if err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("Failed to start WAN Serf: %v", err)
	}

	// 路由设置
	if err := s.router.AddArea(types.AreaWAN, s.serfWAN, s.connPool); err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("Failed to add WAN serf route: %v", err)
	}
	go servers.HandleSerfEvents(s.logger, s.router, types.AreaWAN, s.serfWAN.ShutdownCh(), s.eventChWAN)

	portFn := func(s *agent.Server) (int, bool) {
		if s.WanJoinPort > 0 {
			return s.WanJoinPort, true
		} else {
			return 0, false
		}
	}
	go s.Flood(portFn, s.serfWAN)

	// leadership 监控
	go s.monitorLeadership()

	// ACL 副本
	if s.IsACLReplicationEnabled() {
		go s.runACLReplication()
	}

	// 监听 RPC 请求
	go s.listen()

	// Metrics 监控
	go s.sessionStats()

	// 服务器监控
	go s.serverHealthLoop()

	return s, nil
}
```
