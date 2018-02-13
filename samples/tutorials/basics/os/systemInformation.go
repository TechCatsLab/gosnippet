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
 *     Initial: 2017/06/10	Li Zebang
 */
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const goarch string = runtime.GOARCH
const goos string = runtime.GOOS

func main() {
	fmt.Printf("%s\n%s\n", goos, goarch)
	runCommand()
}

func windows() {
	windows := exec.Command("systeminfo")
	windows.Stdout = os.Stdout
	windows.Run()
}
func linux() {
	linux := exec.Command("uname", "-a")
	linux.Stdout = os.Stdout
	linux.Run()
}
func darwin() {
	darwin := exec.Command("sw_vers")
	darwin.Stdout = os.Stdout
	darwin.Run()
}

func runCommand() {
	switch goos {
	case "windows":
		windows()
	case "linux":
		linux()
	case "darwin":
		darwin()
	default:
		fmt.Println("no more information")
		return
	}
}
