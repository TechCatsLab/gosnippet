## Signal
```go
func (c *Command) handleSignals(config *Config, retryJoin <-chan struct{}, retryJoinWan <-chan struct{}) int {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGPIPE)

	// 等待信号
WAIT:
	var sig os.Signal
	var reloadErrCh chan error
	select {
	case s := <-signalCh:           // 读取到信号量
		sig = s
	case ch := <-c.configReloadCh:  // 需要重新加载配置文件
		sig = syscall.SIGHUP        // 模拟信号
		reloadErrCh = ch
	case <-c.ShutdownCh:            // 关闭
		sig = os.Interrupt
	case <-retryJoin:               // retryJoin 错误，直接退出执行
		return 1
	case <-retryJoinWan:
		return 1
	case <-c.agent.ShutdownCh():    // 正常关闭，退出执行
		return 0
	}

	// 忽略该信号量
	if sig == syscall.SIGPIPE {
		goto WAIT
	}

	c.Ui.Output(fmt.Sprintf("Caught signal: %v", sig))

	// 重新加载配置文件
	if sig == syscall.SIGHUP {
		conf, err := c.handleReload(config)
		if conf != nil {
			config = conf
		}
		if err != nil {
			c.Ui.Error(err.Error())
		}
		
		if reloadErrCh != nil {
			reloadErrCh <- err
		}
		goto WAIT
	}

	graceful := false
	if sig == os.Interrupt && !(*config.SkipLeaveOnInt) {
		graceful = true
	} else if sig == syscall.SIGTERM && (*config.LeaveOnTerm) {
		graceful = true
	}

    // 直接退出，是很不优雅...
	if !graceful {
		return 1
	}

	// 尝试优雅退出
	gracefulCh := make(chan struct{})
	c.Ui.Output("Gracefully shutting down agent...")
	go func() {
		if err := c.agent.Leave(); err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err))
			return
		}
		close(gracefulCh)
	}()

	select {
	case <-signalCh:
		return 1
	case <-time.After(gracefulTimeout):     // 超时，还未退出，强退；注意此类的保护
		return 1
	case <-gracefulCh:
		return 0
	}
}
```
