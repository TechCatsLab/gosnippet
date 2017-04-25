/*
 * MIT License
 *
 * Copyright (c) 2017 SmartestEE Inc.
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

func functions() []func() {
	arr := []int{1, 2, 3, 4}
	result := make([]func(), 0)

	for i := range arr {
		result = append(result, func() { fmt.Printf("index - %d, value - %d\n", i, arr[i]) })
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

	fmt.Println("------ I am the dividing line ------")

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

	fmt.Println("------ I am the dividing line ------")

	fns := functions()
	for f := range fns {
		fns[f]()
	}
}
