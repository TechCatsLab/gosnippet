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
 *     Initial: 2017/04/14        Feng Yifei
 */

package main

import (
	"math/rand"
)

func fa() {
	s := []int{0, 1, 2, 3, 4, 5, 6}
	index := rand.Intn(7)
	_ = s[:index] // bounds check
	_ = s[index:] // bounds check eliminatd!
}

func fb(s []int, index int) {
	_ = s[:index] // bounds check
	_ = s[index:] // bounds check, not smart enough?
}

func fc() {
	s := []int{0, 1, 2, 3, 4, 5, 6}
	s = s[:4]
	index := rand.Intn(7)
	_ = s[:index] // bounds check
	_ = s[index:] // bounds check, not smart enough?
}

func fd(is []int, bs []byte) {
	if len(is) >= 256 {
		is = is[:256] // bounds check. A hint for the compiler.
		for _, n := range bs {
			_ = is[n] // bounds check eliminatd!
		}
	}
}

func fe(s []int) []int {
	s2 := make([]int, len(s))
	s2 = s2[:len(s)] // bounds check. A hint for the compiler.
	for i := range s {
		s2[i] = -s[i] // bounds check eliminatd!
	}
	return s2
}

func main() {}
