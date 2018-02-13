/*
 * MIT License
 *
 * Copyright (c) 2017 TechCatsLab
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

/*
 * Revision History:
 *     Initial: 2017/04/22        Yang Chenglong
 */

/**
 * 简介：
 *     该文件为批量处理人员信息，即将人员信息中的居住地进行统一处理
 */

package main

import (
	"fmt"
	"time"
)

//定义员工数据结构
type Person struct {
	Name    string
	Age     uint8
	Address Addr
}

//定义地址数据结构
type Addr struct {
	city     string
	district string
}

//定义处理接口，方法Batch被声明为实现批量处理人员信息功能的方法，
//其方法声明中的两个通道分别对该方法和该方法的调用方使用它的方式进行了约束
type PersonHandler interface {
	Batch(origs <-chan Person) <-chan Person
	Handle(orig *Person)
}

//定义空结构体，为其添加方法，实现PersonHandler接口
type PersonHandlerImpl struct{}

func (handler PersonHandlerImpl) Batch(origs <-chan Person) <-chan Person {
	dests := make(chan Person, 100)
	go func() {
		for p := range origs {
			handler.Handle(&p)
			dests <- p
		}
		fmt.Println("All the information has been handled.")
		//在发送方关闭通道
		close(dests)
	}()
	return dests
}

func (handler PersonHandlerImpl) Handle(orig *Person) {
	if orig.Address.district == "Haidian" {
		orig.Address.district = "Shijingshan"
	}
}

//定义要被处理的数据并初始化
var personTotal = 200

var persons []Person = make([]Person, personTotal)

var personCount int

func init() {
	for i := 0; i < 200; i++ {
		name := fmt.Sprintf("%s%d", "P", i)
		p := Person{name, 32, Addr{"Beijing", "Haidian"}}
		persons[i] = p
	}
}

//main函数中首先获取handler，初始化origs通道，将人员信息通过origs通道传入
//Batch中处理，处理后的信息放入dests通道中，并将dests通道返回。
//通道初始化完成后，fecthPerson获取人员信息放入到origs中，savePerson从dests中接收处理过的信息进行保存
//其中sign通道作用为在批处理完全执行结束之前阻塞主Goroutine
func main() {
	handler := getPersonHandler()
	origs := make(chan Person, 100)
	dests := handler.Batch(origs)
	fecthPerson(origs)
	sign := savePerson(dests)
	<-sign
}

func getPersonHandler() PersonHandler {
	return PersonHandlerImpl{}
}

func savePerson(dest <-chan Person) <-chan byte {
	sign := make(chan byte, 1)

	go func() {
		ok := true
		var p Person
		for {
			select {
			case p, ok = <-dest:
				{
					if !ok {
						fmt.Println("All the information has been saved.")
						sign <- 0
						break
					}
					savePerson1(p)
				}
			case ok = <-func() chan bool {
				timeout := make(chan bool, 1)
				go func() {
					time.Sleep(time.Millisecond)
					timeout <- false
				}()
				return timeout
			}():
				fmt.Println("TimeOut!")
				sign <- 0
				break
			}

			if !ok {
				break
			}
		}
	}()
	return sign
}

func fecthPerson(origs chan<- Person) {
	//调用cap函数确定origs是否为缓冲通道
	origsCap := cap(origs)
	buffered := origsCap > 0
	//以origsCap的一半作为Goroutine票池的总数，创建票池
	goTicketTotal := origsCap / 2
	goTicket := initGoTicket(goTicketTotal)
	go func() {
		for {
			p, ok := fecthPerson1()
			if !ok {
				for {
					//如果为非缓冲通道或者所有goroutine已完成工作，跳出循环
					if !buffered || len(goTicket) == goTicketTotal {
						break
					}
					time.Sleep(time.Nanosecond)
				}
				fmt.Println("All the information has been fetched.")
				//在发送方关闭通道
				close(origs)
				break
			}

			//如果为缓冲通道，从goTicket接受一个值，表示有一个goroutine被占用
			//当操作完成后，向其中发送一个值，表示接解除占用
			if buffered {
				<-goTicket
				go func() {
					origs <- p
					goTicket <- 1
				}()
			} else {
				origs <- p
			}
		}
	}()
}

//goTicket是为了限制该程序启用的goroutine的数量而声明的一个缓冲通道
//根据传进来的total初始化通道，total即表示可以启用goroutine数量
//每当启用一个goroutine时从该通道中接受一个值表示可用goroutine少了一个
//即每个goroutine要想启动必须要有ticket。上述是在origs为缓冲条件下，即整个过程为异步完成情况下
func initGoTicket(total int) chan byte {
	var goTicket chan byte
	if total == 0 {
		return goTicket
	}
	goTicket = make(chan byte, total)
	for i := 0; i < total; i++ {
		goTicket <- 1
	}
	return goTicket
}

func fecthPerson1() (Person, bool) {
	if personCount < personTotal {
		p := persons[personCount]
		personCount++
		return p, true
	}
	return Person{}, false
}

func savePerson1(p Person) bool {
	return true
}
