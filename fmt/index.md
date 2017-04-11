```go
//i/o函数，类似于c语言的printf和scanf
//doc: https://golang.org/pkg/fmt/
package main

import "fmt"

type user struct {
	name string
}

func main() {
	u := user{"tang"}
	//Printf 格式化输出
	fmt.Printf("% + v\n", u) //格式化输出结构
	fmt.Printf("%#v\n", u) //输出值得Go语言表示方法
	fmt.Printf("%T\n", u) //输出值得类型的Go语言表示
	fmt.Printf("%t\n", true) //输出值得true或false
	fmt.Printf("%b\n", 1024) //二进制表示
	fmt.Printf("%c\n", 11111111) //数值对应的Unicode编码字符
	fmt.Printf("%d\n", 10) //十进制表示
	fmt.Printf("%o\n", 8) //八进制表示
	fmt.Printf("%q\n", 22) //转化为十六进制并附上单引号
	fmt.Printf("%x\n", 1223) //十六进制表示，用a-f表示
	fmt.Printf("%X\n", 1223) //十六进制表示，用A-F表示
	fmt.Printf("%U\n", 1233) //Unicode表示
	fmt.Printf("%b\n", 12.34) //无小数部分，两位指数的科学计数法6946802425218990p-49
	fmt.Printf("%e\n", 12.345) //科学计数法，e表示
	fmt.Printf("%E\n", 12.34455) //科学计数法，E表示
	fmt.Printf("%f\n", 12.3456) //有小数部分，无指数部分
	fmt.Printf("%g\n", 12.3456) //根据实际情况采用%e或%f输出
	fmt.Printf("%G\n", 12.3456) //根据实际情况采用%E或%f输出
	fmt.Printf("%s\n", "wqdew") //直接输出字符串或者[]byte
	fmt.Printf("%q\n", "dedede") //双引号括起来的字符串
	fmt.Printf("%x\n", "abczxc") //每个字节用两字节十六进制表示，a-f表示
	fmt.Printf("%X\n", "asdzxc") //每个字节用两字节十六进制表示，A-F表示
	fmt.Printf("%p\n", 0x123) //0x开头的十六进制数表示
}

```



