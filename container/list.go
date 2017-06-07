/*
 * Revision History:
 *     Translated: 2017/06/06        Yusan Kurban
 */

package list

// 链表元素
type Element struct {
	// Next and previous pointers in the doubly-linked list of elements
	// next 和 prev 是链表中每一个元素的指向先一个和指向上一个的指针
	// 在链表 l 中 &l.root 既是最后一元素(l.Back())的下一个元素，也是第一个元素(l.Front())的上一个元素
	// 这一点与环形队列十分相似.
	// To simplify the implementation, internally a list l is implemented
	// as a ring, such that &l.root is both the next element of the last
	// list element (l.Back()) and the previous element of the first list
	// element (l.Front()).
	next, prev *Element

	// 指向该元素所属于的链表.
	list *List

	// 元素值与元素一起存着.
	Value interface{}
}

// Next 返回下一个链表元素或空值
func (e *Element) Next() *Element {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// Prev 返回上一个链表元素或空值.
func (e *Element) Prev() *Element {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// List 代表着双重连接的链表.
// 空值的 List 是用来代表空的链表.
// The zero value for List is an empty list ready to use.
type List struct {
	root Element // 链表的哨兵元素，只用 &root, root.prev, and root.next
	len  int     // 当前链表长度，但不包括 root 元素
}

// 初始化链表或清空链表
func (l *List) Init() *List {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

// New 返回一个初始化过的链表.
func New() *List { return new(List).Init() }

// Len 返回链表 l 元素数量.
// 复杂度为 O(1).
func (l *List) Len() int { return l.len }

// Front 返回链表的首元素或空.
func (l *List) Front() *Element {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// Back 返回链表的尾元素或空.
func (l *List) Back() *Element {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// lazyInit 初始化一个空的链表.
func (l *List) lazyInit() {
	if l.root.next == nil {
		l.Init()
	}
}

// insert 把 e 插入到 at 后面，并返回它.
func (l *List) insert(e, at *Element) *Element {
	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
	e.list = l
	l.len++
	return e
}

// insertValue 是对 insert(&Element{Value: v}, at) 的封装.
func (l *List) insertValue(v interface{}, at *Element) *Element {
	return l.insert(&Element{Value: v}, at)
}

// remove 把元素 e 从链表中删除并缩减链表长度，返回 e .
func (l *List) remove(e *Element) *Element {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // 避免内存泄漏
	e.prev = nil // 避免内存泄漏
	e.list = nil
	l.len--
	return e
}

// Remove 如果 e 是链表 l 的元素，则把 e 删除 .
// 它返回 e 的值.
func (l *List) Remove(e *Element) interface{} {
	if e.list == l {
		// 如果 l 是空链表(e 是空元素), l.remove 会崩溃
		l.remove(e)
	}
	return e.Value
}

// PushFront 把值为 v 的新元素 e 插到链表的最前面并返回 e .
func (l *List) PushFront(v interface{}) *Element {
	l.lazyInit()
	// 即插入到 root 元素的后面
	return l.insertValue(v, &l.root)
}

// PushBack 把值为 v 的新元素 e 插入到链表最后并返回 e .
func (l *List) PushBack(v interface{}) *Element {
	l.lazyInit()
	return l.insertValue(v, l.root.prev)
}

// InsertBefore 把值为 v 的新元素 e 插入到 mark 元素之前并返回 e .
// 如果 mark 不是链表 l 的元素，则 l 不会被修改 .
func (l *List) InsertBefore(v interface{}, mark *Element) *Element {
	if mark.list != l {
		return nil
	}
	// 没有初始化的过程
	return l.insertValue(v, mark.prev)
}

// InsertAfter 把值为 v 的新元素 e 插入到 mark 元素之后并返回 e .
// 如果 mark 不是链表 l 的元素，则 l 不会被修改 .
func (l *List) InsertAfter(v interface{}, mark *Element) *Element {
	if mark.list != l {
		return nil
	}
	// 没有初始化的过程
	return l.insertValue(v, mark)
}

// MoveToFront 把元素 e 移到链表 l 最前面 .
// 如果元素 e 不属于链表 l，则 l 不会被修改 .
func (l *List) MoveToFront(e *Element) {
	// 如果 e 不属于 l 或 e 已经在第一个位置了，则直接返回 .
	if e.list != l || l.root.next == e {
		return
	}
	// 没有初始化的过程
	l.insert(l.remove(e), &l.root)
}

// MoveToBack 把元素 e 移到链表 l 最后面 .
// 如果元素 e 不属于链表 l，则 l 不会被修改 .
func (l *List) MoveToBack(e *Element) {
	// 如果 e 不属于 l 或 e 已经在最后的位置了，则直接返回 .
	if e.list != l || l.root.prev == e {
		return
	}
	// 没有初始化的过程
	l.insert(l.remove(e), l.root.prev)
}

// MoveBefore 把元素 e 移到 mark 元素之前的位置 .
// 如果 e 或 l 不是链表 l 的元素，或者 e == mark，则 l 不会被修改 .
func (l *List) MoveBefore(e, mark *Element) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.insert(l.remove(e), mark.prev)
}

// MoveAfter 把元素 e 移到 mark 元素之后的位置 .
// 如果 e 或 l 不是链表 l 的元素，或者 e == mark，则 l 不会被修改 .
func (l *List) MoveAfter(e, mark *Element) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.insert(l.remove(e), mark)
}

// PushBackList 把 other 链表插入到链表 l 的后面 .
// 链表 l 和 other 可能是同一个链表 .
func (l *List) PushBackList(other *List) {
	// 如果 l 为空，则初始化 .
	l.lazyInit()
	// i， e 分别为 other 的长度和首元素，每次循环都从 other 头部开始拿一个元素插到 l 的最后(即 l.root.rev)
	for i, e := other.Len(), other.Front(); i > 0; i, e = i-1, e.Next() {
		l.insertValue(e.Value, l.root.prev)
	}
}

// PushFrontList 把 other 链表插入到链表 l 的后面 .
// 链表 l 和 other 可能是同一个链表 .
func (l *List) PushFrontList(other *List) {
	l.lazyInit()
	for i, e := other.Len(), other.Back(); i > 0; i, e = i-1, e.Prev() {
		l.insertValue(e.Value, &l.root)
	}
}
