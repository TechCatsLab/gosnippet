## 控制器
```go
// 配置控制器
type Controller struct {
	caddyfile.Dispenser

	// 针对的实例
	instance *Instance

	// server block 关键字
	Key string

	// server block 只执行一次的功能函数
	OncePerServerBlock func(f func() error) error

	ServerBlockIndex int

	ServerBlockKeyIndex int

	ServerBlockKeys []string

	ServerBlockStorage interface{}
}

func (c *Controller) ServerType() string {
	return c.instance.serverType
}

func (c *Controller) OnFirstStartup(fn func() error) {
    // 注意赋值，append 有可能造成底层存储空间变化
	c.instance.onFirstStartup = append(c.instance.onFirstStartup, fn)
}

func (c *Controller) OnStartup(fn func() error) {
	c.instance.onStartup = append(c.instance.onStartup, fn)
}

func (c *Controller) OnRestart(fn func() error) {
	c.instance.onRestart = append(c.instance.onRestart, fn)
}

func (c *Controller) OnShutdown(fn func() error) {
	c.instance.onShutdown = append(c.instance.onShutdown, fn)
}

func (c *Controller) OnFinalShutdown(fn func() error) {
	c.instance.onFinalShutdown = append(c.instance.onFinalShutdown, fn)
}

func (c *Controller) Context() Context {
	return c.instance.context
}
```
