## HTTP (net/http/server.go)

### 服务端核心结构
```go
// 请求处理接口
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

// HTTP 请求处理函数原型
type HandlerFunc func(ResponseWriter, *Request)

// 实现 Handler 接口
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

type ServeMux struct {
	mu    sync.RWMutex          // 读写锁，保护路由
	m     map[string]muxEntry   // 路由
	hosts bool
}

type muxEntry struct {
	explicit bool
	h        Handler            // 处理函数
	pattern  string             // 路由模式
}

func NewServeMux() *ServeMux { return new(ServeMux) }

// Handler 接口实现
// ServerMux 是 Handler，所以，可以直接传递给 ListenAndServe
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
	
	// 获取路由处理函数
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}

// 添加路由处理
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
    // 转换为 HandlerFunc
	mux.Handle(pattern, HandlerFunc(handler))
}

func (mux *ServeMux) Handle(pattern string, handler Handler) {
    // 写锁保护
	mux.mu.Lock()
	defer mux.mu.Unlock()

    // 参数保护
	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}
	if handler == nil {
		panic("http: nil handler")
	}
	
	// 已经存在该路由
	if mux.m[pattern].explicit {
		panic("http: multiple registrations for " + pattern)
	}

    // 首次添加路由，创建路由表
	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}
	
	// 添加路由
	mux.m[pattern] = muxEntry{explicit: true, h: handler, pattern: pattern}

	if pattern[0] != '/' {
		mux.hosts = true
	}

	n := len(pattern)
	if n > 0 && pattern[n-1] == '/' && !mux.m[pattern[0:n-1]].explicit {
		path := pattern
		if pattern[0] != '/' {
			path = pattern[strings.Index(pattern, "/"):]
		}
		url := &url.URL{Path: path}
		mux.m[pattern[0:n-1]] = muxEntry{h: RedirectHandler(url.String(), StatusMovedPermanently), pattern: pattern}
	}
}

func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	
	// 创建 TCP 连接
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	
	// 处理服务
	return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

func (srv *Server) Serve(l net.Listener) error {
    // 退出时，关闭 Listener
	defer l.Close()
	
	// hook 执行
	if fn := testHookServerServe; fn != nil {
		fn(srv, l)
	}
	
	var tempDelay time.Duration

	if err := srv.setupHTTP2_Serve(); err != nil {
		return err
	}

	srv.trackListener(l, true)
	defer srv.trackListener(l, false)

	baseCtx := context.Background()
	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	ctx = context.WithValue(ctx, LocalAddrContextKey, l.Addr())
	
	// 连接处理
	for {
		rw, e := l.Accept()             // 接受新连接
		if e != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("http: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		c := srv.newConn(rw)
		c.setState(c.rwc, StateNew)
		
		// 每一个连接启动一个 goroutine 处理
		// 要熟悉这种方式！！
		go c.serve(ctx)
	}
}
```

### ListenAndServe
```go
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}
```
