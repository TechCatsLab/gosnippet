## Signal
### sigtrap.go
```go
package caddy

import (
	"log"
	"os"
	"os/signal"
	"sync"
)

// 初始化信号量处理
func TrapSignals() {
	trapSignalsCrossPlatform()
	trapSignalsPosix()
}

func trapSignalsCrossPlatform() {
	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt)   // 处理 syscall.SIGINT

		for i := 0; true; i++ {
			<-shutdown  // 捕获

            // 非首次捕获信号量，强制退出
			if i > 0 {
				log.Println("[INFO] SIGINT: Force quit")
				if PidFile != "" {
					os.Remove(PidFile)
				}
				os.Exit(1)
			}

            // 首次捕获 SIGINT
			log.Println("[INFO] SIGINT: Shutting down")

			if PidFile != "" {
				os.Remove(PidFile)
			}

            // 执行回调
			go os.Exit(executeShutdownCallbacks("SIGINT"))
		}
	}()
}

func executeShutdownCallbacks(signame string) (exitCode int) {
	shutdownCallbacksOnce.Do(func() {
	    // 执行全部 plugins hook 函数
		EmitEvent(ShutdownEvent)

		errs := allShutdownCallbacks()
		if len(errs) > 0 {
			for _, err := range errs {
				log.Printf("[ERROR] %s shutdown: %v", signame, err)
			}
			exitCode = 1
		}
	})
	return
}

func allShutdownCallbacks() []error {
	var errs []error
	
	// 执行全部实例的关闭回调函数
	instancesMu.Lock()
	for _, inst := range instances {
		errs = append(errs, inst.ShutdownCallbacks()...)
	}
	instancesMu.Unlock()
	return errs
}

// 保证关闭回调函数只执行一次
var shutdownCallbacksOnce sync.Once
```

### POSIX Signals
```go
func trapSignalsPosix() {
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1)   // 关注要处理的信号量

		for sig := range sigchan {
			switch sig {
			case syscall.SIGTERM:   // 强制退出，不做善后
				log.Println("[INFO] SIGTERM: Terminating process")
				if PidFile != "" {
					os.Remove(PidFile)
				}
				os.Exit(0)

			case syscall.SIGQUIT:   // 退出，并等待退出回调执行完毕
				log.Println("[INFO] SIGQUIT: Shutting down")
				exitCode := executeShutdownCallbacks("SIGQUIT")
				err := Stop()
				if err != nil {
					log.Printf("[ERROR] SIGQUIT stop: %v", err)
					exitCode = 1
				}
				if PidFile != "" {
					os.Remove(PidFile)
				}
				os.Exit(exitCode)

			case syscall.SIGHUP:    // 退出，不做善后
				log.Println("[INFO] SIGHUP: Hanging up")
				err := Stop()
				if err != nil {
					log.Printf("[ERROR] SIGHUP stop: %v", err)
				}

			case syscall.SIGUSR1:   // 重载配置文件
				log.Println("[INFO] SIGUSR1: Reloading")

				instancesMu.Lock()
				if len(instances) == 0 {    // 没有运行
					instancesMu.Unlock()
					log.Println("[ERROR] SIGUSR1: No server instances are fully running")
					continue
				}
				inst := instances[0]        // 选择第一个实例
				instancesMu.Unlock()

				updatedCaddyfile := inst.caddyfileInput
				if updatedCaddyfile == nil {
					log.Println("[ERROR] SIGUSR1: no Caddyfile to reload (was stdin left open?)")
					continue
				}
				
				if loaderUsed.loader == nil {
					log.Println("[ERROR] SIGUSR1: no Caddyfile loader with which to reload Caddyfile")
					continue
				}

				newCaddyfile, err := loaderUsed.loader.Load(inst.serverType)
				if err != nil {
					log.Printf("[ERROR] SIGUSR1: loading updated Caddyfile: %v", err)
					continue
				}
				if newCaddyfile != nil {
					updatedCaddyfile = newCaddyfile
				}

                // 重启实例
				inst, err = inst.Restart(updatedCaddyfile)
				if err != nil {
					log.Printf("[ERROR] SIGUSR1: %v", err)
				}
			}
		}
	}()
}
```

