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
 *     Initial: 2017/04/05        Tang Xiaoji
 */

package main

import "fmt"

//切片相当于动态数组
func main() {
	var slice1 = make([]int, 3, 5)
	fmt.Println("定义一个长度为3，容量为5的切片", slice1)

	slice2 := []int{1, 2, 3, 4, 5, 6}
	fmt.Println("定义并初始化一个具有三个元素的切片", slice2)

	arr := [5]int{1, 2, 3, 4, 5}
	slice3 := arr[2:3]
	fmt.Println("应用数组的一部分作为切片", slice3)

	subSlice := slice2[2:4]
	fmt.Println("从上限2（包含）到下限4（不包含）截取切片slice2: ", subSlice)

	fmt.Println("slice1 切片的长度是：", len(slice1))

	var nilSlice []int
	fmt.Println("定义一个切片为初始化时为空（nil）切片，长度为0：", "nilSlice:", nilSlice, ", length:", len(nilSlice))

	//切片的容量是可增加的，增加的过程为先创建一个新的更大的切片，再把原切片的内容复制过去
	slice1 = append(slice1, 2, 3)
	fmt.Println("向slice1切片注入新元素：", slice1)

	largerSlice := make([]int, len(slice1), cap(slice1)*2)
	fmt.Println("创建slice1两倍容量的切片：", largerSlice, "容量为：", cap(largerSlice))

	copy(largerSlice, slice1)
	fmt.Println("将slice1中的内容拷贝至largerSlice中：", largerSlice, "容量为：", cap(largerSlice))
}
