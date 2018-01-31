## HTTP 服务插件

### Server 类型注册
```go
// caddy/caddyhttp/httpserver/plugin.go
const serverType = "http"

func init() {
    // 获取参数部分
	flag.StringVar(&HTTPPort, "http-port", HTTPPort, "Default port to use for HTTP")
	...

    // 注册服务
	caddy.RegisterServerType(serverType, caddy.ServerType{
		Directives: func() []string { return directives },
		DefaultInput: func() caddy.Input {
			if Port == DefaultPort && Host != "" {
				return caddy.CaddyfileInput{
					Contents:       []byte(fmt.Sprintf("%s\nroot %s", Host, Root)),
					ServerTypeName: serverType,
				}
			}
			return caddy.CaddyfileInput{
				Contents:       []byte(fmt.Sprintf("%s:%s\nroot %s", Host, Port, Root)),
				ServerTypeName: serverType,
			}
		},
		NewContext: newContext,
	})
	// 注册配置文件加载器
	caddy.RegisterCaddyfileLoader("short", caddy.LoaderFunc(shortCaddyfileLoader))
	// 配置回调
	caddy.RegisterParsingCallback(serverType, "root", hideCaddyfile)
	caddy.RegisterParsingCallback(serverType, "tls", activateHTTPS)
	caddytls.RegisterConfigGetter(serverType, func(c *caddy.Controller) *caddytls.Config { return GetConfig(c).TLS })
}
```

### httpContext
```go
func newContext() caddy.Context {
	return &httpContext{keysToSiteConfigs: make(map[string]*SiteConfig)}
}

type httpContext struct {
	keysToSiteConfigs map[string]*SiteConfig

	siteConfigs []*SiteConfig
}

// 实现 Context 接口
func (h *httpContext) InspectServerBlocks(sourceFile string, serverBlocks []caddyfile.ServerBlock) ([]caddyfile.ServerBlock, error) {
	for _, sb := range serverBlocks {
		for _, key := range sb.Keys {
		    // 不允许重复地址
			key = strings.ToLower(key)
			if _, dup := h.keysToSiteConfigs[key]; dup {
				return serverBlocks, fmt.Errorf("duplicate site address: %s", key)
			}
			addr, err := standardizeAddress(key)
			if err != nil {
				return serverBlocks, err
			}

			// 是否要使用命令行传入的参数
			if addr.Host == "" && Host != DefaultHost {
				addr.Host = Host
			}
			if addr.Port == "" && Port != DefaultPort {
				addr.Port = Port
			}

			// 自定义了 HTTP,HTTPS 端口，记录端口，提供给 ACME 使用
			var altHTTPPort, altTLSSNIPort string
			if HTTPPort != DefaultHTTPPort {
				altHTTPPort = HTTPPort
			}
			if HTTPSPort != DefaultHTTPSPort {
				altTLSSNIPort = HTTPSPort
			}

			// 新建站点配置文件，并保存
			cfg := &SiteConfig{
				Addr: addr,
				Root: Root,
				TLS: &caddytls.Config{
					Hostname:      addr.Host,
					AltHTTPPort:   altHTTPPort,
					AltTLSSNIPort: altTLSSNIPort,
				},
				originCaddyfile: sourceFile,
			}
			h.saveConfig(key, cfg)
		}
	}

	for _, sb := range serverBlocks {
		_, hasGzip := sb.Tokens["gzip"]
		_, hasErrors := sb.Tokens["errors"]
		if hasGzip && !hasErrors {
			sb.Tokens["errors"] = []caddyfile.Token{{Text: "errors"}}
		}
	}

	return serverBlocks, nil
}

func (h *httpContext) MakeServers() ([]caddy.Server, error) {
    // 调整配置参数
	for _, cfg := range h.siteConfigs {
	    // 明确关闭 TLS 的站点配置，不做处理
		if !cfg.TLS.Enabled {
			continue
		}
		
		// HTTPPort 与 Port 相同，或制定了 http，关闭 TLS
		if cfg.Addr.Port == HTTPPort || cfg.Addr.Scheme == "http" {
			cfg.TLS.Enabled = false
			log.Printf("[WARNING] TLS disabled for %s", cfg.Addr)
		} else if cfg.Addr.Scheme == "" {
			// Port 与 HTTPPort 不相同
			cfg.Addr.Scheme = "https"
		}
		
		if cfg.Addr.Port == "" && ((!cfg.TLS.Manual && !cfg.TLS.SelfSigned) || cfg.TLS.OnDemand) {
			cfg.Addr.Port = HTTPSPort
		}
	}

	groups, err := groupSiteConfigsByListenAddr(h.siteConfigs)
	if err != nil {
		return nil, err
	}

	var servers []caddy.Server
	for addr, group := range groups {
		s, err := NewServer(addr, group)
		if err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}

	return servers, nil
}

// 自定义方法
func (h *httpContext) saveConfig(key string, cfg *SiteConfig) {
	h.siteConfigs = append(h.siteConfigs, cfg)
	h.keysToSiteConfigs[key] = cfg
}
```

### 辅助函数及结构
```go
func groupSiteConfigsByListenAddr(configs []*SiteConfig) (map[string][]*SiteConfig, error) {
    // address -> []*SiteConfig
	groups := make(map[string][]*SiteConfig)

	for _, conf := range configs {
		// 是否使用命令行传入端口
		if conf.Addr.Port == "" {
			conf.Addr.Port = Port
		}
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(conf.ListenHost, conf.Addr.Port))
		if err != nil {
			return nil, err
		}
		addrstr := addr.String()
		
		// 根据监听地址，组织配置文件
		groups[addrstr] = append(groups[addrstr], conf)
	}

	return groups, nil
}
```
