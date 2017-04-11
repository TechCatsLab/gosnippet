```go
package main

import (
	"fmt"
	"encoding/json"
)

// json.Marshal() 用于将节结构转化为json
// json.Unmarshal() 将json转化为对象

type User struct {
	//结构字段必须是首字母大写
	Name  string   `json:"name"` //``中的字段用于指定json键名，若不指定，则使用结构名
	Age   int      `json:"age"`
	Sex   string   `json:"sex"`
}

func main() {
	user := &User{"tang", 20, "male"}
	fmt.Println(user)

	//将结构转化为json
	user_json, err := json.Marshal(user)
	if err != nil {
		return
	}

	//需强制转化为string类型，若不强转，则输出byte数组
	fmt.Println(string(user_json))

	//将json转回结构
	user_ := &User{}

	//第一个参数为要处理的json，第二个参数用于接收处理的结果，需指定类型
	err_ := json.Unmarshal(user_json, user_)

	if err_ != nil {
		return
	}

	fmt.Println(user_)
}

```



