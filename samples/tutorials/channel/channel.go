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
)

func main() {
	nonBlocked()
	nonCacheChannelGeneral()
	blocked()
}

func nonBlocked() {
	ch := make(chan int, 1)

	ch <- 0

	select {
	case <-ch:
		fmt.Println("[nonBlocked]:read from channel")
	default:
		fmt.Println("[nonBlocked]:no data")
	}
}

func blocked() {
	ch := make(chan int)

	ch <- 0

	select {
	case <-ch:
		fmt.Println("[blocked]:read from channel")
	default:
		fmt.Println("[blocked]:no data")
	}
}

func nonCacheChannelGeneral() chan<- int {
	ch := make(chan int)

	go func() {
		select {
		case <-ch:
			fmt.Println("[nonCacheChannelGeneral]:read from channel")
		default:
			fmt.Println("[nonCacheChannelGeneral]:no data")
		}
	}()

	ch <- 0

	return ch
}
