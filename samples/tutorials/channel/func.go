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
 *     Initial: 2017/05/09        Feng Yifei
 */

package main

import (
	"fmt"
	"time"
)

type eventFunc func() error

func events(ch <-chan eventFunc) {
	for {
		e := <-ch
		if err := e(); err != nil {
			fmt.Printf("event error!\n")
		}
	}
}

func myfunc(ch chan<- eventFunc) {
	fmt.Printf("Doing Work...\n")
	time.Sleep(2 * time.Second)
	fmt.Printf("Sending Callbacks\n")
	ch <- func() error {
		fmt.Printf("Hello, World!\n")
		return nil
	}
	ch <- func() error {
		fmt.Printf("Done Working!\n")
		return nil
	}
}

func main() {
	ch := make(chan eventFunc, 10)
	go events(ch)
	myfunc(ch)
	time.Sleep(5 * time.Second)
}
