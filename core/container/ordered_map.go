package container

import (
	"gitee.com/mogud/snow/core/stdext/constraints"
	"github.com/tidwall/btree"
)

var _ = Iterator[Pair[int, int]]((*OrderedMap[int, int])(nil))

type OrderedMap[T constraints.Ordered, U any] struct {
	base btree.Map[T, U]
}

func NewOrderedMap[T constraints.Ordered, U any](args ...Pair[T, U]) *OrderedMap[T, U] {
	result := &OrderedMap[T, U]{}
	if len(args) > 0 {
		argList := List[Pair[T, U]](args)
		SortBy(argList, func(lhs, rhs Pair[T, U]) bool { return lhs.First < rhs.First })
		for _, entry := range argList {
			result.base.Load(entry.First, entry.Second)
		}
	}
	return result
}

func (m *OrderedMap[T, U]) ScanIf(fn func(entry Pair[T, U]) bool) {
	m.base.Scan(func(key T, value U) bool {
		return fn(Pair[T, U]{key, value})
	})
}

func (m *OrderedMap[T, U]) Scan(fn func(entry Pair[T, U])) {
	m.base.Scan(func(key T, value U) bool {
		fn(Pair[T, U]{key, value})
		return true
	})
}

func (m *OrderedMap[T, U]) ScanKVIf(fn func(key T, value U) bool) {
	m.base.Scan(func(key T, value U) bool {
		return fn(key, value)
	})
}

func (m *OrderedMap[T, U]) ScanKV(fn func(key T, value U)) {
	m.base.Scan(func(key T, value U) bool {
		fn(key, value)
		return true
	})
}

func (m *OrderedMap[T, U]) ScanIKV(fn func(index int, key T, value U)) {
	i := 0
	m.base.Scan(func(key T, value U) bool {
		fn(i, key, value)
		i++
		return true
	})
}

func (m *OrderedMap[T, U]) Len() int {
	return m.base.Len()
}

func (m *OrderedMap[T, U]) IsEmpty() bool {
	return m.Len() == 0
}

func (m *OrderedMap[T, U]) Copy() *OrderedMap[T, U] {
	newMap := &OrderedMap[T, U]{}
	if m.base.Len() > 0 {
		newBase := newMap.base
		m.base.Scan(func(key T, value U) bool {
			newBase.Load(key, value)
			return true
		})
	}
	return newMap
}

func (m *OrderedMap[T, U]) Contains(key T) bool {
	if _, ok := m.base.Get(key); ok {
		return true
	}
	return false
}

func (m *OrderedMap[T, U]) Get(key T) (U, bool) {
	return m.base.Get(key)
}

func (m *OrderedMap[T, U]) Add(key T, value U) {
	m.base.Set(key, value)
}

func (m *OrderedMap[T, U]) AddAll(iter Iterator[Pair[T, U]]) {
	iter.Scan(func(entry Pair[T, U]) {
		m.Add(entry.First, entry.Second)
	})
}

func (m *OrderedMap[T, U]) Remove(key T) {
	m.base.Delete(key)
}

func (m *OrderedMap[T, U]) RemoveAll(iter Iterator[T]) {
	iter.Scan(func(key T) {
		m.Remove(key)
	})
}

func (m *OrderedMap[T, U]) Clear() {
	*m = OrderedMap[T, U]{}
}

func (m *OrderedMap[T, U]) Keys() List[T] {
	result := make(List[T], 0, m.Len())
	m.ScanKV(func(k T, v U) { result.Add(k) })
	return result
}

func (m *OrderedMap[T, U]) Values() List[U] {
	result := make(List[U], 0, m.Len())
	m.ScanKV(func(k T, v U) { result.Add(v) })
	return result
}

func (m *OrderedMap[T, U]) WithKey(key T, fn func(key T, value U)) int {
	if v, ok := m.base.Get(key); ok {
		fn(key, v)
		return 1
	}
	return 0
}

// TODO
func (m *OrderedMap[T, U]) WithKeyRange(keyLow T, keyHigh T, fn func(key T, value U)) (cnt int) {
	for i := 0; i < m.Len(); i++ {
		if k, v, ok := m.base.GetAt(i); ok && keyLow <= k && k <= keyHigh {
			fn(k, v)
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
	// 		if k, v, OK := m.GetAt(I); OK {
	// 			fn(k, v)
	// 			cnt++
	// 		}
	// 	}
	// }
	// return cnt
}
