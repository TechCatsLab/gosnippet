# 中间件

中间件是一个函数，嵌入在HTTP 的请求和响应之间。它可以获得 Echo#Context 对象用来进行一些特殊的操作， 比如记录每个请求或者统计请求数。

Action的处理在所有的中间件运行完成之后。

## 编写自定义的中间件

如以下需求：

- 通过该中间件去统计请求数目、状态和时间 .
- 中间件自定义返回的 Response .

### server

`server.go`

```go
package main

import (
	"net/http"
	"strconv"
	"sync"
	"time"

  "github.com/labstack/echo"
)

type (
	Stats struct {
		Uptime       time.Time      `json:"uptime"`
		RequestCount uint64         `json:"requestCount"`
		Statuses     map[string]int `json:"statuses"`
		mutex        sync.RWMutex
	}
)

func NewStats() *Stats {
	return &Stats{
		Uptime:   time.Now(),
		Statuses: map[string]int{},
	}
}

// Process is the middleware function.
func (s *Stats) Process(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := next(c); err != nil {
			c.Error(err)
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.RequestCount++
		status := strconv.Itoa(c.Response().Status)
		s.Statuses[status]++
		return nil
	}
}

// Handle is the endpoint to get stats.
func (s *Stats) Handle(c echo.Context) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return c.JSON(http.StatusOK, s)
}

// ServerHeader middleware adds a `Server` header to the response.
func ServerHeader(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderServer, "Echo/3.0")
		return next(c)
	}
}

func main() {
	e := echo.New()

	// Debug mode
	e.Debug = true

	//-------------------
	// Custom middleware
	//-------------------
	// Stats
	s := NewStats()
	e.Use(s.Process)
	e.GET("/stats", s.Handle) // Endpoint to get stats

	// Server header
	e.Use(ServerHeader)

	// Handler
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
```

### 响应

Header

``` html
Content-Length:122
Content-Type:application/json; charset=utf-8
Date:Thu, 14 Apr 2016 20:31:46 GMT
Server:Echo/3.0
```

Body

```json
{
  "uptime": "2016-04-14T13:28:48.486548936-07:00",
  "requestCount": 5,
  "statuses": {
    "200": 4,
    "404": 1
  }
}
```

## 检查登录中间件

需求：

- 在请求某个路由的时候检查该用户的登录状态，若非登录状态则直接返回 Response，不调用对应的函数 .

`handler`

```go

func MustLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// check session
		sess := utility.GlobalSessions.SessionStart(c.Response().Writer, c.Request())
		id := sess.Get(general.SessionUserID)

		if id == nil {
			return c.JSON(errcode.ErrLoginRequired, "User Must Login.")
		}

		return next(c)
	}
}
```

`router.go`

```go
server.GET("/api/v1/user/getInfo", handler.GetInfo, handler.MustLogin)
```