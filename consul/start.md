## main
```go
var included []string

// 遍历全部指令
// 并移除 configtest
for command := range Commands {
	if command != "configtest" {
		included = append(included, command)
	}
}

// 设置 CLI
cli := &cli.CLI{
	Args:     args,
	Commands: Commands,
	HelpFunc: cli.FilteredHelpFunc(included, cli.BasicHelpFunc("consul")),
}

// 命令行，每次执行一条命令
// 如： consul xxx
exitCode, err := cli.Run()
```