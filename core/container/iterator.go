package container

import (
	"snow/core/stdext/constraints"
)

type Iterator[T any] interface {
	ScanIf(fn func(elem T) bool)
	Scan(fn func(elem T))
	Len() int
}

func Any[T any](iter Iterator[T], fn func(elem T) bool) bool {
	result := false
	iter.ScanIf(func(elem T) bool {
		if fn(elem) {
			result = true
			return false
		}
		return true
	})
	return result
}

func All[T any](iter Iterator[T], fn func(elem T) bool) bool {
	result := true
	iter.ScanIf(func(elem T) bool {
		if !fn(elem) {
			result = false
			return false
		}
		return true
	})
	return result
}

func ScanWithIndex[T any](iter Iterator[T], fn func(index int, elem T)) {
	var i int
	iter.Scan(func(elem T) {
		fn(i, elem)
		i++
	})
}

func ScanIfWithIndex[T any](iter Iterator[T], fn func(index int, elem T) bool) {
	var i int
	iter.ScanIf(func(elem T) bool {
		if !fn(i, elem) {
			return false
		}
		i++
		return true
	})
}

func Fold[T, U any](iter Iterator[T], init U, fn func(acc U, elem T) U) U {
	var acc = init
	iter.Scan(func(elem T) {
		acc = fn(acc, elem)
	})
	return acc
}

func Trans[T, U any](iter Iterator[T], fn func(elem T) U) List[U] {
	list := make(List[U], 0, max(iter.Len(), 4))
	iter.Scan(func(elem T) {
		list.Add(fn(elem))
	})
	return list
}

func Filter[T any](iter Iterator[T], fn func(elem T) bool) List[T] {
	list := make(List[T], 0, max(iter.Len(), 4))
	return Fold(iter, list, func(acc List[T], elem T) List[T] {
		if fn(elem) {
			acc.Add(elem)
		}
		return acc
	})
}

func FlatTrans[T, U any](iter Iterator[T], fn func(elem T) Iterator[U]) List[U] {
	list := make(List[U], 0, max(iter.Len(), 4))
	return Fold(iter, list, func(acc List[U], elemRaw T) List[U] {
		fn(elemRaw).Scan(func(elem U) {
			acc.Add(elem)
		})
		return acc
	})
}

func GroupBy[T any, U comparable, V any](iter Iterator[T], keyFn func(T) U, valueFn func(T) V) Map[U, List[V]] {
	m := Map[U, List[V]]{}
	iter.Scan(func(elem T) {
		key := keyFn(elem)
		if m.Contains(key) {
			v := m[key]
			v.Add(valueFn(elem))
			m[key] = v
		} else {
			m.Add(key, List[V]{valueFn(elem)})
		}
	})
	return m
}

func Zip[T, U any](iterA Iterator[T], iterB Iterator[U]) List[Pair[T, U]] {
	list := ListOf(iterA)
	result := make(List[Pair[T, U]], 0, list.Len())
	ScanIfWithIndex(iterB, func(index int, elem U) bool {
		if index >= list.Len() {
			return false
		}

		result.Add(Pair[T, U]{list[index], elem})
		return true
	})
	return result
}

func ListOf[T any](iter Iterator[T]) List[T] {
	list := make(List[T], 0, max(iter.Len(), 4))
	iter.Scan(func(elem T) {
		list.Add(elem)
	})
	return list
}

func SetOf[T comparable](iter Iterator[T]) Set[T] {
	set := Set[T]{}
	iter.Scan(func(elem T) {
		set.Add(elem)
	})
	return set
}

func OrderedSetOf[T constraints.Ordered](iter Iterator[T]) *OrderedSet[T] {
	set := &OrderedSet[T]{}
	iter.Scan(func(elem T) {
		set.Add(elem)
	})
	return set
}

func MapOf[T comparable, U any](iter Iterator[Pair[T, U]]) Map[T, U] {
	m := Map[T, U]{}
	iter.Scan(func(pair Pair[T, U]) {
		m.Add(pair.First, pair.Second)
	})
	return m
}

func OrderedMapOf[T constraints.Ordered, U any](iter Iterator[Pair[T, U]]) *OrderedMap[T, U] {
	m := &OrderedMap[T, U]{}
	iter.Scan(func(pair Pair[T, U]) {
		m.Add(pair.First, pair.Second)
	})
	return m
}

func MapBy[T comparable, U any](iter Iterator[T], fn func(elem T) U) Map[T, U] {
	m := Map[T, U]{}
	iter.Scan(func(elem T) {
		m.Add(elem, fn(elem))
	})
	return m
}

func OrderedMapBy[T constraints.Ordered, U any](iter Iterator[T], fn func(elem T) U) *OrderedMap[T, U] {
	m := &OrderedMap[T, U]{}
	iter.Scan(func(elem T) {
		m.Add(elem, fn(elem))
	})
	return m
}
