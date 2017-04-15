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
