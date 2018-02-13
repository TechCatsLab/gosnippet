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
 *     Initial: 2017/04/19        Feng Yifei
 */

/**
 * 抽象工厂模式：
 *     提供一个创建一系列相关或相互依赖对象的接口，而无须指定它们具体的类；
 *     在抽象工厂模式中，每一个具体工厂都提供了多个工厂方法用于产生多种不同类型的对象
 * 特点：
 *     横向功能支持简单易行
 *     纵向支持新特性难
 */

package main

import (
	"fmt"
)

// 核心结构：
//     AbstractFactory: 声明一组用于创建对象的方法
//     ConcreteFactory: 实现抽象工厂中的方法
//     AbstractProduct: 每种对象的方法
//     ConcreteProduct: 实现每种对象的方法

type AbstractFactory interface {
	CreateButton() AbstractButton
	CreateSwitch() AbstractSwitch
}

type Windows struct{}

func (f *Windows) CreateButton() AbstractButton {
	return new(WindowsButton)
}

func (f *Windows) CreateSwitch() AbstractSwitch {
	return new(WindowsSwitch)
}

type MacOS struct{}

func (f *MacOS) CreateButton() AbstractButton {
	return new(MacOSButton)
}

func (f *MacOS) CreateSwitch() AbstractSwitch {
	return new(MacOSSwitch)
}

type AbstractButton interface{}
type AbstractSwitch interface{}

type WindowsButton struct{}
type WindowsSwitch struct{}
type MacOSButton struct{}
type MacOSSwitch struct{}

func main() {
	windows := new(Windows)
	button1 := windows.CreateButton()
	switch1 := windows.CreateSwitch()
	fmt.Printf("%T\n", button1)
	fmt.Printf("%T\n", switch1)

	mac := new(MacOS)
	button2 := mac.CreateButton()
	switch2 := mac.CreateSwitch()
	fmt.Printf("%T\n", button2)
	fmt.Printf("%T\n", switch2)
}
