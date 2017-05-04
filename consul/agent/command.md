## Command

### base.Command
```go
type FlagSetFlags uint

const (
	FlagSetNone       FlagSetFlags = 1 << iota
	FlagSetClientHTTP FlagSetFlags = 1 << iota
	FlagSetServerHTTP FlagSetFlags = 1 << iota

	FlagSetHTTP = FlagSetClientHTTP | FlagSetServerHTTP
)

type Command struct {
	Ui    cli.Ui
	Flags FlagSetFlags

	flagSet *flag.FlagSet
	hidden  *flag.FlagSet

	// HTTP API 设置
	httpAddr      StringValue
	token         StringValue
	caFile        StringValue
	caPath        StringValue
	certFile      StringValue
	keyFile       StringValue
	tlsServerName StringValue

	datacenter StringValue
	stale      BoolValue
}
```

### agent.Command
```go
type Command struct {
	base.Command
	Revision          string
	Version           string
	VersionPrerelease string
	HumanVersion      string
	ShutdownCh        <-chan struct{}
	configReloadCh    chan chan error
	args              []string
	logFilter         *logutils.LevelFilter
	logOutput         io.Writer
	agent             *Agent
	httpServers       []*HTTPServer
	dnsServer         *DNSServer
	scadaProvider     *scada.Provider
	scadaHttp         *HTTPServer
}

func (c *Command) Run(args []string) int {
    // 控制台交互设置
	c.Ui = &cli.PrefixedUi{
		OutputPrefix: "==> ",
		InfoPrefix:   "    ",
		ErrorPrefix:  "==> ",
		Ui:           c.Ui,
	}

	// 读取配置
	c.args = args
	config := c.readConfig()
	if config == nil {
		return 1
	}

	// 日志配置
	logConfig := &logger.Config{
		LogLevel:       config.LogLevel,
		EnableSyslog:   config.EnableSyslog,
		SyslogFacility: config.SyslogFacility,
	}
	logFilter, logGate, logWriter, logOutput, ok := logger.Setup(logConfig, c.Ui)
	if !ok {
		return 1
	}
	c.logFilter = logFilter
	c.logOutput = logOutput

	// 创建重新加载配置文件 chan
	c.configReloadCh = make(chan chan error)

	// 设置计量相关
	inm := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics.DefaultInmemSignal(inm)
	metricsConf := metrics.DefaultConfig(config.Telemetry.StatsitePrefix)
	metricsConf.EnableHostname = !config.Telemetry.DisableHostname

	var fanout metrics.FanoutSink
	if config.Telemetry.StatsiteAddr != "" {
		sink, err := metrics.NewStatsiteSink(config.Telemetry.StatsiteAddr)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start statsite sink. Got: %s", err))
			return 1
		}
		fanout = append(fanout, sink)
	}

	if config.Telemetry.StatsdAddr != "" {
		sink, err := metrics.NewStatsdSink(config.Telemetry.StatsdAddr)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start statsd sink. Got: %s", err))
			return 1
		}
		fanout = append(fanout, sink)
	}

	if config.Telemetry.DogStatsdAddr != "" {
		var tags []string

		if config.Telemetry.DogStatsdTags != nil {
			tags = config.Telemetry.DogStatsdTags
		}

		sink, err := datadog.NewDogStatsdSink(config.Telemetry.DogStatsdAddr, metricsConf.HostName)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start DogStatsd sink. Got: %s", err))
			return 1
		}
		sink.SetTags(tags)
		fanout = append(fanout, sink)
	}

	if config.Telemetry.CirconusAPIToken != "" || config.Telemetry.CirconusCheckSubmissionURL != "" {
		cfg := &circonus.Config{}
		cfg.Interval = config.Telemetry.CirconusSubmissionInterval
		cfg.CheckManager.API.TokenKey = config.Telemetry.CirconusAPIToken
		cfg.CheckManager.API.TokenApp = config.Telemetry.CirconusAPIApp
		cfg.CheckManager.API.URL = config.Telemetry.CirconusAPIURL
		cfg.CheckManager.Check.SubmissionURL = config.Telemetry.CirconusCheckSubmissionURL
		cfg.CheckManager.Check.ID = config.Telemetry.CirconusCheckID
		cfg.CheckManager.Check.ForceMetricActivation = config.Telemetry.CirconusCheckForceMetricActivation
		cfg.CheckManager.Check.InstanceID = config.Telemetry.CirconusCheckInstanceID
		cfg.CheckManager.Check.SearchTag = config.Telemetry.CirconusCheckSearchTag
		cfg.CheckManager.Check.DisplayName = config.Telemetry.CirconusCheckDisplayName
		cfg.CheckManager.Check.Tags = config.Telemetry.CirconusCheckTags
		cfg.CheckManager.Broker.ID = config.Telemetry.CirconusBrokerID
		cfg.CheckManager.Broker.SelectTag = config.Telemetry.CirconusBrokerSelectTag

		if cfg.CheckManager.Check.DisplayName == "" {
			cfg.CheckManager.Check.DisplayName = "Consul"
		}

		if cfg.CheckManager.API.TokenApp == "" {
			cfg.CheckManager.API.TokenApp = "consul"
		}

		if cfg.CheckManager.Check.SearchTag == "" {
			cfg.CheckManager.Check.SearchTag = "service:consul"
		}

		sink, err := circonus.NewCirconusSink(cfg)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start Circonus sink. Got: %s", err))
			return 1
		}
		sink.Start()
		fanout = append(fanout, sink)
	}

	if len(fanout) > 0 {
		fanout = append(fanout, inm)
		metrics.NewGlobal(metricsConf, fanout)
	} else {
		metricsConf.EnableHostname = false
		metrics.NewGlobal(metricsConf, inm)
	}

	// 创建 agent
	if err := c.setupAgent(config, logOutput, logWriter); err != nil {
		return 1
	}
	
	// 设置善后
	defer c.agent.Shutdown()
	if c.dnsServer != nil {
		defer c.dnsServer.Shutdown()
	}
	for _, server := range c.httpServers {
		defer server.Shutdown()
	}

	defer func() {
		if c.scadaHttp != nil {
			c.scadaHttp.Shutdown()
		}
		if c.scadaProvider != nil {
			c.scadaProvider.Shutdown()
		}
	}()

	// 加入 startup 节点簇
	if err := c.startupJoin(config); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if err := c.startupJoinWan(config); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var httpAddr net.Addr
	var err error
	if config.Ports.HTTP != -1 {
		httpAddr, err = config.ClientListener(config.Addresses.HTTP, config.Ports.HTTP)
	} else if config.Ports.HTTPS != -1 {
		httpAddr, err = config.ClientListener(config.Addresses.HTTPS, config.Ports.HTTPS)
	} else if len(config.WatchPlans) > 0 {
		c.Ui.Error("Error: cannot use watches if both HTTP and HTTPS are disabled")
		return 1
	}
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to determine HTTP address: %v", err))
	}

	// 注册 watches
	for _, wp := range config.WatchPlans {
		go func(wp *watch.WatchPlan) {
			wp.Handler = makeWatchHandler(logOutput, wp.Exempt["handler"])
			wp.LogOutput = c.logOutput
			addr := httpAddr.String()
			
			// unix 地址处理
			if httpAddr.Network() == "unix" {
				addr = "unix://" + addr
			}
			
			// 启动
			if err := wp.Run(addr); err != nil {
				c.Ui.Error(fmt.Sprintf("Error running watch: %v", err))
			}
		}(wp)
	}

	// gossip 协议相关
	var gossipEncrypted bool
	if config.Server {
		gossipEncrypted = c.agent.server.Encrypted()
	} else {
		gossipEncrypted = c.agent.client.Encrypted()
	}

	atlas := "<disabled>"
	if config.AtlasInfrastructure != "" {
		atlas = fmt.Sprintf("(Infrastructure: '%s' Join: %v)", config.AtlasInfrastructure, config.AtlasJoin)
	}

	// 启动 agent
	c.agent.StartSync()

    // 输出运行状态信息
	c.Ui.Output("Consul agent running!")
	c.Ui.Info(fmt.Sprintf("       Version: '%s'", c.HumanVersion))
	c.Ui.Info(fmt.Sprintf("       Node ID: '%s'", config.NodeID))
	c.Ui.Info(fmt.Sprintf("     Node name: '%s'", config.NodeName))
	c.Ui.Info(fmt.Sprintf("    Datacenter: '%s'", config.Datacenter))
	c.Ui.Info(fmt.Sprintf("        Server: %v (bootstrap: %v)", config.Server, config.Bootstrap))
	c.Ui.Info(fmt.Sprintf("   Client Addr: %v (HTTP: %d, HTTPS: %d, DNS: %d)", config.ClientAddr,
		config.Ports.HTTP, config.Ports.HTTPS, config.Ports.DNS))
	c.Ui.Info(fmt.Sprintf("  Cluster Addr: %v (LAN: %d, WAN: %d)", config.AdvertiseAddr,
		config.Ports.SerfLan, config.Ports.SerfWan))
	c.Ui.Info(fmt.Sprintf("Gossip encrypt: %v, RPC-TLS: %v, TLS-Incoming: %v",
		gossipEncrypted, config.VerifyOutgoing, config.VerifyIncoming))
	c.Ui.Info(fmt.Sprintf("         Atlas: %s", atlas))

	c.Ui.Info("")
	c.Ui.Output("Log data will now stream in as it occurs:\n")
	logGate.Flush()

	// 启动 retry join 协程
	errCh := make(chan struct{})
	go c.retryJoin(config, errCh)

	// 启动 retry -wan join 携程
	errWanCh := make(chan struct{})
	go c.retryJoinWan(config, errWanCh)

	// 等待退出
	return c.handleSignals(config, errCh, errWanCh)
}
```
