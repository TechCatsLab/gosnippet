## 官方示例
### Once
```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var once sync.Once
	onceBody := func() {
		fmt.Println("Only once")
	}
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			once.Do(onceBody)
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
```

### Pool
```go
package main

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		// The Pool's New function should generally only return pointer
		// types, since a pointer can be put into the return interface
		// value without an allocation:
		return new(bytes.Buffer)
	},
}

// timeNow is a fake version of time.Now for tests.
func timeNow() time.Time {
	return time.Unix(1136214245, 0)
}

func Log(w io.Writer, key, val string) {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	// Replace this with time.Now() in a real logger.
	b.WriteString(timeNow().UTC().Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(val)
	w.Write(b.Bytes())
	bufPool.Put(b)
}

func main() {
	Log(os.Stdout, "path", "/search?q=flowers")
}
```

### WaitGroup
```go
var wg sync.WaitGroup
    var urls = []string{
            "http://www.golang.org/",
            "http://www.google.com/",
            "http://www.somestupidname.com/",
    }
    for _, url := range urls {
            // Increment the WaitGroup counter.
            wg.Add(1)
            // Launch a goroutine to fetch the URL.
            go func(url string) {
                    // Decrement the counter when the goroutine completes.
                    defer wg.Done()
                    // Fetch the URL.
                    http.Get(url)
            }(url)
    }
    // Wait for all HTTP fetches to complete.
    wg.Wait()
```
