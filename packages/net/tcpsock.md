## TCP (net/tcpsock.go)

### 地址结构
```go
type TCPAddr struct {
	IP   IP
	Port int   
	Zone string     // IPv6 域
}

// 实现 Addr 接口
func (a *TCPAddr) Network() string { return "tcp" }

func (a *TCPAddr) String() string {
    // 注意参数保护！！
	if a == nil {
		return "<nil>"
	}
	
	ip := ipEmptyString(a.IP)
	if a.Zone != "" {
		return JoinHostPort(ip+"%"+a.Zone, itoa(a.Port))
	}
	
	return JoinHostPort(ip, itoa(a.Port))
}

// 结构新方法
func (a *TCPAddr) isWildcard() bool {
	if a == nil || a.IP == nil {
		return true
	}
	return a.IP.IsUnspecified()
}

func (a *TCPAddr) opAddr() Addr {
	if a == nil {
		return nil
	}
	return a
}
```

### 连接结构
```go
type TCPConn struct {
	conn                // 自动包含结构方法
}

func (c *TCPConn) ReadFrom(r io.Reader) (int64, error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	
	// 传入平台相关代码，不要深究
	n, err := c.readFrom(r)
	if err != nil && err != io.EOF {
		err = &OpError{Op: "readfrom", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return n, err
}

// 单向关闭读通道
func (c *TCPConn) CloseRead() error {
	if !c.ok() {
		return syscall.EINVAL
	}
	
	// 传入 conn，使用平台相关代码
	if err := c.fd.closeRead(); err != nil {
		return &OpError{Op: "close", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return nil
}

// 单向关闭写通道
func (c *TCPConn) CloseWrite() error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := c.fd.closeWrite(); err != nil {
		return &OpError{Op: "close", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return nil
}

// 设置连接关闭时，如果有未发送完毕数据时，连接行为
func (c *TCPConn) SetLinger(sec int) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	
	if err := setLinger(c.fd, sec); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return nil
}

// Keepalive 设置
func (c *TCPConn) SetKeepAlive(keepalive bool) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := setKeepAlive(c.fd, keepalive); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return nil
}

func (c *TCPConn) SetKeepAlivePeriod(d time.Duration) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := setKeepAlivePeriod(c.fd, d); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return nil
}

// noDelay 为 true时，包立即发送；每包一个 ACK
func (c *TCPConn) SetNoDelay(noDelay bool) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := setNoDelay(c.fd, noDelay); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return nil
}

func newTCPConn(fd *netFD) *TCPConn {
	c := &TCPConn{conn{fd}}
	
	// 默认设置 noDelay 为 true！！很好
	setNoDelay(c.fd, true)
	return c
}
```

### 监听处理
```go
type TCPListener struct {
	fd *netFD
}

// 接受新连接
func (l *TCPListener) AcceptTCP() (*TCPConn, error) {
	if !l.ok() {
		return nil, syscall.EINVAL
	}
	c, err := l.accept()
	if err != nil {
		return nil, &OpError{Op: "accept", Net: l.fd.net, Source: nil, Addr: l.fd.laddr, Err: err}
	}
	return c, nil
}


func (l *TCPListener) Accept() (Conn, error) {
	if !l.ok() {
		return nil, syscall.EINVAL
	}
	c, err := l.accept()
	if err != nil {
		return nil, &OpError{Op: "accept", Net: l.fd.net, Source: nil, Addr: l.fd.laddr, Err: err}
	}
	return c, nil
}

// 关闭监听
func (l *TCPListener) Close() error {
	if !l.ok() {
		return syscall.EINVAL
	}
	if err := l.close(); err != nil {
		return &OpError{Op: "close", Net: l.fd.net, Source: nil, Addr: l.fd.laddr, Err: err}
	}
	return nil
}

// 监听地址
func (l *TCPListener) Addr() Addr { return l.fd.laddr }

func (l *TCPListener) SetDeadline(t time.Time) error {
	if !l.ok() {
		return syscall.EINVAL
	}
	if err := l.fd.setDeadline(t); err != nil {
		return &OpError{Op: "set", Net: l.fd.net, Source: nil, Addr: l.fd.laddr, Err: err}
	}
	return nil
}

func (l *TCPListener) File() (f *os.File, err error) {
	if !l.ok() {
		return nil, syscall.EINVAL
	}
	f, err = l.file()
	if err != nil {
		return nil, &OpError{Op: "file", Net: l.fd.net, Source: nil, Addr: l.fd.laddr, Err: err}
	}
	return
}

```

### 全局方法
```go
func ResolveTCPAddr(net, addr string) (*TCPAddr, error) {
	switch net {
	case "tcp", "tcp4", "tcp6":
	case "":
		net = "tcp"
	default:
		return nil, UnknownNetworkError(net)
	}
	
	addrs, err := DefaultResolver.internetAddrList(context.Background(), net, addr)
	if err != nil {
		return nil, err
	}
	return addrs.first(isIPv4).(*TCPAddr), nil
}

// 创建 TCP 连接
func DialTCP(net string, laddr, raddr *TCPAddr) (*TCPConn, error) {
    // 注意参数保护
	switch net {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, &OpError{Op: "dial", Net: net, Source: laddr.opAddr(), Addr: raddr.opAddr(), Err: UnknownNetworkError(net)}
	}
	
	if raddr == nil {
		return nil, &OpError{Op: "dial", Net: net, Source: laddr.opAddr(), Addr: nil, Err: errMissingAddress}
	}
	
	c, err := dialTCP(context.Background(), net, laddr, raddr)
	if err != nil {
		return nil, &OpError{Op: "dial", Net: net, Source: laddr.opAddr(), Addr: raddr.opAddr(), Err: err}
	}
	return c, nil
}

// 监听地址
func ListenTCP(net string, laddr *TCPAddr) (*TCPListener, error) {
	switch net {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, &OpError{Op: "listen", Net: net, Source: nil, Addr: laddr.opAddr(), Err: UnknownNetworkError(net)}
	}
	if laddr == nil {
		laddr = &TCPAddr{}
	}
	ln, err := listenTCP(context.Background(), net, laddr)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: net, Source: nil, Addr: laddr.opAddr(), Err: err}
	}
	return ln, nil
}
```
