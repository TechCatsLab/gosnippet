/*
 * MIT License
 *
 * Copyright (c) 2018 TechCatsLab
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
 *     Initial: 2018/07/06        Feng Yifei
 */

package main

import (
	"fmt"
	"time"
)

func main() {
	size := 4

	ch := make(chan int, size)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("channel sending interrupt")
			}
		}()

		for i := 0; i < size; i++ {
			ch <- i
		}

		fmt.Println("Channel is full, send another item")

		ch <- 4

		fmt.Println("channel now is full, send failed")
	}()

	go func() {
		time.Sleep(2 * time.Second)

		close(ch)
	}()

	time.Sleep(4 * time.Second)

	for c := range ch {
		fmt.Println("Reading from ch:", c)
	}
}
