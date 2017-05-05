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
```
