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
 *     Initial: 25/04/2017        Jia Chenhui
 */

package main

import (
	"fmt"
)

func intSeq() func() int {
	i := 0
	return func() int {
		i += 1
		return i
	}
}

func counter(start int) (func() int, func()) {
	// start 变化时，闭包中的值也发生变化
	ctr := func() int {
		return start
	}

	incr := func() {
		start++
	}

	// ctr 和 incr 都指向 start
	return ctr, incr
}

func functions1() []func() {
	arr := []int{1, 2, 3, 4}
	result := make([]func(), 0)

	// 此处的 i、v 为闭包外的变量，四个闭包都使用的同一个变量，添加到 result 时最后一个闭包里面是 3、4，
	// 所以下面调用时，四个闭包返回的值都是 3、4
	for i, v := range arr {
		result = append(result, func() { fmt.Printf("index - %d, value - %d\n", i, v) })
	}

	return result
}

// functions 函数变体
func functions2() []func() {
	arr := []int{1, 2, 3, 4}
	result := make([]func(), 0)

	for i := range arr {
		var t = i // 在闭包里面新定义变量则每个闭包里面的变量都不同
		var v = arr[i]

		result = append(result, func() { fmt.Printf("[variant] index - %d, value - %d\n", t, v) })
	}

	return result
}

func main() {
	nextInt := intSeq()

	fmt.Println("[next] -->", nextInt())
	fmt.Println("[next] -->", nextInt()) // 保留上次执行后的状态
	fmt.Println("[next] -->", nextInt())

	newInts := intSeq() // 重置函数返回状态

	fmt.Println("[new] -->", newInts())

	fmt.Println("\n------ I am the dividing line ------\n")

	// ctr, incr 和 ctr1, incr1 是不同的
	ctr1, incr1 := counter(100)
	ctr2, incr2 := counter(100)

	fmt.Println("[counter1] -->", ctr1()) // 100
	fmt.Println("[counter2] -->", ctr2()) // 100

	incr1()
	fmt.Println("[counter1] -->", ctr1()) // 101
	fmt.Println("[counter2] -->", ctr2()) // 100

	incr2()
	incr2()
	fmt.Println("[counter1] -->", ctr1()) // 101
	fmt.Println("[counter2] -->", ctr2()) // 102

	fmt.Println("\n------ I am the dividing line ------\n")

	fns1 := functions1()
	for f1 := range fns1 {
		fns1[f1]()
	}

	fmt.Println("\n------ I am the dividing line ------\n")

	fns2 := functions2()
	for f2 := range fns2 {
		fns2[f2]()
	}
}
