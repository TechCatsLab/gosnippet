## HTTP Server 核心结构

### Server 类型
```go
type Server struct {
	Server      *http.Server            // HTTP Server
	quicServer  *h2quic.Server          // QUIC Server
	listener    net.Listener
	listenerMu  sync.Mutex
	sites       []*SiteConfig           // 站点配置
	connTimeout time.Duration
	tlsGovChan  chan struct{} goroutine
	vhosts      *vhostTrie
}

// 确保 Server 为 GracefulServer，否则编译错误
var _ caddy.GracefulServer = new(Server)

// 站点配置
type SiteConfig struct {
	Addr Address

	ListenHost string

	TLS *caddytls.Config

	middleware []Middleware

	middlewareChain Handler

	listenerMiddleware []ListenerMiddleware

	Root string

	HiddenFiles []string

	MaxRequestBodySizes []PathLimit

	Timeouts Timeouts
}

type (
    // 传入 Handler 并返回 Handler，链式连接 Handler
	Middleware func(Handler) Handler

	ListenerMiddleware func(caddy.Listener) caddy.Listener

	Handler interface {
		ServeHTTP(http.ResponseWriter, *http.Request) (int, error)
	}

	HandlerFunc func(http.ResponseWriter, *http.Request) (int, error)
)

// HandlerFunc 实现 Handler 接口
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	return f(w, r)
}

```

### 工具函数
```go
// 创建
func NewServer(addr string, group []*SiteConfig) (*Server, error) {
	s := &Server{
		Server:      makeHTTPServerWithTimeouts(addr, group),
		vhosts:      newVHostTrie(),
		sites:       group,
		connTimeout: GracefulTimeout,
	}
	
	// Server 满足 Handler 接口
	s.Server.Handler = s

    // 获取 TLS 配置
	tlsConfig, err := makeTLSConfig(group)
	if err != nil {
		return nil, err
	}
	s.Server.TLSConfig = tlsConfig

	// 使用 QUIC
	if QUIC {
		s.quicServer = &h2quic.Server{Server: s.Server}
		s.Server.Handler = s.wrapWithSvcHeaders(s.Server.Handler)
	}

	// TLS 开启
	if s.Server.TLSConfig != nil {
		tlsh := &tlsHandler{next: s.Server.Handler}
		s.Server.Handler = tlsh

		// 设置连接状态改变回调
		s.Server.ConnState = func(c net.Conn, cs http.ConnState) {
			if tlsh.listener != nil {
				if cs == http.StateHijacked || cs == http.StateClosed {
					tlsh.listener.helloInfosMu.Lock()
					delete(tlsh.listener.helloInfos, c.RemoteAddr().String())
					tlsh.listener.helloInfosMu.Unlock()
				}
			}
		}

		if HTTP2 && len(s.Server.TLSConfig.NextProtos) == 0 {
			// some experimenting shows that this NextProtos must have at least
			// one value that overlaps with the NextProtos of any other tls.Config
			// that is returned from GetConfigForClient; if there is no overlap,
			// the connection will fail (as of Go 1.8, Feb. 2017).
			s.Server.TLSConfig.NextProtos = defaultALPN
		}
	}

    // 初始化每个站点使用的中间件
	for _, site := range group {
	    // 第一个中间件为文件处理
		stack := Handler(staticfiles.FileServer{Root: http.Dir(site.Root), Hide: site.HiddenFiles})
		for i := len(site.middleware) - 1; i >= 0; i-- {
		    // 拼接中间件处理函数
			stack = site.middleware[i](stack)
		}
		
		// 设置最终的中间件处理
		site.middlewareChain = stack
		s.vhosts.Insert(site.Addr.VHost(), site)
	}

	return s, nil
}
```
