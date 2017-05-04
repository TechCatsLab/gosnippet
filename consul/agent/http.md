## HTTP

### 结构
```go
type HTTPServer struct {
	agent    *Agent             // 拥有者 Agent 指针
	mux      *http.ServeMux     // Mux
	listener net.Listener       // Listener
	logger   *log.Logger
	uiDir    string
	addr     string
}
```

### 方法
```go
func NewHTTPServers(agent *Agent, config *Config, logOutput io.Writer) ([]*HTTPServer, error) {
	if logOutput == nil {
		return nil, fmt.Errorf("Please provide a valid logOutput(io.Writer)")
	}

	var servers []*HTTPServer

	if config.Ports.HTTPS > 0 {
	    // 读取需要的配置内容
		httpAddr, err := config.ClientListener(config.Addresses.HTTPS, config.Ports.HTTPS)
		if err != nil {
			return nil, err
		}

		tlsConf := &tlsutil.Config{
			VerifyIncoming: config.VerifyIncoming,
			VerifyOutgoing: config.VerifyOutgoing,
			CAFile:         config.CAFile,
			CertFile:       config.CertFile,
			KeyFile:        config.KeyFile,
			NodeName:       config.NodeName,
			ServerName:     config.ServerName,
			TLSMinVersion:  config.TLSMinVersion,
		}

        // TLS 配置
		tlsConfig, err := tlsConf.IncomingTLSConfig()
		if err != nil {
			return nil, err
		}

        // 监听地址
		ln, err := net.Listen(httpAddr.Network(), httpAddr.String())
		if err != nil {
			return nil, fmt.Errorf("Failed to get Listen on %s: %v", httpAddr.String(), err)
		}

		list := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, tlsConfig)

		// MUX
		mux := http.NewServeMux()

		// Server
		srv := &HTTPServer{
			agent:    agent,
			mux:      mux,
			listener: list,
			logger:   log.New(logOutput, "", log.LstdFlags),
			uiDir:    config.UiDir,
			addr:     httpAddr.String(),
		}
		
		// 注册 MUX
		srv.registerHandlers(config.EnableDebug)

		// 启动
		go http.Serve(list, mux)
		servers = append(servers, srv)
	}

	if config.Ports.HTTP > 0 {
		httpAddr, err := config.ClientListener(config.Addresses.HTTP, config.Ports.HTTP)
		if err != nil {
			return nil, fmt.Errorf("Failed to get ClientListener address:port: %v", err)
		}

		// 防止监听同一个 Unix Socket
		socketPath, isSocket := unixSocketAddr(config.Addresses.HTTP)
		if isSocket {
			if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
				agent.logger.Printf("[WARN] agent: Replacing socket %q", socketPath)
			}
			if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("error removing socket file: %s", err)
			}
		}

		ln, err := net.Listen(httpAddr.Network(), httpAddr.String())
		if err != nil {
			return nil, fmt.Errorf("Failed to get Listen on %s: %v", httpAddr.String(), err)
		}

		var list net.Listener
		if isSocket {
			if err := setFilePermissions(socketPath, config.UnixSockets); err != nil {
				return nil, fmt.Errorf("Failed setting up HTTP socket: %s", err)
			}
			list = ln
		} else {
			list = tcpKeepAliveListener{ln.(*net.TCPListener)}
		}

		mux := http.NewServeMux()

		srv := &HTTPServer{
			agent:    agent,
			mux:      mux,
			listener: list,
			logger:   log.New(logOutput, "", log.LstdFlags),
			uiDir:    config.UiDir,
			addr:     httpAddr.String(),
		}
		srv.registerHandlers(config.EnableDebug)

		go http.Serve(list, mux)
		servers = append(servers, srv)
	}

	return servers, nil
}

// 关闭
func (s *HTTPServer) Shutdown() {
	if s != nil {
		s.logger.Printf("[DEBUG] http: Shutting down http server (%v)", s.addr)
		s.listener.Close()
	}
}

// wrap 方法
// 注意这种使用技巧！
func (s *HTTPServer) wrap(handler func(resp http.ResponseWriter, req *http.Request) (interface{}, error)) func(resp http.ResponseWriter, req *http.Request) {
	f := func(resp http.ResponseWriter, req *http.Request) {
		setHeaders(resp, s.agent.config.HTTPAPIResponseHeaders)
		setTranslateAddr(resp, s.agent.config.TranslateWanAddrs)

		formVals, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			s.logger.Printf("[ERR] http: Failed to decode query: %s from=%s", err, req.RemoteAddr)
			resp.WriteHeader(http.StatusInternalServerError) // 500
			return
		}
		logURL := req.URL.String()
		if tokens, ok := formVals["token"]; ok {
			for _, token := range tokens {
				if token == "" {
					logURL += "<hidden>"
					continue
				}
				logURL = strings.Replace(logURL, token, "<hidden>", -1)
			}
		}

		start := time.Now()
		defer func() {
			s.logger.Printf("[DEBUG] http: Request %s %v (%v) from=%s", req.Method, logURL, time.Now().Sub(start), req.RemoteAddr)
		}()
		obj, err := handler(resp, req)

	HAS_ERR:
		if err != nil {
			s.logger.Printf("[ERR] http: Request %s %v, error: %v from=%s", req.Method, logURL, err, req.RemoteAddr)
			code := http.StatusInternalServerError
			errMsg := err.Error()
			if strings.Contains(errMsg, "Permission denied") || strings.Contains(errMsg, "ACL not found") {
				code = http.StatusForbidden
			}

			resp.WriteHeader(code)
			resp.Write([]byte(err.Error()))
			return
		}

		if obj != nil {
			var buf []byte
			buf, err = s.marshalJSON(req, obj)
			if err != nil {
				goto HAS_ERR
			}

			resp.Header().Set("Content-Type", "application/json")
			resp.Write(buf)
		}
	}
	return f
}
```
