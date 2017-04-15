## 插件

### caddy/plugin.go

```go
package caddy

import (
	"fmt"
	"log"
	"net"
	"sort"

	"github.com/mholt/caddy/caddyfile"
)

var (
	// 注册的 server 类型
	serverTypes = make(map[string]ServerType)

	// 对特定类型 server 注册的插件
	plugins = make(map[string]map[string]Plugin)

	// 钩子函数
	eventHooks = make(map[string]EventHook)

	// 特定类型 server 的指令解析回调
	parsingCallbacks = make(map[string]map[string][]ParsingCallback)

	// 配置文件加载器
	caddyfileLoaders []caddyfileLoader
)

func DescribePlugins() string {
    // 注册的 server 类型
	str := "Server types:\n"
	for name := range serverTypes {
		str += "  " + name + "\n"
	}

	// 配置文件加载器
	str += "\nCaddyfile loaders:\n"
	for _, loader := range caddyfileLoaders {
		str += "  " + loader.name + "\n"
	}
	// 默认加载器
	if defaultCaddyfileLoader.name != "" {
		str += "  " + defaultCaddyfileLoader.name + "\n"
	}

	if len(eventHooks) > 0 {
		// 事件钩子函数
		str += "\nEvent hook plugins:\n"
		for hookPlugin := range eventHooks {
			str += "  hook." + hookPlugin + "\n"
		}
	}

	var others []string
	for stype, stypePlugins := range plugins {
		for name := range stypePlugins {
			var s string
			if stype != "" {
				s = stype + "."
			}
			s += name
			others = append(others, s)
		}
	}

    // 排序
	sort.Strings(others)
	str += "\nOther plugins:\n"
	for _, name := range others {
		str += "  " + name + "\n"
	}

	return str
}

func ValidDirectives(serverType string) []string {
	stype, err := getServerType(serverType)
	if err != nil {
		return nil
	}
	
	// 返回特定类型的 server 支持的指令
	return stype.Directives()
}

type ServerListener struct {
	server   Server
	listener net.Listener
	packet   net.PacketConn
}

// UDP 地址
func (s ServerListener) LocalAddr() net.Addr {
	if s.packet == nil {
		return nil
	}
	return s.packet.LocalAddr()
}

// TCP 地址
func (s ServerListener) Addr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// server 启动过程控制接口
type Context interface {
	// server 指令执行前
	InspectServerBlocks(string, []caddyfile.ServerBlock) ([]caddyfile.ServerBlock, error)

	// 创建服务器实例
	MakeServers() ([]Server, error)
}

// 注册服务类型
func RegisterServerType(typeName string, srv ServerType) {
	if _, ok := serverTypes[typeName]; ok {
		panic("server type already registered")
	}
	serverTypes[typeName] = srv
}

// 核心数据结构
type ServerType struct {
	// 支持的指令
	Directives func() []string

	// 默认配置
	DefaultInput func() Input

	// 创建新 Context
	NewContext func() Context
}

// 插件
type Plugin struct {
	// 关联的服务类型
	ServerType string

	// 插件配置函数
	Action SetupFunc
}

// 注册插件
func RegisterPlugin(name string, plugin Plugin) {
    // 必须有插件名称
	if name == "" {
		panic("plugin must have a name")
	}
	
	// 第一个插件
	if _, ok := plugins[plugin.ServerType]; !ok {
		plugins[plugin.ServerType] = make(map[string]Plugin)
	}
	
	// 插件已存在
	if _, dup := plugins[plugin.ServerType][name]; dup {
		panic("plugin named " + name + " already registered for server type " + plugin.ServerType)
	}
	plugins[plugin.ServerType][name] = plugin
}

// 事件名称
type EventName string

const (
	StartupEvent  EventName = "startup"     // 启动事件
	ShutdownEvent EventName = "shutdown"    // 关闭事件
)

// 事件 Hook 函数类型定义
type EventHook func(eventType EventName) error

func RegisterEventHook(name string, hook EventHook) {
    // 必须有名称
	if name == "" {
		panic("event hook must have a name")
	}
	
	// 不能重复注册
	if _, dup := eventHooks[name]; dup {
		panic("hook named " + name + " already registered")
	}
	eventHooks[name] = hook
}


func EmitEvent(event EventName) {
    // 遍历 eventHooks
	for name, hook := range eventHooks {
		err := hook(event)      // 执行 Hook

		if err != nil {
			log.Printf("error on '%s' hook: %v", name, err)
		}
	}
}

type ParsingCallback func(Context) error


func RegisterParsingCallback(serverType, afterDir string, callback ParsingCallback) {
	if _, ok := parsingCallbacks[serverType]; !ok {
		parsingCallbacks[serverType] = make(map[string][]ParsingCallback)
	}
	parsingCallbacks[serverType][afterDir] = append(parsingCallbacks[serverType][afterDir], callback)
}

type SetupFunc func(c *Controller) error

func DirectiveAction(serverType, dir string) (SetupFunc, error) {
	if stypePlugins, ok := plugins[serverType]; ok {
		if plugin, ok := stypePlugins[dir]; ok {
			return plugin.Action, nil
		}
	}
	if genericPlugins, ok := plugins[""]; ok {
		if plugin, ok := genericPlugins[dir]; ok {
			return plugin.Action, nil
		}
	}
	return nil, fmt.Errorf("no action found for directive '%s' with server type '%s' (missing a plugin?)",
		dir, serverType)
}

type Loader interface {
	Load(serverType string) (Input, error)
}

type LoaderFunc func(serverType string) (Input, error)

func (lf LoaderFunc) Load(serverType string) (Input, error) {
	return lf(serverType)
}

func RegisterCaddyfileLoader(name string, loader Loader) {
	caddyfileLoaders = append(caddyfileLoaders, caddyfileLoader{name: name, loader: loader})
}

func SetDefaultCaddyfileLoader(name string, loader Loader) {
	defaultCaddyfileLoader = caddyfileLoader{name: name, loader: loader}
}

func loadCaddyfileInput(serverType string) (Input, error) {
	var loadedBy string
	var caddyfileToUse Input
	for _, l := range caddyfileLoaders {
		cdyfile, err := l.loader.Load(serverType)
		if err != nil {
			return nil, fmt.Errorf("loading Caddyfile via %s: %v", l.name, err)
		}
		if cdyfile != nil {
			if caddyfileToUse != nil {
				return nil, fmt.Errorf("Caddyfile loaded multiple times; first by %s, then by %s", loadedBy, l.name)
			}
			loaderUsed = l
			caddyfileToUse = cdyfile
			loadedBy = l.name
		}
	}
	if caddyfileToUse == nil && defaultCaddyfileLoader.loader != nil {
		cdyfile, err := defaultCaddyfileLoader.loader.Load(serverType)
		if err != nil {
			return nil, err
		}
		if cdyfile != nil {
			loaderUsed = defaultCaddyfileLoader
			caddyfileToUse = cdyfile
		}
	}
	return caddyfileToUse, nil
}

type caddyfileLoader struct {
	name   string
	loader Loader
}

var (
	defaultCaddyfileLoader caddyfileLoader
	loaderUsed             caddyfileLoader
)

```

### Server Interface
```go
type TCPServer interface {
	Listen() (net.Listener, error)

	Serve(net.Listener) error
}

type UDPServer interface {
	ListenPacket() (net.PacketConn, error)

	ServePacket(net.PacketConn) error
}

type Server interface {
	TCPServer
	UDPServer
}
```
