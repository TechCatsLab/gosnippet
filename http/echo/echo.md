## echo.go

### echo.New

```go
// New creates an instance of Echo.
func New() (e *Echo) {
  // 创建一个 Echo 实例
	e = &Echo{
		Server:    new(http.Server),	// 创建一个 http Server
		TLSServer: new(http.Server),	// 创建一个 http Server
		AutoTLSManager: autocert.Manager{	// TLS 自动认证管理器
			Prompt: autocert.AcceptTOS,
		},
		Logger:   log.New("echo"),	// 创建一个 labstack/gommon/log 的实例
		colorer:  color.New(),	// 给打印的文字添加颜色用的
      maxParam: new(int),		// 设置每个请求 URL 中带“:”的参数的最大值
	}
	e.Server.Handler = e
	e.TLSServer.Handler = e
	e.HTTPErrorHandler = e.DefaultHTTPErrorHandler	// 设置默认错误处理
	e.Binder = &DefaultBinder{}	// 设置默认 Binder 
	e.Logger.SetLevel(log.OFF)	// 设置默认不开启 log
	e.stdLogger = stdLog.New(e.Logger.Output(), e.Logger.Prefix()+": ", 0)
	e.pool.New = func() interface{} {	// 初始化Context 对象池的 New 函数
		return e.NewContext(nil, nil)
	}
	e.router = NewRouter(e)	// 初始化 Router
	return
}
```

### echo.NewContext

```go
// NewContext returns a Context instance.
func (e *Echo) NewContext(r *http.Request, w http.ResponseWriter) Context {
	return &context{
		request:  r,	// http 的 request
		response: NewResponse(w, e),	// echo 封装了 response，并且反指 echo
		store:    make(Map),	// 初始化一个 map[string]interface{} 用于存储
		echo:     e,	// 反指 echo
		pvalues:  make([]string, *e.maxParam),	// 用于存储每次请求URL 中带“:”的参数
		handler:  NotFoundHandler,	// 设置默认处理函数 NotFoundHandler
	}
}
```

### echo.DefaultHTTPErrorHandler

```go
// DefaultHTTPErrorHandler is the default HTTP error handler. It sends a JSON response
// with status code.
func (e *Echo) DefaultHTTPErrorHandler(err error, c Context) {
	var (
		code = http.StatusInternalServerError	// code = 500
		msg  interface{}
	)
	
	if he, ok := err.(*HTTPError); ok {		// 断言是不是 HTTPError，是的话赋值
		code = he.Code
		msg = he.Message
	} else if e.Debug {		// 不是的话判断是否 Debug 模式，是的话把错误信息给 msg
		msg = err.Error()
	} else {	// 再不是的话把 http response code 对应的描述给 msg
		msg = http.StatusText(code)
	}
	if _, ok := msg.(string); ok {	// 类型断言，如果可以转成 string 的话，包装成 map
		msg = Map{"message": msg}
	}

	if !c.Response().Committed {	// 判断是否已经写了 response 返回头信息
		if c.Request().Method == HEAD { // 判断请求方法是否是 HEAD
			if err := c.NoContent(code); err != nil {	// NoContent 只写返回头信息
				goto ERROR
			}
		} else {
			if err := c.JSON(code, msg); err != nil {	// 返回 json 格式返回值
				goto ERROR
			}
		}
	}
ERROR:
	e.Logger.Error(err)	// 有错误的话，打印错误栈
}
```

### echo.Pre

```go
// Pre adds middleware to the chain which is run before router.
func (e *Echo) Pre(middleware ...MiddlewareFunc) {
  	// 在 premiddleware 切片往后加路由匹配之前的 middleware
	e.premiddleware = append(e.premiddleware, middleware...)	
}
```

### echo.Use

```go
// Use adds middleware to the chain which is run after router.
func (e *Echo) Use(middleware ...MiddlewareFunc) {
  	// 在 middleware 切片往后加路由匹配之后的 middleware
	e.middleware = append(e.middleware, middleware...)
}
```

### echo.Static

```go
// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (e *Echo) Static(prefix, root string) {
	if root == "" {
		root = "." // For security we want to restrict to CWD.
	}
	static(e, prefix, root)
}

func static(i i, prefix, root string) {
  	// 返回 (root + * 对应的参数) 的路径的文件
	h := func(c Context) error {
		name := filepath.Join(root, path.Clean("/"+c.Param("*"))) // "/"+ for security
		return c.File(name)
	}
	i.GET(prefix, h)	// 添加一个 prefix 路由
	if prefix == "/" {
		i.GET(prefix+"*", h) // 添加一个 /* 的路由
	} else {
		i.GET(prefix+"/*", h)	// 添加一个 prefix + /* 的路由
	}
}

func (c *context) File(file string) (err error) {
	file, err = url.QueryUnescape(file) // 把 字符串中 %AB 转换成 0xAB, '+'、 转换成 ' '等
	if err != nil {
		return
	}

	f, err := os.Open(file)
	if err != nil {
		return ErrNotFound
	}
	defer f.Close()

	fi, _ := f.Stat() // 获取文件 info 结构
	if fi.IsDir() {	// 如果是目录，则在 file 字符串后面加上 index.html
		file = filepath.Join(file, indexPage)	// indexPage = index.html
		f, err = os.Open(file)
		if err != nil {
			return ErrNotFound
		}
		defer f.Close()
		if fi, err = f.Stat(); err != nil {
			return
		}
	}
  	// 返回文件名、修改时间、字节流
	http.ServeContent(c.Response(), c.Request(), fi.Name(), fi.ModTime(), f) 
	return
}
```

