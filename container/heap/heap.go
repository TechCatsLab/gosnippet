/*
src/container/heap/heap.go
*/

package heap

import "sort"


type Interface interface {
	sort.Interface
	Push(x interface{}) // 往堆里放一个元素
	Pop() interface{}   // 从堆里删除最小的元素
}

// 初始化堆
func Init(h Interface) {
	// 堆化
	n := h.Len()
	for i := n/2 - 1; i >= 0; i-- {
		down(h, i, n)
	}
}

// 把元素 x 放到堆中
// 复杂度为 O(log(n)) 其中 n = h.Len().
//
func Push(h Interface, x interface{}) {
	h.Push(x)
	up(h, h.Len()-1)
}

// 从堆中删除最小(即堆的根元素)的元素并且返回这个元素
// 复杂度为 O(log(n)) 其中 n = h.Len().
// Pop() 等同于 Remove(h, 0).
//
func Pop(h Interface) interface{} {
	n := h.Len() - 1
	// 根元素和最后一个元素交换
	h.Swap(0, n)
	// 新的根元素往下移
	down(h, 0, n)
	return h.Pop()
}

// Remove 删除堆中索引为 i 的元素
// 复杂度为 O(log(n)) 其中 n = h.Len().
//
func Remove(h Interface, i int) interface{} {
	n := h.Len() - 1
	if n != i {
		h.Swap(i, n)
		down(h, i, n)
		up(h, i)
	}
	return h.Pop()
}

// 当索引为 i 的元素值发生改变时，重新堆化 .
// 修改索引为 i 的元素的值并调用 Fix() 函数等同于给堆里插入新元素并调用 Remove(h, i) 函数，但是前者效率更高
// Changing the value of the element at index i and then calling Fix is equivalent to,
// but less expensive than, calling Remove(h, i) followed by a Push of the new value.
// 复杂度为 O(log(n)) 其中 n = h.Len().
func Fix(h Interface, i int) {
	if !down(h, i, h.Len()) {
		up(h, i)
	}
}

func up(h Interface, j int) {
	for {
		i := (j - 1) / 2 // 父节点
		// h.Less(i,j) 如果索引为 i 的元素值小于索引为 j 的元素值，则返回 true .
		if i == j || !h.Less(j, i) {
			break
		}
		// 交换索引为 i, j 的元素
		h.Swap(i, j)
		j = i
	}
}

// 把 i0 元素往下移并放到合适的(适合堆排序原则)位置
func down(h Interface, i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 溢出
			break
		}
		j := j1 // 左孩子
		if j2 := j1 + 1; j2 < n && !h.Less(j1, j2) {
			j = j2 // = 2*i + 2  // 右孩子
		}
		if !h.Less(j, i) {
			break
		}
		h.Swap(i, j)
		i = j
	}
	return i > i0
}
