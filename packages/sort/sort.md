## sort

排序,是我们日常写业务逻辑的时候,经常能遇到的问题. 如果针对简单的 slice, 如 `[]int`, `[]string` 等,只需直接调用 `sort` 包的 `sort.Sort()` 即可.但是针对复杂的 slice, 进行排序时,很多时候可能会束手无策的,下面通过一个例子来讲解.
首先,想自定义排序的话,我们需要实现以下三个方法
``` go
// A type, typically a collection, that satisfies sort.Interface can be
// sorted by the routines in this package. The methods require that the
// elements of the collection be enumerated by an integer index.
type Interface interface {
	// Len is the number of elements in the collection.
	Len() int
	// Less reports whether the element with
	// index i should sort before the element with index j.
	Less(i, j int) bool
	// Swap swaps the elements with indexes i and j.
	Swap(i, j int)
}
```

以下是一个自定义的结构体,如果想根据其中的每一条字段,进行不同的排序 `[]*Vendor`的怎么实现呢?

```go
type Vendor struct {
    Id          int
    Name        string
    CreatedAt   int64
}
```

``` go
type VendorListSort struct {
	Vendors []*Vendor
	By      func(p, q *models.Vendor) bool
}

func (v VendorListSort) Len() int {
	return len(v.Vendors)
}

func (v VendorListSort) Less(i, j int) bool {
	return v.By(v.Vendors[i], v.Vendors[j])
}

func (v VendorListSort) Swap(i, j int) {
	v.Vendors[i], v.Vendors[j] = v.Vendors[j], v.Vendors[i]
}

// 关键在于这两项
type VendorSort func(p, q *models.Vendor) bool

// 排序调用此函数即可
func SortVendor(vendors []*models.Vendor, by VendorSort) {
	sort.Sort(VendorListSort{vendors, by})
}
```

真正使用的时候:

``` go 
SortVendor(vendors, func(p, q *models.Vendor) bool {
    // 在此函数内可以尽情的发挥,可以根据不同的字段进行排序,非常方便
	return p.Id < q.Id
})

SortVendor(vendors, func(p, q *models.Vendor) bool {
	return p.Name < q.Name
})

SortVendor(vendors, func(p, q *models.Vendor) bool {
	return p.CreatedAt < q.CreatedAt
})
```