## 连接池
### muxSession 接口
```go
type muxSession interface {
	Open() (net.Conn, error)
	Close() error
}
```

### StreamClient
```go
// 封装 RPC Client 流
type StreamClient struct {
	stream net.Conn
	codec  rpc.ClientCodec
}

// 关闭流方法
func (sc *StreamClient) Close() {
	sc.stream.Close()
	sc.codec.Close()
}
```

### Conn
```go
type Conn struct {
	refCount    int32       // 引用计数
	shouldClose int32

	addr     net.Addr       // 网络地址
	session  muxSession     // Session
	lastUsed time.Time      // 最后使用时间
	version  int            // 版本

	pool *ConnPool          // 连接池引用

	clients    *list.List   // 客户端列表
	clientLock sync.Mutex
}

func (c *Conn) Close() error {
	return c.session.Close()
}

func (c *Conn) getClient() (*StreamClient, error) {
	c.clientLock.Lock()
	front := c.clients.Front()
	if front != nil {
		c.clients.Remove(front)
	}
	c.clientLock.Unlock()
	
	// 有可用 RPC 连接
	if front != nil {
		return front.Value.(*StreamClient), nil
	}

	// 没有可用，创建新 Session
	stream, err := c.session.Open()
	if err != nil {
		return nil, err
	}

	// 创建 RPC codec
	codec := msgpackrpc.NewClientCodec(stream)

	// 返回 client
	sc := &StreamClient{
		stream: stream,
		codec:  codec,
	}
	return sc, nil
}

func (c *Conn) returnClient(client *StreamClient) {
	didSave := false
	c.clientLock.Lock()
	// 还可以保存连接
	// 未关闭
	if c.clients.Len() < c.pool.maxStreams && atomic.LoadInt32(&c.shouldClose) == 0 {
		c.clients.PushFront(client)
		didSave = true

		// yamux 流，回收
		if ys, ok := client.stream.(*yamux.Stream); ok {
			ys.Shrink()
		}
	}
	c.clientLock.Unlock()
	
	// 没有保存，则关闭
	if !didSave {
		client.Close()
	}
}

func (c *Conn) markForUse() {
	c.lastUsed = time.Now()
	atomic.AddInt32(&c.refCount, 1)
}

```

### ConnPool
```go
type ConnPool struct {
	sync.Mutex
	logOutput io.Writer                 // 日志
	maxTime time.Duration               // 连接保留最长时间
	maxStreams int                      // 最大连接数量
	pool map[string]*Conn               // 连接映射
	limiter map[string]chan struct{}    // 连接控流器
	tlsWrap tlsutil.DCWrapper

	shutdown   bool                     // 关闭控制
	shutdownCh chan struct{}
}

func NewPool(logOutput io.Writer, maxTime time.Duration, maxStreams int, tlsWrap tlsutil.DCWrapper) *ConnPool {
	pool := &ConnPool{
		logOutput:  logOutput,
		maxTime:    maxTime,
		maxStreams: maxStreams,
		pool:       make(map[string]*Conn),
		limiter:    make(map[string]chan struct{}),
		tlsWrap:    tlsWrap,
		shutdownCh: make(chan struct{}),
	}
	
	// 典型 Go 用法，内部启动服务
	if maxTime > 0 {
		go pool.reap()
	}
	return pool
}

func (p *ConnPool) Shutdown() error {
	p.Lock()
	defer p.Unlock()

    // 关闭现有连接
	for _, conn := range p.pool {
		conn.Close()
	}
	// 丢弃之前的映射
	p.pool = make(map[string]*Conn)

	if p.shutdown {
		return nil
	}
	
	// 发送关闭信号
	p.shutdown = true
	close(p.shutdownCh)
	return nil
}

func (p *ConnPool) acquire(dc string, addr net.Addr, version int) (*Conn, error) {
	addrStr := addr.String()

    // 注意加锁、解锁位置
    // 锁在不用时，尽快释放
	p.Lock()
	c := p.pool[addrStr]
	// 有连接可用
	if c != nil {
		c.markForUse()
		p.Unlock()
		return c, nil
	}

	// 没有可用连接
	var wait chan struct{}
	var ok bool
	
	// 创建限流器
	if wait, ok = p.limiter[addrStr]; !ok {
		wait = make(chan struct{})
		p.limiter[addrStr] = wait
	}
	
	// 如果存在，则不是 Lead Thread
	// 只有 LeadThread 才能创建限流器
	isLeadThread := !ok
	p.Unlock()

	if isLeadThread {
	    // 创建新连接
		c, err := p.getNewConn(dc, addr, version)
		
		// 移除地址限流器
		p.Lock()
		delete(p.limiter, addrStr)
		close(wait)
		if err != nil {
			p.Unlock()
			return nil, err
		}

		p.pool[addrStr] = c
		p.Unlock()
		return c, nil
	}

	// 非 Lead Thread 执行到此
	select {
	case <-p.shutdownCh:
		return nil, fmt.Errorf("rpc error: shutdown")
	case <-wait:        // Lead Thread 执行完毕
	}

	// 是否有可用的连接
	p.Lock()
	if c := p.pool[addrStr]; c != nil {
		c.markForUse()
		p.Unlock()
		return c, nil
	}

	p.Unlock()
	return nil, fmt.Errorf("rpc error: lead thread didn't get connection")
}

// TCP 半关闭
type HalfCloser interface {
	CloseWrite() error
}

func (p *ConnPool) DialTimeout(dc string, addr net.Addr, timeout time.Duration) (net.Conn, HalfCloser, error) {
	// 尝试连接
	conn, err := net.DialTimeout("tcp", addr.String(), defaultDialTimeout)
	if err != nil {
		return nil, nil, err
	}

	var hc HalfCloser
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetNoDelay(true)
		hc = tcp        // TCPConn 本身有 CloseWrite 方法
	}

	// TLS 检查
	if p.tlsWrap != nil {
		// 切换至 TLS 模式
		if _, err := conn.Write([]byte{byte(rpcTLS)}); err != nil {
			conn.Close()
			return nil, nil, err
		}

		tlsConn, err := p.tlsWrap(dc, conn)
		if err != nil {
			conn.Close()
			return nil, nil, err
		}
		conn = tlsConn
	}

	return conn, hc, nil
}

func (p *ConnPool) getNewConn(dc string, addr net.Addr, version int) (*Conn, error) {
	conn, _, err := p.DialTimeout(dc, addr, defaultDialTimeout)
	if err != nil {
		return nil, err
	}

	var session muxSession
	if version < 2 {
	    // 版本太低，关闭，并返回错误
		conn.Close()
		return nil, fmt.Errorf("cannot make client connection, unsupported protocol version %d", version)
	} else {
		// 发送版本号
		if _, err := conn.Write([]byte{byte(rpcMultiplexV2)}); err != nil {
			conn.Close()
			return nil, err
		}

		// 日志设置
		conf := yamux.DefaultConfig()
		conf.LogOutput = p.logOutput

		// Session
		session, _ = yamux.Client(conn, conf)
	}

	c := &Conn{
		refCount: 1,            // 引用计数
		addr:     addr,
		session:  session,
		clients:  list.New(),
		lastUsed: time.Now(),   // 最后使用
		version:  version,
		pool:     p,
	}
	return c, nil
}

func (p *ConnPool) clearConn(conn *Conn) {
	// 原子操作，确保关闭设置
	atomic.StoreInt32(&conn.shouldClose, 1)

	addrStr := conn.addr.String()
	p.Lock()
	if c, ok := p.pool[addrStr]; ok && c == conn {
		delete(p.pool, addrStr)
	}
	p.Unlock()

	// 引用计数处理
	if refCount := atomic.LoadInt32(&conn.refCount); refCount == 0 {
		conn.Close()
	}
}

func (p *ConnPool) releaseConn(conn *Conn) {
    // 引用计数减一
	refCount := atomic.AddInt32(&conn.refCount, -1)
	
	// 是否需要关闭连接
	if refCount == 0 && atomic.LoadInt32(&conn.shouldClose) == 1 {
		conn.Close()
	}
}

func (p *ConnPool) getClient(dc string, addr net.Addr, version int) (*Conn, *StreamClient, error) {
	retries := 0
START:
	// 尝试获取连接
	conn, err := p.acquire(dc, addr, version)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get conn: %v", err)
	}

	// 获取可用 client
	client, err := conn.getClient()
	if err != nil {
		p.clearConn(conn)
		p.releaseConn(conn)

		if retries == 0 {
			retries++
			goto START          // 注意 goto 使用
		}
		return nil, nil, fmt.Errorf("failed to start stream: %v", err)
	}
	return conn, client, nil
}

func (p *ConnPool) RPC(dc string, addr net.Addr, version int, method string, args interface{}, reply interface{}) error {
	conn, sc, err := p.getClient(dc, addr, version)
	if err != nil {
		return fmt.Errorf("rpc error: %v", err)
	}

	// RPC 调用
	err = msgpackrpc.CallWithCodec(sc.codec, method, args, reply)
	if err != nil {
		sc.Close()
		p.releaseConn(conn)
		return fmt.Errorf("rpc error: %v", err)
	}

    // 返还 client
	conn.returnClient(sc)
	
	// 释放连接
	p.releaseConn(conn)
	return nil
}

func (p *ConnPool) PingConsulServer(s *agent.Server) (bool, error) {
	conn, sc, err := p.getClient(s.Datacenter, s.Addr, s.Version)
	if err != nil {
		return false, err
	}

	var out struct{}
	err = msgpackrpc.CallWithCodec(sc.codec, "Status.Ping", struct{}{}, &out)
	if err != nil {
		sc.Close()
		p.releaseConn(conn)
		return false, err
	}

	conn.returnClient(sc)
	p.releaseConn(conn)
	return true, nil
}

func (p *ConnPool) reap() {
	for {
		// Sleep，释放 CPU
		select {
		case <-p.shutdownCh:
			return
		case <-time.After(time.Second):
		}

		p.Lock()
		var removed []string
		now := time.Now()
		for host, conn := range p.pool {
			// 没有超时，继续
			if now.Sub(conn.lastUsed) < p.maxTime {
				continue
			}

			// 仍在使用，继续
			if atomic.LoadInt32(&conn.refCount) > 0 {
				continue
			}

			// 关闭
			conn.Close()

			removed = append(removed, host)
		}
		
		// 统一处理
		for _, host := range removed {
			delete(p.pool, host)
		}
		p.Unlock()
	}
}
```
