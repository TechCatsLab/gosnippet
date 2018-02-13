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
 * 工厂方法模式：
 *     定义一个用于创建对象的接口，让子类决定将哪一个类实例化
 * 特点：
 *     工厂方法模式让一个类的实例化延迟到其子类
 */

package main

import (
	"fmt"
)

// 抽象工厂方法接口
type Factory interface {
	factoryMethod() Product
}

// 要调用的抽象方法
type Product interface {
	method()
}

// 对象创建者
type Creator struct {
	factory Factory
}

// 对外暴露的接口
func (c *Creator) Operation() {
	product := c.factory.factoryMethod()
	product.method()
}

// 具体工厂创建具体产品，且只创建一种产品
type ConcreteCreator struct{}

func (c *ConcreteCreator) factoryMethod() Product {
	return new(ConcreteProduct)
}

// 具体产品
type ConcreteProduct struct{}

func (p *ConcreteProduct) method() {
	fmt.Println("ConcreteProduct.method()")
}

func main() {
	creator := Creator{new(ConcreteCreator)}
	creator.Operation()
}
