## 基础接口及结构 (net/net.go)

### 初始化
```go
func init() {
	sysInit()
	supportsIPv4 = probeIPv4Stack()
	supportsIPv6, supportsIPv4map = probeIPv6Stack()    // IPv6 支持，基本逻辑与 probeIPv4Stack 类似，不在代码中详解
}
```

#### sysInit
* Unix 版本
```go
func sysInit() {
}
```

* Windows 版本
```go
func sysInit() {
    // Windows 使用网络必须的初始化代码，不要深究
	var d syscall.WSAData
	e := syscall.WSAStartup(uint32(0x202), &d)
	if e != nil {
		initErr = os.NewSyscallError("wsastartup", e)
	}
	canCancelIO = syscall.LoadCancelIoEx() == nil
	hasLoadSetFileCompletionNotificationModes = syscall.LoadSetFileCompletionNotificationModes() == nil
	if hasLoadSetFileCompletionNotificationModes {
		skipSyncNotif = true
		protos := [2]int32{syscall.IPPROTO_TCP, 0}
		var buf [32]syscall.WSAProtocolInfo
		len := uint32(unsafe.Sizeof(buf))
		n, err := syscall.WSAEnumProtocols(&protos[0], &buf[0], &len)
		if err != nil {
			skipSyncNotif = false
		} else {
			for i := int32(0); i < n; i++ {
				if buf[i].ServiceFlags1&syscall.XP1_IFS_HANDLES == 0 {
					skipSyncNotif = false
					break
				}
			}
		}
	}
}
```

#### probeIPv4Stack
net/ipsock_posix.go
```go
func probeIPv4Stack() bool {
    // 直接调用底层代码
    // socketFunc, closeFunc 参照下面说明
	s, err := socketFunc(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	switch err {
	case syscall.EAFNOSUPPORT, syscall.EPROTONOSUPPORT:
		return false
	case nil:
		closeFunc(s)
	}
	return true
}
```

net/hook_unix.go
```go
var (
	socketFunc        func(int, int, int) (int, error)         = syscall.Socket
	closeFunc         func(int) error                          = syscall.Close
	connectFunc       func(int, syscall.Sockaddr) error        = syscall.Connect
	listenFunc        func(int, int) error                     = syscall.Listen
	acceptFunc        func(int) (int, syscall.Sockaddr, error) = syscall.Accept
	getsockoptIntFunc func(int, int, int) (int, error)         = syscall.GetsockoptInt
)
```

net/hook_windows.go
```go
var (
	socketFunc    func(int, int, int) (syscall.Handle, error)                                                             = syscall.Socket
	closeFunc     func(syscall.Handle) error                                                                              = syscall.Closesocket
	connectFunc   func(syscall.Handle, syscall.Sockaddr) error                                                            = syscall.Connect
	connectExFunc func(syscall.Handle, syscall.Sockaddr, *byte, uint32, *uint32, *syscall.Overlapped) error               = syscall.ConnectEx
	listenFunc    func(syscall.Handle, int) error                                                                         = syscall.Listen
	acceptFunc    func(syscall.Handle, syscall.Handle, *byte, uint32, uint32, uint32, *uint32, *syscall.Overlapped) error = syscall.AcceptEx
)
```

然后各个 syscall.xxx 均在对应的 syscall_platform.go 中实现，只列举一个：

```go

func Socket(domain, typ, proto int) (fd int, err error) {
	if domain == AF_INET6 && SocketDisableIPv6 {
		return -1, EAFNOSUPPORT
	}
	fd, err = socket(domain, typ, proto)
	return
}

func socket(domain int, typ int, proto int) (fd int, err error) {
	r0, _, e1 := RawSyscall(SYS_SOCKET, uintptr(domain), uintptr(typ), uintptr(proto))
	fd = int(r0)
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}
```

### 核心接口
```go
type Addr interface {
	Network() string // 网络类型，如："tcp", "udp"
	String() string  // 字符串形式地址表示，如： "192.0.2.1:25"
}

// 基于流（如：TCP）的通用连接接口
type Conn interface {
	Read(b []byte) (n int, err error)   // 读数据
	Write(b []byte) (n int, err error)  // 写数据
	Close() error                       // 关闭连接
	LocalAddr() Addr                    // 本地地址
	RemoteAddr() Addr                   // 对端地址
	SetDeadline(t time.Time) error      // 设置读、写超时
	SetReadDeadline(t time.Time) error  // 设置读操作超时
	SetWriteDeadline(t time.Time) error // 设置写操作超时
}

// 基于流（如：TCP）的通用监听接口
type Listener interface {
	Accept() (Conn, error)              // 接受新连接
	Close() error                       // 关闭监听
	Addr() Addr                         // 监听地址
}

// 基于数据包（如：UDP）的通用连接接口
type PacketConn interface {
	ReadFrom(b []byte) (n int, addr Addr, err error)    // 读操作，同时获取发送者地址
	WriteTo(b []byte, addr Addr) (n int, err error)     // 写操作
	Close() error                                       // 关闭连接
	LocalAddr() Addr                                    // 本地地址
	SetDeadline(t time.Time) error                      // 设置读、写超时
	SetReadDeadline(t time.Time) error                  // 设置读操作超时
	SetWriteDeadline(t time.Time) error                 // 设置写操作超时
}
```

### conn 结构
```go
type conn struct {
    // 对 go 来说，结构体包含就是继承
    // 小写变量名类似 C++ 的私有继承，含义为：通过...实现...
    // 大些变量名类似 C++ 的公有继承，含义为：...是...
	fd *netFD
}

// 简单的保护
func (c *conn) ok() bool { return c != nil && c.fd != nil }

// Conn 接口实现部分

func (c *conn) Read(b []byte) (int, error) {
    // 连接是否可用，不可用直接返回错误
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	
	// 使用底层结构的操作
	n, err := c.fd.Read(b)
	
	// err == io.EOF ：读结束，没有更多内容
	if err != nil && err != io.EOF {
		err = &OpError{Op: "read", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return n, err
}

func (c *conn) Write(b []byte) (int, error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	
	n, err := c.fd.Write(b)
	if err != nil {
		err = &OpError{Op: "write", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return n, err
}

func (c *conn) Close() error {
	if !c.ok() {
		return syscall.EINVAL
	}
	err := c.fd.Close()
	if err != nil {
		err = &OpError{Op: "close", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return err
}

func (c *conn) LocalAddr() Addr {
	if !c.ok() {
		return nil
	}
	return c.fd.laddr
}

func (c *conn) RemoteAddr() Addr {
	if !c.ok() {
		return nil
	}
	return c.fd.raddr
}

func (c *conn) SetDeadline(t time.Time) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := c.fd.setDeadline(t); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: nil, Addr: c.fd.laddr, Err: err}
	}
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := c.fd.setReadDeadline(t); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: nil, Addr: c.fd.laddr, Err: err}
	}
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := c.fd.setWriteDeadline(t); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: nil, Addr: c.fd.laddr, Err: err}
	}
	return nil
}

// 设置读缓存大小
func (c *conn) SetReadBuffer(bytes int) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := setReadBuffer(c.fd, bytes); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: nil, Addr: c.fd.laddr, Err: err}
	}
	return nil
}

// 设置写缓存大小
func (c *conn) SetWriteBuffer(bytes int) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := setWriteBuffer(c.fd, bytes); err != nil {
		return &OpError{Op: "set", Net: c.fd.net, Source: nil, Addr: c.fd.laddr, Err: err}
	}
	return nil
}

// 获取底层 File 结构
func (c *conn) File() (f *os.File, err error) {
	f, err = c.fd.dup()
	if err != nil {
		err = &OpError{Op: "file", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return
}

```

### Back Log
```go
var listenerBacklog = maxListenerBacklog()

func maxListenerBacklog() int {
	var (
		n   uint32
		err error
	)
	
	// 根据操作系统做处理
	switch runtime.GOOS {
	case "darwin", "freebsd":
		n, err = syscall.SysctlUint32("kern.ipc.somaxconn")
	case "netbsd":
		// NOTE: NetBSD has no somaxconn-like kernel state so far
	case "openbsd":
		n, err = syscall.SysctlUint32("kern.somaxconn")
	}
	
	// 没有获得预设值或有错误时，使用系统默认
	if n == 0 || err != nil {
		return syscall.SOMAXCONN        // 0x80
	}
	
	// FreeBSD/Linux 使用 uint16 存储该值，通过这里设置不要超长
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	return int(n)
}
```

### 线程控制
```go
// threadLimit 控制 cgo 使用的 goroutine，在 cgo 中，每一个 goroutine 会阻塞整个 thread
var threadLimit = make(chan struct{}, 500)

// 占用一个 channel 缓冲，相当于信号量操作，但是性能要高很多，记住这样的使用
func acquireThread() {
	threadLimit <- struct{}{}
}

// 释放一个 channel
func releaseThread() {
	<-threadLimit
}

```

### 通用缓存
```go
type buffersWriter interface {
	writeBuffers(*Buffers) (int64, error)
}

// 需要注意，go 本身是没有多级 slice 的，底层存储结构都是一维
type Buffers [][]byte

var (
	_ io.WriterTo = (*Buffers)(nil)
	_ io.Reader   = (*Buffers)(nil)
)

func (v *Buffers) WriteTo(w io.Writer) (n int64, err error) {
    // 如果有自定义写入方法，这种预留接口方式也要学习！
	if wv, ok := w.(buffersWriter); ok {
		return wv.writeBuffers(v)
	}
	
	// 默认方法
	// 遍历全部剩余 []byte
	for _, b := range *v {
		nb, err := w.Write(b)       // 写入 []byte，这里写入，一定不会超过 []byte 长度！！
		n += int64(nb)              // 计数
		if err != nil {             // 写入失败，标记写入成功的计数
			v.consume(n)
			return n, err
		}
	}
	v.consume(n)                    // 写入成功，全部标记
	return n, nil
}

func (v *Buffers) Read(p []byte) (n int, err error) {
    // len(p) > 0：读缓存还有空间
    // len(*v) > 0：缓存中还有数据
	for len(p) > 0 && len(*v) > 0 {
		n0 := copy(p, (*v)[0])      // 读出数据
		v.consume(int64(n0))        // 移动 buffer 当前位置
		p = p[n0:]                  // 移动读缓存空闲指针位置
		n += n0                     // 计数
	}
	
	// 是否读完缓存
	if len(*v) == 0 {
		err = io.EOF
	}
	return
}

// 使用 consume 本身，并不能保证 buffer 剩余空间一定足够
// consume 本身含义，只是调整 buffer 剩余空间
// 这种二步骤做法，也要学习使用
func (v *Buffers) consume(n int64) {
	for len(*v) > 0 {                   // len(*v) > 0：当前 buffer 还有可用空间
		ln0 := int64(len((*v)[0]))      // 获取第一个可用的 []byte 长度
		if ln0 > n {                    // 第一个 []byte 空间足够
			(*v)[0] = (*v)[0][n:]       // 修改第一个 []byte 起始位置（长度会跟随变化）
			return
		}
		n -= ln0                        // 第一个 []byte 空间全部分配完毕后，剩余需要的空间
		*v = (*v)[1:]                   // 修改全局 buffer 长度
	}
}
```
