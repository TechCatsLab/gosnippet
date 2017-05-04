## github.com/mitchellh/cli

### 基础知识
* [Radix Tree](https://en.wikipedia.org/wiki/Radix_tree)
* [Radix Tree Golang](https://github.com/armon/go-radix)

### Command 相关
```go
type Command interface {
	// 返回帮助信息
	Help() string

	// 执行
	Run(args []string) int

	// 返回简短的帮助信息
	Synopsis() string
}

// 辅助 Help() 实现方法
type CommandHelpTemplate interface {
	HelpTemplate() string
}

// 命令工厂，用于返回 Command
// 在内部可初始化 Command
type CommandFactory func() (Command, error)
```

### CLI
```go
type CLI struct {
	// 参数，不包含命令本身 (args[0]) 
	Args []string

	// 子命令映射
	Commands map[string]CommandFactory

	Name string

	Version string

	// 命令帮助
	HelpFunc   HelpFunc
	HelpWriter io.Writer

	once           sync.Once
	
	commandTree    *radix.Tree      // 命令树
	commandNested  bool             // 包含嵌套命令
	isHelp         bool             // 是否含帮助命令
	subcommand     string           // 子命令
	subcommandArgs []string         // 子命令参数
	topFlags       []string         // 主命令参数

	isVersion bool                  // 是否含版本命令
}
```

#### init
```go
func (c *CLI) init() {
    // 初始化 Help 相关
	if c.HelpFunc == nil {
		c.HelpFunc = BasicHelpFunc("app")

		if c.Name != "" {
			c.HelpFunc = BasicHelpFunc(c.Name)
		}
	}

	if c.HelpWriter == nil {
		c.HelpWriter = os.Stderr
	}

	// 构建命令树
	c.commandTree = radix.New()
	c.commandNested = false                 // 嵌套命令置为 false
	for k, v := range c.Commands {
		k = strings.TrimSpace(k)            // 清除前、后空格
		c.commandTree.Insert(k, v)          // 保存
		if strings.ContainsRune(k, ' ') {
			c.commandNested = true          // 包含空格，嵌套为 true
		}
	}

	// 命令含嵌套
	// 遍历命令树，补全可能丢失的父命令
	if c.commandNested {
		var walkFn radix.WalkFn
		toInsert := make(map[string]struct{})
		walkFn = func(k string, raw interface{}) bool {
			idx := strings.LastIndex(k, " ")
			if idx == -1 {
				return false
			}

			// 获取父命令
			k = k[:idx]
			if _, ok := c.commandTree.Get(k); ok {
				// 父命令已存在
				return false
			}

			// 添加父命令
			toInsert[k] = struct{}{}

			// 递归，父命令仍然可能丢失
			return walkFn(k, nil)
		}

		// 遍历
		c.commandTree.Walk(walkFn)

		// 添加父命令
		// 由于本身不存在这些父命令，因此，不能独立调用
		for k := range toInsert {
			var f CommandFactory = func() (Command, error) {
				return &MockCommand{
					HelpText:  "This command is accessed by using one of the subcommands below.",
					RunResult: RunResultHelp,
				}, nil
			}

			c.commandTree.Insert(k, f)
		}
	}

	c.processArgs()
}
```

#### processArgs
```go
func (c *CLI) processArgs() {
	for i, arg := range c.Args {
		if arg == "--" {
			break
		}

		// 内置的帮助参数
		if arg == "-h" || arg == "-help" || arg == "--help" {
			c.isHelp = true
			continue
		}

		if c.subcommand == "" {
			// 内置的版本参数
			if arg == "-v" || arg == "-version" || arg == "--version" {
				c.isVersion = true
				continue
			}

			if arg != "" && arg[0] == '-' {
				// 记录参数
				c.topFlags = append(c.topFlags, arg)
			}
		}

		// 如果 subcommand 为空
		// 且不以 '-' 开头
		// 设置为子命令
		if c.subcommand == "" && arg != "" && arg[0] != '-' {
			c.subcommand = arg
			if c.commandNested {
			    // 搜索最长匹配的子命令
				searchKey := strings.Join(c.Args[i:], " ")
				k, _, ok := c.commandTree.LongestPrefix(searchKey)
				if ok {
					reVerify := regexp.MustCompile(regexp.QuoteMeta(k) + `( |$)`)
					if reVerify.MatchString(searchKey) {
						c.subcommand = k
						i += strings.Count(k, " ")
					}
				}
			}

			// 剩余参数为子命令参数
			c.subcommandArgs = c.Args[i+1:]
		}
	}

	// 没有子命令
	if c.subcommand == "" {
	    // 支持默认命令
		if _, ok := c.Commands[""]; ok {
		    // 切换至默认命令
			args := c.topFlags
			args = append(args, c.subcommandArgs...)
			c.topFlags = nil
			c.subcommandArgs = args
		}
	}
}
```

#### Run
```go
func (c *CLI) Run() (int, error) {
	c.once.Do(c.init)           // 确保 init 只执行一次

	// 显示版本信息并退出
	if c.IsVersion() && c.Version != "" {
		c.HelpWriter.Write([]byte(c.Version + "\n"))
		return 0, nil
	}

    // 含帮助参数，且无子命令
	// 显示命令帮助，并退出
	if c.IsHelp() && c.Subcommand() == "" {
		c.HelpWriter.Write([]byte(c.HelpFunc(c.Commands) + "\n"))
		return 0, nil
	}

	// 获取子命令，如果不存在，报错
	raw, ok := c.commandTree.Get(c.Subcommand())
	if !ok {
		c.HelpWriter.Write([]byte(c.HelpFunc(c.helpCommands(c.subcommandParent())) + "\n"))
		return 1, nil
	}

    // 创建 command
	command, err := raw.(CommandFactory)()
	if err != nil {
		return 1, err
	}

	// 显示子命令帮助信息
	if c.IsHelp() {
		c.commandHelp(command)
		return 0, nil
	}

	// 子命令前不允许有参数
	if len(c.topFlags) > 0 {
		c.HelpWriter.Write([]byte(
			"Invalid flags before the subcommand. If these flags are for\n" +
				"the subcommand, please put them after the subcommand.\n\n"))
		c.commandHelp(command)
		return 1, nil
	}

    // 运行子命令
	code := command.Run(c.SubcommandArgs())
	if code == RunResultHelp {
		c.commandHelp(command)
		return 1, nil
	}

	return code, nil
}
```

### Help
```go
type HelpFunc func(map[string]CommandFactory) string
```
