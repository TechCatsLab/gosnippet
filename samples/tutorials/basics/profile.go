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
 *     Initial: 2017/04/05        Tang Xiaozhe
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

func main() {
	cpuf, err := os.Create("cpu_profile")
	if err != nil {
		log.Fatal(err)
	}

	pprof.StartCPUProfile(cpuf)
	defer pprof.StopCPUProfile()

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	test(ctx)

	time.Sleep(time.Second * 3)

	memf, err := os.Create("mem_profile")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	if err := pprof.WriteHeapProfile(memf); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	memf.Close()
}

func test(c context.Context) {
	i := 0
	j := 0

	go func() {
		m := map[int]int{}
		for {
			i++
			m[i] = i
		}
	}()

	go func() {
		m := map[int]int{}
		for {
			j++
			m[i] = i
		}
	}()

	select {
	case <-c.Done():
		fmt.Println("DONE, i:", i, " j:", j)
		return
	}
}
