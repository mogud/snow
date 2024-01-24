package container

import (
	"gitee.com/mogud/snow/core/constraints"
	"github.com/tidwall/btree"
)

var _ = Iterator[int]((*OrderedSet[int])(nil))

type OrderedSet[T constraints.Ordered] struct {
	base btree.Set[T]
}

func NewOrderedSet[T constraints.Ordered](args ...T) *OrderedSet[T] {
	result := &OrderedSet[T]{}
	if len(args) > 0 {
		argList := List[T](args)
		Sort(argList)
		for _, elem := range argList {
			result.base.Load(elem)
		}
	}
	return result
}

func (set *OrderedSet[T]) ScanIf(fn func(elem T) bool) {
	set.base.Scan(fn)
}

func (set *OrderedSet[T]) Scan(fn func(elem T)) {
	set.base.Scan(func(key T) bool {
		fn(key)
		return true
	})
}

func (set *OrderedSet[T]) ScanIV(fn func(i int, elem T)) {
	i := 0
	set.base.Scan(func(key T) bool {
		fn(i, key)
		i++
		return true
	})
}

func (set *OrderedSet[T]) Len() int {
	return set.base.Len()
}

func (set *OrderedSet[T]) IsEmpty() bool {
	return set.Len() == 0
}

func (set *OrderedSet[T]) Copy() *OrderedSet[T] {
	newSet := &OrderedSet[T]{}
	if set.base.Len() > 0 {
		newBase := newSet.base
		set.base.Scan(func(key T) bool {
			newBase.Load(key)
			return true
		})
	}
	return newSet
}

func (set *OrderedSet[T]) Contains(elem T) bool {
	return set.base.Contains(elem)
}

func (set *OrderedSet[T]) Add(elem T) {
	set.base.Insert(elem)
}

func (set *OrderedSet[T]) Remove(elem T) {
	set.base.Delete(elem)
}

func (set *OrderedSet[T]) Clear() {
	set.base = btree.Set[T]{}
}

func (set *OrderedSet[T]) Union(rhs *OrderedSet[T]) {
	rhs.base.Scan(func(elem T) bool {
		set.Add(elem)
		return true
	})
}

func (set *OrderedSet[T]) Intersect(rhs *OrderedSet[T]) {
	toDel := make([]T, set.Len())
	set.base.Scan(func(elem T) bool {
		if !rhs.Contains(elem) {
			toDel = append(toDel, elem)
		}
		return true
	})
	for _, elem := range toDel {
		set.Remove(elem)
	}
}

func (set *OrderedSet[T]) Substract(rhs *OrderedSet[T]) {
	rhs.base.Scan(func(elem T) bool {
		set.Remove(elem)
		return true
	})
}

func (set *OrderedSet[T]) Unioned(rhs *OrderedSet[T]) *OrderedSet[T] {
	var result = &OrderedSet[T]{}
	set.base.Scan(func(elem T) bool {
		result.Add(elem)
		return true
	})
	rhs.base.Scan(func(elem T) bool {
		result.Add(elem)
		return true
	})
	return result
}

func (set *OrderedSet[T]) Intersected(rhs *OrderedSet[T]) *OrderedSet[T] {
	var result = &OrderedSet[T]{}
	set.base.Scan(func(elem T) bool {
		if rhs.Contains(elem) {
			result.Add(elem)
		}
		return true
	})
	return result
}

func (set *OrderedSet[T]) Substracted(rhs *OrderedSet[T]) *OrderedSet[T] {
	var result = &OrderedSet[T]{}
	set.base.Scan(func(elem T) bool {
		if !rhs.Contains(elem) {
			result.Add(elem)
		}
		return true
	})
	return result
}

func (m *OrderedSet[T]) WithKey(key T, fn func(key T)) int {
	if m.base.Contains(key) {
		fn(key)
		return 1
	}
	return 0
}

// TODO
func (m *OrderedSet[T]) WithKeyRange(keyLow T, keyHigh T, fn func(key T)) (cnt int) {
	for i := 0; i < m.Len(); i++ {
		if k, ok := m.base.GetAt(i); ok && keyLow <= k && k <= keyHigh {
			fn(k)
			cnt++
		}
	}
	return
	// TODO FIXME
	// NOTE waiting https://github.com/tidwall/btree/issues/12
	// indexLow, okLow := m.Find(KeyLow)
	// if okHight {
	// 	indexHigh, okHight := m.Find(KeyHigh)
	// 	for i := indexLow; i <= indexHigh; i++ {
	// 		if k, OK := m.GetAt(I); OK {
	// 			fn(k)
	// 			cnt++
	// 		}
	// 	}
	// }
	// return cnt
}

func (m *OrderedSet[T]) GetAt(key int) (value T, err bool) {
	return m.base.GetAt(key)
}
