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
 * 简单工厂模式：
 *     定义一个工厂类，根据传入的参数不同返回不同的实例，被创建的实例具有共同的父类或接口
 * 特点：
 *     需要创建的对象不多
 *     对象种类不会频繁变更
 *     模块不关注对象创建过程
 */

package main

import (
	"fmt"
)

// 接口
type Shape interface {
	Draw()
}

// Circle
type Circle struct{}

func (this *Circle) Draw() {
	fmt.Println("Circle drawing")
}

// Rectangle
type Rectangle struct{}

func (this *Rectangle) Draw() {
	fmt.Println("Rectangle drawing")
}

// Simple Factory
type SimpleFactory struct{}

func (this *SimpleFactory) CreateShape(shape string) Shape {
	var (
		result Shape
	)

	switch shape {
	case "circle":
		result = &Circle{}

	case "rectangle":
		result = &Rectangle{}
	}

	return result
}

func main() {
	factory := &SimpleFactory{}

	factory.CreateShape("circle").Draw()
	factory.CreateShape("rectangle").Draw()
}
