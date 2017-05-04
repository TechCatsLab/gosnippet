## Agent
### 结构
```go
type Agent struct {
	config *Config

	logger *log.Logger

	logOutput io.Writer

	logWriter *logger.LogWriter

	server *consul.Server
	client *consul.Client

	acls *aclManager

	state localState

	checkReapAfter map[types.CheckID]time.Duration

	checkMonitors map[types.CheckID]*CheckMonitor

	checkHTTPs map[types.CheckID]*CheckHTTP

	checkTCPs map[types.CheckID]*CheckTCP

	checkTTLs map[types.CheckID]*CheckTTL

	checkDockers map[types.CheckID]*CheckDocker

	checkLock sync.Mutex

	eventCh chan serf.UserEvent

	eventBuf    []*UserEvent
	eventIndex  int
	eventLock   sync.RWMutex
	eventNotify state.NotifyGroup

	reloadCh chan chan error

	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	endpoints map[string]string
}
```
