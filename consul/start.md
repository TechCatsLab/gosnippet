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

cli := &cli.CLI{
	Args:     args,
	Commands: Commands,
	HelpFunc: cli.FilteredHelpFunc(included, cli.BasicHelpFunc("consul")),
}

exitCode, err := cli.Run()
```