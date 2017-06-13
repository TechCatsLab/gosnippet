/*
 * MIT License
 *
 * Copyright (c) 2017 SmartestEE Inc.
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
	"errors"
	"runtime"
	"os"
	"os/exec"
)

const (
	goArch string = runtime.GOARCH
	goOS string = runtime.GOOS
	WINDOWS = "windows"
	LINUX = "linux"
	DARWIN = "darwin"
)

var systemMap = map[string]func()error{
	WINDOWS : windows,
	LINUX : linux,
	DARWIN : darwin,
}

func main() {
	err := runCommand()
	if err != nil {
		fmt.Println(err)
	}
}

func runCommand() error {
	fmt.Println(goArch)
	funcA, bool := systemMap[goOS]
	if bool != false {
		err := funcA()
		if err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("not support your system")
	}
}

func windows() error {
	windows := exec.Command("systeminfo")
	windows.Stdout = os.Stdout
	return windows.Run()
}

func linux() error {
	linux := exec.Command("uname", "-a")
	linux.Stdout = os.Stdout
	return linux.Run()
}

func darwin() error {
	darwin := exec.Command("sw_vers" )
	darwin.Stdout = os.Stdout
	return darwin.Run()
}
