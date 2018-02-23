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
 *     Initial: 2018/02/23        Feng Yifei
 */

package main

import (
	"fmt"
	"math"

	"github.com/google/go-cmp/cmp"
)

func main() {
	opt := cmp.Comparer(func(x, y float64) bool {
		return (math.IsNaN(x) && math.IsNaN(y)) || x == y
	})

	x := []float64{1.0, math.NaN(), math.E, -0.0, +0.0}
	y := []float64{1.0, math.NaN(), math.E, -0.0, +0.0}
	z := []float64{1.0, math.NaN(), math.Pi, -0.0, +0.0} // Pi constant instead of E

	fmt.Println(cmp.Equal(x, y, opt))
	fmt.Println(cmp.Equal(y, z, opt))
	fmt.Println(cmp.Equal(z, x, opt))
}
