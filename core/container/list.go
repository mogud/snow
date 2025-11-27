package container

import (
	"github.com/mogud/snow/core/constraints"
	"math/rand/v2"
)

var _ = Iterator[struct{}]((List[struct{}])(nil))

type List[T any] []T

func NewList[T any](args ...T) List[T] {
	result := make(List[T], len(args))
	copy(result, args)
	return result
}

func (list List[T]) ScanIf(fn func(elem T) bool) {
	for _, v := range list {
		if !fn(v) {
			break
		}
	}
}

func (list List[T]) Scan(fn func(elem T)) {
	for _, v := range list {
		fn(v)
	}
}

func (list List[T]) ScanIV(fn func(index int, elem T)) {
	for i, v := range list {
		fn(i, v)
	}
}

func (list List[T]) Len() int {
	return len(list)
}

func (list List[T]) IsEmpty() bool {
	return list.Len() == 0
}

func (list List[T]) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list List[T]) Copy() List[T] {
	newList := make(List[T], list.Len())
	copy(newList, list)
	return newList
}

func (list *List[T]) Add(elem T) {
	*list = append(*list, elem)
}

func (list *List[T]) AddIndex(index int, elem T) {
	*list = append(*list, elem)
	copy((*list)[index+1:], (*list)[index:])
	(*list)[index] = elem
}

func (list *List[T]) RemoveLast() {
	*list = (*list)[:list.Len()-1]
}

func (list *List[T]) RemoveIndex(index int) {
	*list = append((*list)[:index], (*list)[index+1:]...)
}

func (list *List[T]) RemoveSwap(index int) {
	(*list)[index] = (*list)[list.Len()-1]
	*list = (*list)[:list.Len()-1]
}

func (list *List[T]) RemoveIf(fn func(elem T) bool) {
	for i := 0; i < list.Len(); {
		if fn((*list)[i]) {
			list.RemoveIndex(i)
		} else {
			i++
		}
	}
}

func (list *List[T]) Clear() {
	*list = (*list)[:0]
}

func (list List[T]) Shuffle() {
	rand.Shuffle(list.Len(), func(i, j int) { list.Swap(i, j) })
}

func (list List[T]) Shuffled() List[T] {
	result := list.Copy()
	result.Shuffle()
	return result
}

func ListRemove[T comparable](list *List[T], elem T) {
	for i := 0; i < list.Len(); {
		if (*list)[i] == elem {
			list.RemoveIndex(i)
		} else {
			i++
		}
	}
}

func ListSearch[T comparable](list List[T], elem T) int {
	for idx, value := range list {
		if value == elem {
			return idx
		}
	}
	return -1
}

func ListContains[T comparable](list List[T], elem T) bool {
	return ListSearch(list, elem) != -1
}

func ListBinarySearch[T constraints.Ordered](list List[T], elem T) int {
	var binarySearch func(a []T, search T) int
	binarySearch = func(a []T, search T) int {
		mid := len(a) / 2
		switch {
		case len(a) == 0:
			return -1
		case a[mid] > search:
			return binarySearch(a[:mid], search)
		case a[mid] < search:
			result := binarySearch(a[mid+1:], search)
			if result >= 0 {
				result += mid + 1
			}
			return result
		default:
			return mid
		}
	}
	return binarySearch(list, elem)
}

func ListBinaryContains[T constraints.Ordered](list List[T], elem T) bool {
	return ListBinarySearch(list, elem) != -1
}

func SortBy[T any](list List[T], fn func(lhs, rhs T) bool) {
	less := func(i, j int) bool { return fn(list[i], list[j]) }
	swap := list.Swap
	length := list.Len()
	sortQuick(sortLessSwap{less, swap}, 0, length, sortMaxDepth(length))
}

func Sort[T constraints.Ordered](list List[T]) {
	SortBy(list, func(lhs, rhs T) bool { return lhs < rhs })
}

func SortStableBy[T any](list List[T], fn func(lhs, rhs T) bool) {
	less := func(i, j int) bool { return fn(list[i], list[j]) }
	swap := list.Swap
	length := list.Len()
	sortStable(sortLessSwap{less, swap}, length)
}

func SortStable[T constraints.Ordered](list List[T]) {
	SortStableBy(list, func(lhs, rhs T) bool { return lhs < rhs })
}

func SortedBy[T any](list List[T], fn func(lhs, rhs T) bool) List[T] {
	result := list.Copy()
	less := func(i, j int) bool { return fn(result[i], result[j]) }
	swap := result.Swap
	length := result.Len()
	sortQuick(sortLessSwap{less, swap}, 0, length, sortMaxDepth(length))
	return result
}

func Sorted[T constraints.Ordered](list List[T]) List[T] {
	return SortedBy(list, func(lhs, rhs T) bool { return lhs < rhs })
}

func SortedStableBy[T any](list List[T], fn func(lhs, rhs T) bool) List[T] {
	result := list.Copy()
	less := func(i, j int) bool { return fn(result[i], result[j]) }
	swap := result.Swap
	length := result.Len()
	sortStable(sortLessSwap{less, swap}, length)
	return result
}

func SortedStable[T constraints.Ordered](list List[T]) List[T] {
	return SortedStableBy(list, func(lhs, rhs T) bool { return lhs < rhs })
}

func sortQuick(data sortLessSwap, a, b, maxDepth int) {
	for b-a > 12 {
		if maxDepth == 0 {
			sortHeap(data, a, b)
			return
		}
		maxDepth--
		mlo, mhi := sortDoPivot(data, a, b)
		if mlo-a < b-mhi {
			sortQuick(data, a, mlo, maxDepth)
			a = mhi
		} else {
			sortQuick(data, mhi, b, maxDepth)
			b = mlo
		}
	}
	if b-a > 1 {
		for i := a + 6; i < b; i++ {
			if data.Less(i, i-6) {
				data.Swap(i, i-6)
			}
		}
		sortInsertion(data, a, b)
	}
}

func sortStable(data sortLessSwap, n int) {
	blockSize := 20
	a, b := 0, blockSize
	for b <= n {
		sortInsertion(data, a, b)
		a = b
		b += blockSize
	}
	sortInsertion(data, a, n)
	for blockSize < n {
		a, b = 0, 2*blockSize
		for b <= n {
			sortSymMerge(data, a, a+blockSize, b)
			a = b
			b += 2 * blockSize
		}
		if m := a + blockSize; m < n {
			sortSymMerge(data, a, m, n)
		}
		blockSize *= 2
	}
}

type sortLessSwap struct {
	Less func(i, j int) bool
	Swap func(i, j int)
}

func sortMaxDepth(n int) int {
	var depth int
	for i := n; i > 0; i >>= 1 {
		depth++
	}
	return depth * 2
}

func sortInsertion(data sortLessSwap, a, b int) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && data.Less(j, j-1); j-- {
			data.Swap(j, j-1)
		}
	}
}

func sortShiftDown(data sortLessSwap, lo, hi, first int) {
	root := lo
	for {
		child := 2*root + 1
		if child >= hi {
			break
		}
		if child+1 < hi && data.Less(first+child, first+child+1) {
			child++
		}
		if !data.Less(first+root, first+child) {
			return
		}
		data.Swap(first+root, first+child)
		root = child
	}
}

func sortHeap(data sortLessSwap, a, b int) {
	first := a
	lo := 0
	hi := b - a
	for i := (hi - 1) / 2; i >= 0; i-- {
		sortShiftDown(data, i, hi, first)
	}
	for i := hi - 1; i >= 0; i-- {
		data.Swap(first, first+i)
		sortShiftDown(data, lo, i, first)
	}
}

func sortMedianOfThree(data sortLessSwap, m1, m0, m2 int) {
	if data.Less(m1, m0) {
		data.Swap(m1, m0)
	}
	if data.Less(m2, m1) {
		data.Swap(m2, m1)
		if data.Less(m1, m0) {
			data.Swap(m1, m0)
		}
	}
}

func sortSwapRange(data sortLessSwap, a, b, n int) {
	for i := 0; i < n; i++ {
		data.Swap(a+i, b+i)
	}
}

func sortDoPivot(data sortLessSwap, lo, hi int) (midlo, midhi int) {
	m := int(uint(lo+hi) >> 1)
	if hi-lo > 40 {
		s := (hi - lo) / 8
		sortMedianOfThree(data, lo, lo+s, lo+2*s)
		sortMedianOfThree(data, m, m-s, m+s)
		sortMedianOfThree(data, hi-1, hi-1-s, hi-1-2*s)
	}
	sortMedianOfThree(data, lo, m, hi-1)
	pivot := lo
	a, c := lo+1, hi-1
	for ; a < c && data.Less(a, pivot); a++ {
	}
	b := a
	for {
		for ; b < c && !data.Less(pivot, b); b++ {
		}
		for ; b < c && data.Less(pivot, c-1); c-- {
		}
		if b >= c {
			break
		}
		data.Swap(b, c-1)
		b++
		c--
	}
	protect := hi-c < 5
	if !protect && hi-c < (hi-lo)/4 {
		dups := 0
		if !data.Less(pivot, hi-1) {
			data.Swap(c, hi-1)
			c++
			dups++
		}
		if !data.Less(b-1, pivot) {
			b--
			dups++
		}
		if !data.Less(m, pivot) {
			data.Swap(m, b-1)
			b--
			dups++
		}
		protect = dups > 1
	}
	if protect {
		for {
			for ; a < b && !data.Less(b-1, pivot); b-- {
			}
			for ; a < b && data.Less(a, pivot); a++ {
			}
			if a >= b {
				break
			}
			data.Swap(a, b-1)
			a++
			b--
		}
	}
	data.Swap(pivot, b-1)
	return b - 1, c
}

func sortSymMerge(data sortLessSwap, a, m, b int) {
	if m-a == 1 {
		i := m
		j := b
		for i < j {
			h := int(uint(i+j) >> 1)
			if data.Less(h, a) {
				i = h + 1
			} else {
				j = h
			}
		}
		for k := a; k < i-1; k++ {
			data.Swap(k, k+1)
		}
		return
	}
	if b-m == 1 {
		i := a
		j := m
		for i < j {
			h := int(uint(i+j) >> 1)
			if !data.Less(m, h) {
				i = h + 1
			} else {
				j = h
			}
		}
		for k := m; k > i; k-- {
			data.Swap(k, k-1)
		}
		return
	}
	mid := int(uint(a+b) >> 1)
	n := mid + m
	var start, r int
	if m > mid {
		start = n - b
		r = mid
	} else {
		start = a
		r = m
	}
	p := n - 1
	for start < r {
		c := int(uint(start+r) >> 1)
		if !data.Less(p-c, c) {
			start = c + 1
		} else {
			r = c
		}
	}
	end := n - start
	if start < m && m < end {
		sortRotate(data, start, m, end)
	}
	if a < start && start < mid {
		sortSymMerge(data, a, start, mid)
	}
	if mid < end && end < b {
		sortSymMerge(data, mid, end, b)
	}
}

func sortRotate(data sortLessSwap, a, m, b int) {
	i := m - a
	j := b - m
	for i != j {
		if i > j {
			sortSwapRange(data, m-i, m, j)
			i -= j
		} else {
			sortSwapRange(data, m-i, m+j-i, i)
			j -= i
		}
	}
	sortSwapRange(data, m-i, m, i)
}
