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
 *     Initial: 2017/04/26        Feng Yifei
 */

package main

import (
	"fmt"
	"strconv"
)

// 产品
type Fruit struct {
	name  string
	price int
}

func (p *Fruit) String() string {
	return "Fruit [name=" + p.name + ", price=" + strconv.Itoa(p.price) + "]"
}

// 指挥者
type Director struct {
	builder Builder
}

// 构建
func (d *Director) Construct() *Fruit {
	d.builder.SetName()
	d.builder.SetPrice()
	return d.builder.GetResult()
}

// 抽象构建
type Builder interface {
	SetName()
	SetPrice()
	GetResult() *Fruit
}

type AppleBuilder struct {
	product *Fruit
}

func NewAppleBuilder() *AppleBuilder {
	return &AppleBuilder{new(Fruit)}
}

func (b *AppleBuilder) SetName() {
	b.product.name = "apple"
}

func (b *AppleBuilder) SetPrice() {
	b.product.price = 10
}

func (b *AppleBuilder) GetResult() *Fruit {
	return b.product
}

type OrangeBuilder struct {
	product *Fruit
}

func NewOrangeBuilder() *OrangeBuilder {
	return &OrangeBuilder{new(Fruit)}
}

func (b *OrangeBuilder) SetName() {
	b.product.name = "orange"
}

func (b *OrangeBuilder) SetPrice() {
	b.product.price = 20
}

func (b *OrangeBuilder) GetResult() *Fruit {
	return b.product
}

func main() {
	appleDirector := &Director{NewAppleBuilder()}
	apple := appleDirector.Construct()
	fmt.Println(apple)

	orangeDirector := &Director{NewOrangeBuilder()}
	orange := orangeDirector.Construct()
	fmt.Println(orange)
}
