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
	"sync"
	"time"
)

func main() {
	simple()

	fmt.Println("[simple] executed succeed")

	concurrent()
	fmt.Println("[concurrent] executed finished")

	time.Sleep(3 * time.Second)
}

func simple() {
	var (
		wg    sync.WaitGroup
		count int = 5
	)

	for i := 0; i < count; i++ {
		wg.Add(1) // 或者在 for 循环外： wg.Add(count)

		go func(a int) {
			defer wg.Done()

			fmt.Println("[simple] goroutine label:", a)
		}(i)
	}

	wg.Wait()
	wg.Wait() // 可以多次使用，不会报错！
}

func concurrent() {
	var (
		wg    sync.WaitGroup
		count int = 5
	)

	for i := 0; i < count; i++ {
		go func(a int) {
			wg.Add(1) // 这里添加计数，会造成执行序列不可控，一定注意！
			defer wg.Done()

			fmt.Println("[concurrent] goroutine label:", a)
		}(i)
	}

	wg.Wait()
}
