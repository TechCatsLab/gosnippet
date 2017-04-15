## 启动流程
### caddy/main.go
```go
package main

import "github.com/mholt/caddy/caddy/caddymain"

var run = caddymain.Run     // go 中函数也可像变量一样使用

func main() {
	run()
}
```

### caddy/caddyadmin/run.go
```go
func Run() {
	flag.Parse()            // 获取命令行参数                  

	switch logfile {        // 设置日志文件
	    ...
	}

	// 是否使用插件
	if plugins {
		fmt.Println(caddy.DescribePlugins())
		os.Exit(0)
	}

	// 设置 CPU 使用 (频率或核数)
	err := setCPU(cpu)
	if err != nil {
		mustLogFatalf("%v", err)
	}

	// 发送启动事件给 plugins
	caddy.EmitEvent(caddy.StartupEvent)

	// 获取配置文件
	caddyfileinput, err := caddy.LoadCaddyfile(serverType)
	if err != nil {
		mustLogFatalf("%v", err)
	}

    // 验证配置文件
	if validate {
		err := caddy.ValidateAndExecuteDirectives(caddyfileinput, nil, true)
		if err != nil {
			mustLogFatalf("%v", err)
		}
		...
		os.Exit(0)
	}

	// 启动
	instance, err := caddy.Start(caddyfileinput)
	if err != nil {
		mustLogFatalf("%v", err)
	}

	instance.Wait()
}
```

### caddy/caddy.go
```go
type Instance struct {
	// 类型
	serverType string

	// 配置文件
	caddyfileInput Input

	// 用于等待全部 Server 停止
	wg *sync.WaitGroup

	context Context

	servers []ServerListener

	// 回调
	onFirstStartup  []func() error  // 非重启时
	onStartup       []func() error  // 非重启时
	onRestart       []func() error  // 重启前调用
	onShutdown      []func() error  // 无论是否重启过程
	onFinalShutdown []func() error  // 非重启
}

var (
	instances []*Instance

	instancesMu sync.Mutex
)

func Start(cdyfile Input) (*Instance, error) {
	writePidFile()          // 将 PID 写入 pid 文件
	inst := &Instance{serverType: cdyfile.ServerType(), wg: new(sync.WaitGroup)}    // 创建实例
	return inst, startWithListenerFds(cdyfile, inst, nil)
}

func startWithListenerFds(cdyfile Input, inst *Instance, restartFds map[string]restartTriple) error {
    // 配置文件为空，创建新配置结构
	if cdyfile == nil {
		cdyfile = CaddyfileInput{}
	}

    // 读取配置文件
    // 执行配置文件指令，创建 Server，并存入 instance
	err := ValidateAndExecuteDirectives(cdyfile, inst, false)
	if err != nil {
		return err
	}

    // 创建 Server
	slist, err := inst.context.MakeServers()
	if err != nil {
		return err
	}

	// 执行首次启动回调函数
	if restartFds == nil {
		for _, firstStartupFunc := range inst.onFirstStartup {
			err := firstStartupFunc()
			if err != nil {
				return err
			}
		}
	}
	
	// 执行启动回调函数
	for _, startupFunc := range inst.onStartup {
		err := startupFunc()
		if err != nil {
			return err
		}
	}

    // 启动 Server
	err = startServers(slist, inst, restartFds)
	if err != nil {
		return err
	}

	instancesMu.Lock()
	instances = append(instances, inst)
	instancesMu.Unlock()

	// 执行启动完成回调
	if restartFds == nil {
		for _, srvln := range inst.servers {
			if srv, ok := srvln.server.(AfterStartup); ok {
				srv.OnStartupComplete()
			}
		}
		if !Quiet {
			for _, srvln := range inst.servers {
				if !IsLoopback(srvln.listener.Addr().String()) {
					checkFdlimit()
					break
				}
			}
		}
	}

	mu.Lock()
	started = true
	mu.Unlock()

	return nil
}
```
