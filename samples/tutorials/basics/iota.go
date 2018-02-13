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
 *     Initial:  2017/04/05        Feng Yifei
 *     Version2: 2017/04/11        Jia Chenhui
 */

package main

import (
	"fmt"
)

const (
	mutexLocked       = 1 << iota // iota == 0: 1 << 0 == 1
	mutexWoken                    // iota == 1: 1 << 1 == 2
	mutexWaiterShift  = iota      // iota == 2
	mutexWaiterShift1             // iota == 3
	mutexLocked1      = 1 << iota // iota == 4: 1 << 4 == 16
	mutexWaiterShift2             // iota == 5: i << 5 == 32
)

const ( // iota is reset to 0
	iotaReset = 1 << iota
)

func main() {
	fmt.Println("mutexLocked:", mutexLocked)
	fmt.Println("mutexWoken:", mutexWoken)
	fmt.Println("mutexWaiterShift:", mutexWaiterShift)
	fmt.Println("mutexWaiterShift1:", mutexWaiterShift1)
	fmt.Println("mutexLocked1:", mutexLocked1)
	fmt.Println("mutexWaiterShift2:", mutexWaiterShift2)

	fmt.Println("iotaReset:", iotaReset)
}
