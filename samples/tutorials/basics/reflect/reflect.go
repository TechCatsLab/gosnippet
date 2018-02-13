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
	"fmt"
	"reflect"
)

type Struct struct {
	a uint32
}

func main() {
	var (
		a uint32 = 5
		b        = &a
	)

	tryReflect(false, 5, 8.0, "hello", &Struct{}, Struct{}, &a, b)
}

func tryReflect(a ...interface{}) {
	var (
		s Struct
	)

	for argIndex, arg := range a {
		fmt.Println("[reflect] argNum: ", argIndex, ", arg:", arg, ", type:", reflect.TypeOf(arg), ",kind", reflect.TypeOf(arg).Kind())

		switch f := arg.(type) {
		case bool:
			fmt.Println("[reflect] type bool")
		case int:
			fmt.Println("[reflect] type int")
		case uintptr:
			fmt.Println("[reflect] type uintptr")
		case string:
			fmt.Println("[reflect] type string")
		case float64:
			fmt.Println("[reflect] type float64")
		case reflect.Value:
			fmt.Println("[reflect] type reflect.Value")
		default:
			fmt.Println("[reflect] argType:", f)
		}
	}

	fmt.Println("[reflect] Value of s:", reflect.ValueOf(s), ", type of s:", reflect.TypeOf(s))
}
