# echo

使用 echo 实现 websocket 的两种方式

## 使用 `net` 库的 websocket

服务端

`server.go`

```go

package main

import (
    "fmt"
    "log"

    "github.com/labstack/echo"
    "github.com/labstack/echo/middleware"
    "golang.org/x/net/websocket"
)

func hello(c echo.Context) error {
    websocket.Handler(func(ws *websocket.Conn) {
        defer ws.Close()
        for {
            // 写消息
            err := websocket.Message.Send(ws, "Hello, Client!")
            if err != nil {
                log.Fatal(err)
            }

            // 读取消息并打印
            msg := ""
            err = websocket.Message.Receive(ws, &msg)
            if err != nil {
                log.Fatal(err)
            }
            fmt.Printf("%s\n", msg)
        }
        // ServerHTTP 会把 http 升级成 websocket 连接
    }).ServeHTTP(c.Response(), c.Request())
    return nil
}

func main() {
    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Static("/", "../public")
    e.GET("/ws", hello)
    e.Logger.Fatal(e.Start(":1323"))
}
```

## 使用 `gorilla` 的 websocket

服务端

`server.go`

```go
package main

import (
    "fmt"
    "log"

    "github.com/labstack/echo"

    "github.com/gorilla/websocket"
    "github.com/labstack/echo/middleware"
)

var (
    upgrader = websocket.Upgrader{}
)

func hello(c echo.Context) error {
    // 升级成 websocket 连接
    ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
    if err != nil {
        return err
    }
    defer ws.Close()

    for {
        // 写消息
        err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
        if err != nil {
            log.Fatal(err)
        }

        // 读消息并打印
        _, msg, err := ws.ReadMessage()
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("%s\n", msg)
    }
}

func main() {
    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Static("/", "../public")
    e.GET("/ws", hello)
    e.Logger.Fatal(e.Start(":1323"))
}
```

客户端

``` html
<!doctype html>
<html lang="en">

<head>
  <meta charset="utf-8">
  <title>WebSocket</title>
</head>

<body>
  <p id="output"></p>

  <script>
    var loc = window.location;
    // 拼凑连接
    var uri = 'ws:';

    if (loc.protocol === 'https:') {
      uri = 'wss:';
    }
    uri += '//' + loc.host;
    uri += loc.pathname + 'ws';

    // 连接 websocket
    ws = new WebSocket(uri)

    ws.onopen = function() {
      console.log('Connected')
    }

    // 读取消息并打印
    ws.onmessage = function(evt) {
      var out = document.getElementById('output');
      out.innerHTML += evt.data + '<br>';
    }

    // 每一秒发一次消息
    setInterval(function() {
      ws.send('Hello, Server!');
    }, 1000);
  </script>
</body>

</html>
```

最后结果：

`client`

```html
Hello, Client!
Hello, Client!
Hello, Client!
Hello, Client!
Hello, Client!
```

`server`

```html
Hello, Server!
Hello, Server!
Hello, Server!
Hello, Server!
Hello, Server!
```
