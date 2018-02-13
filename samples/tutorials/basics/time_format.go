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
 *     Initial: 2017/04/11        Yusan Kurban
 */
package main

import (
	"fmt"
	"time"
)

const (
	format1   = "20060102"
	longForm  = "Jan 2, 2006 at 3:04pm (MST)"
	shortForm = "2006-Jan-02"
	// 格式不止上述这些格式
)

func exampleDate() {
	t := time.Now()
	fmt.Printf("now is : %s\n", t)
}

// 把时间格式转换成给定的格式
func exampleTimeFormat() {
	fmt.Println("time is ", time.Now().Format(format1))
	fmt.Println("time is ", time.Now().Format(longForm))
	fmt.Println("time is ", time.Now().Format(shortForm))
}

// 传入的字符串解析并转换成默认的 2017-06-02 00:00:00 +0000 UTC 格式
// 但是传入格式与 parse 的第一参数的格式得一致。
func exmapleParse() {
	t, _ := time.Parse(shortForm, "2017-Jun-02")
	fmt.Println(t)
}

func main() {
	exampleDate()
	exampleTimeFormat()
	exmapleParse()
}
