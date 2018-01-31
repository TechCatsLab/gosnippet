package main

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
)

func main() {
	c := cron.New()
	// cronTime := "* * * * * Mon,Wed" 表示星期一，星期三执行
	// cronTime := "0 0 19 * * *" 每天 19:00 点执行一次
	// cronTime := "* * 8-16 * * *" 表示 8am 到 4pm 整点执行（包括8和16）
	cronTime := "* */5 * * * *" // 每五分钟执行一次

	c.AddFunc(cronTime, func() {
		// Do want you want
		fmt.Println(time.Now())
	})
	c.Start()

	// 确保函数不会跳出
	select {}
}
