## 升级检测
### HTTP Client
```go
func DefaultTransport() *http.Transport {
	transport := DefaultPooledTransport()
	transport.DisableKeepAlives = true
	transport.MaxIdleConnsPerHost = -1
	return transport
}

func DefaultPooledTransport() *http.Transport {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
	return transport
}

func DefaultClient() *http.Client {
	return &http.Client{
		Transport: DefaultTransport(),
	}
}

func DefaultPooledClient() *http.Client {
	return &http.Client{
		Transport: DefaultPooledTransport(),
	}
}

```

### 检测
```go

var magicBytes [4]byte = [4]byte{0x35, 0x77, 0x69, 0xFB}            // 唯一标示

// 请求
type CheckParams struct {
	Product string
	Version string
	Arch string
	OS   string
	Signature     string
	SignatureFile string
	CacheFile     string
	CacheDuration time.Duration
	Force bool
}

// 应答
type CheckResponse struct {
	Product             string
	CurrentVersion      string `json:"current_version"`
	CurrentReleaseDate  int    `json:"current_release_date"`
	CurrentDownloadURL  string `json:"current_download_url"`
	CurrentChangelogURL string `json:"current_changelog_url"`
	ProjectWebsite      string `json:"project_website"`
	Outdated            bool   `json:"outdated"`
	Alerts              []*CheckAlert
}

// 报警信息
type CheckAlert struct {
	ID      int
	Date    int
	Message string
	URL     string
	Level   string
}

// 检测新版本
func Check(p *CheckParams) (*CheckResponse, error) {
    // 从环境变量中获取是否开启升级检测
    // 并检查是否要强制升级
	if disabled := os.Getenv("CHECKPOINT_DISABLE"); disabled != "" && !p.Force {
		return &CheckResponse{}, nil
	}

	// 有缓存且没有过期
	if r, err := checkCache(p.Version, p.CacheFile, p.CacheDuration); err != nil {
		return nil, err
	} else if r != nil {
		defer r.Close()
		return checkResult(r)
	}

	var u url.URL

    // Arch && Os 信息
	if p.Arch == "" {
		p.Arch = runtime.GOARCH
	}
	if p.OS == "" {
		p.OS = runtime.GOOS
	}

	// 检查签名文件
	signature := p.Signature
	if p.Signature == "" && p.SignatureFile != "" {
		var err error
		signature, err = checkSignature(p.SignatureFile)
		if err != nil {
			return nil, err
		}
	}

    // 构造参数
	v := u.Query()
	v.Set("version", p.Version)
	v.Set("arch", p.Arch)
	v.Set("os", p.OS)
	v.Set("signature", signature)

    // 拼接 URL
	u.Scheme = "https"
	u.Host = "checkpoint-api.hashicorp.com"
	u.Path = fmt.Sprintf("/v1/check/%s", p.Product)
	u.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	
	// HTTP Request Header
	// User-Agent: 可在服务端做简单验证处理
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "HashiCorp/go-checkpoint")

	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Unknown status: %d", resp.StatusCode)
	}

	var r io.Reader = resp.Body
	if p.CacheFile != "" {
		// 确保缓存目录存在
		// 不存在尝试创建
		if err := os.MkdirAll(filepath.Dir(p.CacheFile), 0755); err != nil {
			return nil, err
		}

		// 保存缓存
		f, err := os.Create(p.CacheFile)
		if err != nil {
			return nil, err
		}

		if err := writeCacheHeader(f, p.Version); err != nil {
			f.Close()
			os.Remove(p.CacheFile)
			return nil, err
		}

		defer f.Close()
		r = io.TeeReader(r, f)
	}

	return checkResult(r)
}

// 定期检测
// 典型 Go 代码，函数内部执行一个协程，返回关停控制，由外部调用者决定关停
func CheckInterval(p *CheckParams, interval time.Duration, cb func(*CheckResponse, error)) chan struct{} {
	doneCh := make(chan struct{})

	if disabled := os.Getenv("CHECKPOINT_DISABLE"); disabled != "" {
		return doneCh
	}

	go func() {
		for {
			select {
			case <-time.After(randomStagger(interval)):
				resp, err := Check(p)
				cb(resp, err)
			case <-doneCh:
				return
			}
		}
	}()

	return doneCh
}

func randomStagger(interval time.Duration) time.Duration {
	stagger := time.Duration(mrand.Int63()) % (interval / 2)
	return 3*(interval/4) + stagger
}

func checkCache(current string, path string, d time.Duration) (io.ReadCloser, error) {
	fi, err := os.Stat(path)
	if err != nil {
	    // 文件不存在，不是错误情况
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	if d == 0 {
		d = 48 * time.Hour      // 默认缓存有效期为 2 天
	}

    // 缓存过期
	if fi.ModTime().Add(d).Before(time.Now()) {
		os.Remove(path)
		return nil, nil
	}

	// 检查内容
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// 签名是否正确，注意使用：LittleEndian
	var sig [4]byte
	if err := binary.Read(f, binary.LittleEndian, sig[:]); err != nil {
		f.Close()
		return nil, err
	}
	
	// slice 比较的一种方式
	if !reflect.DeepEqual(sig, magicBytes) {
		f.Close()
		return nil, nil
	}

	// 检查版本信息
	var length uint32
	if err := binary.Read(f, binary.LittleEndian, &length); err != nil {
		f.Close()
		return nil, err
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(f, data); err != nil {
		f.Close()
		return nil, err
	}
	if string(data) != current {
		// 版本发生变化，重置
		f.Close()
		return nil, nil
	}

	return f, nil
}

func checkResult(r io.Reader) (*CheckResponse, error) {
	var result CheckResponse
	dec := json.NewDecoder(r)
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func checkSignature(path string) (string, error) {
	_, err := os.Stat(path)
	if err == nil {
		// 读取签名内容
		sigBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}

		lines := strings.SplitN(string(sigBytes), "\n", 2)
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0]), nil
		}
	}

	// 如果文件存在，且有错误
	if !os.IsNotExist(err) {
		return "", err
	}

	// 文件不存在
	var b [16]byte
	n := 0
	for n < 16 {
		n2, err := rand.Read(b[n:])
		if err != nil {
			return "", err
		}

		n += n2
	}
	signature := fmt.Sprintf(
		"%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(path, []byte(signature+"\n\n"+userMessage+"\n"), 0644); err != nil {
		return "", err
	}

	return signature, nil
}

func writeCacheHeader(f io.Writer, v string) error {
	if err := binary.Write(f, binary.LittleEndian, magicBytes); err != nil {
		return err
	}

	var length uint32 = uint32(len(v))
	if err := binary.Write(f, binary.LittleEndian, length); err != nil {
		return err
	}

	_, err := f.Write([]byte(v))
	return err
}

var userMessage = `
This signature is a randomly generated UUID used to de-duplicate
alerts and version information. This signature is random, it is
not based on any personally identifiable information. To create
a new signature, you can simply delete this file at any time.
See the documentation for the software using Checkpoint for more
information on how to disable it.
`
```
