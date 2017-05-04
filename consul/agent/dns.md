## DNS Server

### 结构
```go
type DNSServer struct {
	agent        *Agent
	config       *DNSConfig
	dnsHandler   *dns.ServeMux
	dnsServer    *dns.Server
	dnsServerTCP *dns.Server
	domain       string
	recursors    []string
	logger       *log.Logger
}
```

### 方法
#### NewDNSServer
```go
func NewDNSServer(agent *Agent, config *DNSConfig, logOutput io.Writer, domain string, bind string, recursors []string) (*DNSServer, error) {
	domain = dns.Fqdn(strings.ToLower(domain))

	// 构建 DNS Mux
	mux := dns.NewServeMux()

    // 等待
	var wg sync.WaitGroup

	// 初始化服务器
	// TCP && UDP
	server := &dns.Server{
		Addr:              bind,
		Net:               "udp",
		Handler:           mux,
		UDPSize:           65535,
		NotifyStartedFunc: wg.Done,     // 变量方法传入，注意这种使用方式；下同
	}
	serverTCP := &dns.Server{
		Addr:              bind,
		Net:               "tcp",
		Handler:           mux,
		NotifyStartedFunc: wg.Done,
	}

	// 构建 DNS Server
	srv := &DNSServer{
		agent:        agent,
		config:       config,
		dnsHandler:   mux,
		dnsServer:    server,
		dnsServerTCP: serverTCP,
		domain:       domain,
		recursors:    recursors,
		logger:       log.New(logOutput, "", log.LstdFlags),
	}

	// 注册方法
	mux.HandleFunc("arpa.", srv.handlePtr)
	mux.HandleFunc(domain, srv.handleQuery)
	if len(recursors) > 0 {
		validatedRecursors := make([]string, len(recursors))

		for idx, recursor := range recursors {
			recursor, err := recursorAddr(recursor)
			if err != nil {
				return nil, fmt.Errorf("Invalid recursor address: %v", err)
			}
			validatedRecursors[idx] = recursor
		}

		srv.recursors = validatedRecursors
		mux.HandleFunc(".", srv.handleRecurse)
	}

	wg.Add(2)           // 等待计数置为 2 (TCP && UDP)

	// 异步启动服务
	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			srv.logger.Printf("[ERR] dns: error starting udp server: %v", err)
			errCh <- fmt.Errorf("dns udp setup failed: %v", err)
		}
	}()

	errChTCP := make(chan error, 1)
	go func() {
		if err := serverTCP.ListenAndServe(); err != nil {
			srv.logger.Printf("[ERR] dns: error starting tcp server: %v", err)
			errChTCP <- fmt.Errorf("dns tcp setup failed: %v", err)
		}
	}()

	// 异步等待服务启动结果
	startCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(startCh)
	}()

	// 统一等待结果
	select {
	case e := <-errCh:                  // UDP Server 错误
		return srv, e
	case e := <-errChTCP:               // TCP Server 错误
		return srv, e
	case <-startCh:                     // 启动成功
		return srv, nil
	case <-time.After(time.Second):     // 超时
		return srv, fmt.Errorf("timeout setting up DNS server")
	}
}
```

#### NewDNSServer
```go
func (d *DNSServer) Shutdown() {
    // 关闭 UDP 服务器
	if err := d.dnsServer.Shutdown(); err != nil {
		d.logger.Printf("[ERR] dns: error stopping udp server: %v", err)
	}
	
	// 关闭 TCP 服务器
	if err := d.dnsServerTCP.Shutdown(); err != nil {
		d.logger.Printf("[ERR] dns: error stopping tcp server: %v", err)
	}
}
```
