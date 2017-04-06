## 连接 (net/dial.go)

### Dialer
```go
// 连接选项
// 默认值为选项不起作用
type Dialer struct {
    // 两个最短的会先起作用
	Timeout time.Duration       // 连接超时时长
	Deadline time.Time          // 连接超时时刻
	LocalAddr Addr              // 使用的本地地址
	DualStack bool              // 对端同时支持 IPv4,IPv6
	FallbackDelay time.Duration // 配合 DualStack，当一个连接中断时，更换连接等待时间
	KeepAlive time.Duration     // KeepAlive 时长，0 为不使用 KeepAlive
	Resolver *Resolver          // 地址查询
	Cancel <-chan struct{}      // 连接取消，类似 Context，当去消失，chan 关闭
}

// 两个时间中，获取最近的时间
func minNonzeroTime(a, b time.Time) time.Time {
    // 为 0，说明无设置，使用另一个
	if a.IsZero() {
		return b
	}
	
	// b 无设置，或 a 在 b 前
	if b.IsZero() || a.Before(b) {
		return a
	}
	return b
}

func (d *Dialer) deadline(ctx context.Context, now time.Time) (earliest time.Time) {
    // Timeout 有设置
	if d.Timeout != 0 {
		earliest = now.Add(d.Timeout)
	}
	
	// ctx 有截止时间；
	// Context 超过截止时间后自动失效
	if d, ok := ctx.Deadline(); ok {
		earliest = minNonzeroTime(earliest, d)
	}
	
	// 选择最近的返回
	return minNonzeroTime(earliest, d.Deadline)
}

// 使用的地址查询
func (d *Dialer) resolver() *Resolver {
    // 优先使用配置的
	if d.Resolver != nil {
		return d.Resolver
	}
	
	// 默认
	return DefaultResolver
}

// 当执行多个连接时，每个连接的超时
func partialDeadline(now, deadline time.Time, addrsRemaining int) (time.Time, error) {
    // 无设置，直接返回
	if deadline.IsZero() {
		return deadline, nil
	}
	
	// 超时已触发，返回错误
	timeRemaining := deadline.Sub(now)
	if timeRemaining <= 0 {
		return time.Time{}, errTimeout
	}
	
	// 平均分配
	timeout := timeRemaining / time.Duration(addrsRemaining)
	const saneMinimum = 2 * time.Second
	
	// 超时时间太短
	if timeout < saneMinimum {
		if timeRemaining < saneMinimum {
			timeout = timeRemaining
		} else {
			timeout = saneMinimum
		}
	}
	return now.Add(timeout), nil
}

func (d *Dialer) fallbackDelay() time.Duration {
	if d.FallbackDelay > 0 {
		return d.FallbackDelay
	} else {
		return 300 * time.Millisecond
	}
}

// 获取网络地址
func parseNetwork(ctx context.Context, net string) (afnet string, proto int, err error) {
	i := last(net, ':')
	if i < 0 {
		switch net {
		case "tcp", "tcp4", "tcp6":
		case "udp", "udp4", "udp6":
		case "ip", "ip4", "ip6":
		case "unix", "unixgram", "unixpacket":
		default:
			return "", 0, UnknownNetworkError(net)
		}
		return net, 0, nil
	}
	afnet = net[:i]
	switch afnet {
	case "ip", "ip4", "ip6":
		protostr := net[i+1:]
		proto, i, ok := dtoi(protostr)
		if !ok || i != len(protostr) {
			proto, err = lookupProtocol(ctx, protostr)
			if err != nil {
				return "", 0, err
			}
		}
		return afnet, proto, nil
	}
	return "", 0, UnknownNetworkError(net)
}

// 获取地址列表
// type addrList []Addr
func (r *Resolver) resolveAddrList(ctx context.Context, op, network, addr string, hint Addr) (addrList, error) {
    ...
}

// 连接
func (d *Dialer) Dial(network, address string) (Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// 连接参数
type dialParam struct {
	Dialer
	network, address string
}

// 实际连接执行
// address 可以为域名
// address 为域名时，如果解析到多个 IP 地址
func (d *Dialer) DialContext(ctx context.Context, network, address string) (Conn, error) {
    // 必须传入 ctx
	if ctx == nil {
		panic("nil context")
	}
	
	// 计数超时时间
	deadline := d.deadline(ctx, time.Now())
	if !deadline.IsZero() {
	    // 传入 ctx 没有 Deadline
	    // Deadline 早于计算的超时时间
	    // 则新建 Cancel Context
		if d, ok := ctx.Deadline(); !ok || deadline.Before(d) {
			subCtx, cancel := context.WithDeadline(ctx, deadline)
			defer cancel()  // 正常退出是，自动取消
			ctx = subCtx    // 使用新的 Timer Context
		}
	}
	
	// 如果 Dialer 支持 Cancel
	if oldCancel := d.Cancel; oldCancel != nil {
		subCtx, cancel := context.WithCancel(ctx)
		defer cancel()              // 新建 Cancel Context，并在退出后自动 cancel
		
		go func() {
			select {
			case <-oldCancel:       // Dialer 自身 Cancel 触发
				cancel()            // 同时取消新建的 Cancel Context
			case <-subCtx.Done():   // 新建 Cancel Context 触发
			}
		}()
		ctx = subCtx                // 使用新 Cancel Context
	}

	// 解析 Context 使用最后设置的 Context
	resolveCtx := ctx
	// 有预设 Key/Value，则地址解释使用新 Context
	if trace, _ := ctx.Value(nettrace.TraceKey{}).(*nettrace.Trace); trace != nil {
		shadow := *trace
		shadow.ConnectStart = nil
		shadow.ConnectDone = nil
		resolveCtx = context.WithValue(resolveCtx, nettrace.TraceKey{}, &shadow)
	}

    // 解释地址
	addrs, err := d.resolver().resolveAddrList(resolveCtx, "dial", network, address, d.LocalAddr)
	if err != nil {
		return nil, &OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
	}

    // 连接参数
	dp := &dialParam{
		Dialer:  *d,
		network: network,
		address: address,
	}

    // 双协议栈支持
	var primaries, fallbacks addrList
	if d.DualStack && network == "tcp" {
	    // 根据是否包含 IPv4 地址，分为两组
		primaries, fallbacks = addrs.partition(isIPv4)
	} else {
		primaries = addrs
	}

    // 地址获取完毕，开始连接
	var c Conn
	if len(fallbacks) > 0 {
		c, err = dialParallel(ctx, dp, primaries, fallbacks)
	} else {
		c, err = dialSerial(ctx, dp, primaries)
	}
	
	// 连接过程发生错误
	if err != nil {
		return nil, err
	}

    // TCP 连接，是否需要设置 KeepAlive
	if tc, ok := c.(*TCPConn); ok && d.KeepAlive > 0 {
		setKeepAlive(tc.fd, true)
		setKeepAlivePeriod(tc.fd, d.KeepAlive)
		testHookSetKeepAlive()
	}
	return c, nil
}
```
### 连接方法
```go
// 顺序连接
// 当有一个地址连接成功时，直接返回
// 连接失败，返回第一个错误
func dialSerial(ctx context.Context, dp *dialParam, ras addrList) (Conn, error) {
	var firstErr error

    // 遍历地址列表
	for i, ra := range ras {
		select {
		case <-ctx.Done():      // Context 生命周期完结
			return nil, &OpError{Op: "dial", Net: dp.network, Source: dp.LocalAddr, Addr: ra, Err: mapErr(ctx.Err())}
		default:                // 不阻塞
		}

        // 计算本次连接超时时间
		deadline, _ := ctx.Deadline()
		partialDeadline, err := partialDeadline(time.Now(), deadline, len(ras)-i)
		
		// 已超时
		if err != nil {
			if firstErr == nil {
				firstErr = &OpError{Op: "dial", Net: dp.network, Source: dp.LocalAddr, Addr: ra, Err: err}
			}
			break
		}
		
		// 设置本次连接 Context
		dialCtx := ctx
		// 如果需要重设 Context
		if partialDeadline.Before(deadline) {
			var cancel context.CancelFunc
			dialCtx, cancel = context.WithDeadline(ctx, partialDeadline)
			defer cancel()
		}

        // 连接
		c, err := dialSingle(dialCtx, dp, ra)
		
		// 一个地址连接成功就返回！！！
		if err == nil {
			return c, nil
		}
		
		// 记录首个错误
		if firstErr == nil {
			firstErr = err
		}
	}

    // 连接不成功时才会执行到这里
    // 如果没有错误，那么只有一种可能，地址列表为空
	if firstErr == nil {
		firstErr = &OpError{Op: "dial", Net: dp.network, Source: nil, Addr: nil, Err: errMissingAddress}
	}
	return nil, firstErr
}

// 根据类型，选择合适的连接方法
func dialSingle(ctx context.Context, dp *dialParam, ra Addr) (c Conn, err error) {
    // 是否有预设跟踪址
	trace, _ := ctx.Value(nettrace.TraceKey{}).(*nettrace.Trace)
	if trace != nil {
		raStr := ra.String()
		if trace.ConnectStart != nil {
			trace.ConnectStart(dp.network, raStr)
		}
		if trace.ConnectDone != nil {
			defer func() { trace.ConnectDone(dp.network, raStr, err) }()
		}
	}
	
	// 根据参数地址类型，执行对应连接方法
	// dp.LocalAddr 为 Dialer 的 LocalAddr
	la := dp.LocalAddr
	switch ra := ra.(type) {
	case *TCPAddr:
		la, _ := la.(*TCPAddr)
		c, err = dialTCP(ctx, dp.network, la, ra)
	case *UDPAddr:
		la, _ := la.(*UDPAddr)
		c, err = dialUDP(ctx, dp.network, la, ra)
	case *IPAddr:
		la, _ := la.(*IPAddr)
		c, err = dialIP(ctx, dp.network, la, ra)
	case *UnixAddr:
		la, _ := la.(*UnixAddr)
		c, err = dialUnix(ctx, dp.network, la, ra)
	default:
		return nil, &OpError{Op: "dial", Net: dp.network, Source: la, Addr: ra, Err: &AddrError{Err: "unexpected address type", Addr: dp.address}}
	}
	
	if err != nil {
		return nil, &OpError{Op: "dial", Net: dp.network, Source: la, Addr: ra, Err: err}
	}
	return c, nil
}
```

### 全局方法
```go

// 使用默认选项连接
func Dial(network, address string) (Conn, error) {
	var d Dialer
	return d.Dial(network, address)
}

// 添加连接超时选项
func DialTimeout(network, address string, timeout time.Duration) (Conn, error) {
	d := Dialer{Timeout: timeout}
	return d.Dial(network, address)
}

// 监听地址(TCP 类)
func Listen(net, laddr string) (Listener, error) {
    // 解析地址
	addrs, err := DefaultResolver.resolveAddrList(context.Background(), "listen", net, laddr, nil)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: net, Source: nil, Addr: nil, Err: err}
	}
	
	// 根据类型执行对应监听函数 
	var l Listener
	switch la := addrs.first(isIPv4).(type) {
	case *TCPAddr:
		l, err = ListenTCP(net, la)
	case *UnixAddr:
		l, err = ListenUnix(net, la)
	default:
		return nil, &OpError{Op: "listen", Net: net, Source: nil, Addr: la, Err: &AddrError{Err: "unexpected address type", Addr: laddr}}
	}
	
	if err != nil {
		return nil, err
	}
	return l, nil
}

// 监听地址(UDP 类)
func ListenPacket(net, laddr string) (PacketConn, error) {
	addrs, err := DefaultResolver.resolveAddrList(context.Background(), "listen", net, laddr, nil)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: net, Source: nil, Addr: nil, Err: err}
	}
	
	var l PacketConn
	switch la := addrs.first(isIPv4).(type) {
	case *UDPAddr:
		l, err = ListenUDP(net, la)
	case *IPAddr:
		l, err = ListenIP(net, la)
	case *UnixAddr:
		l, err = ListenUnixgram(net, la)
	default:
		return nil, &OpError{Op: "listen", Net: net, Source: nil, Addr: la, Err: &AddrError{Err: "unexpected address type", Addr: laddr}}
	}
	
	if err != nil {
		return nil, err
	}
	return l, nil
}
```
