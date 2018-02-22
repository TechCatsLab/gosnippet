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
 *     Initial: 2017/04/05        Feng Yifei
 */

package main

import (
	"errors"
	"fmt"
)

func deferWithoutReturnName() int {
	var (
		i int
	)

	defer func() {
		i++
		fmt.Println("1st:", i)
	}()

	defer func() {
		i++
		fmt.Println("2nd:", i)
	}()

	return i
}

func deferWithReturnName() (i int) {
	defer func() {
		i++
		fmt.Println("1st:", i)
	}()

	defer func() {
		i++
		fmt.Println("2nd:", i)
	}()

	return i // return 0 效果是一样的
}

func deferWithAddress() *int {
	var (
		i int
	)

	defer func() {
		i++
		fmt.Println("1st:", i)
	}()

	defer func() {
		i++
		fmt.Println("2nd:", i)
	}()

	return &i
}

func deferError() error {
	var (
		err error
	)

	defer func() {
		err = errors.New("shouldn't change the return value")
	}()

	return err
}

/**
 * 从结果分析看：
 *     当返回值命名时，使用的是地址方式的引用
 */
func main() {
	fmt.Println("deferWithoutReturnName", deferWithoutReturnName())
	fmt.Println("deferWithReturnName", deferWithReturnName())
	fmt.Println("deferWithAddress", *deferWithAddress())
	fmt.Println("deferError", deferError())
}
