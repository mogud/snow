package container

var _ = Iterator[struct{}]((Set[struct{}])(nil))

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](args ...T) Set[T] {
	var set = Set[T]{}
	for _, arg := range args {
		set[arg] = struct{}{}
	}
	return set
}

func (set Set[T]) ScanIf(fn func(elem T) bool) {
	for k := range set {
		if !fn(k) {
			break
		}
	}
}

func (set Set[T]) Scan(fn func(elem T)) {
	for k := range set {
		fn(k)
	}
}

func (set Set[T]) Len() int {
	return len(set)
}

func (set Set[T]) IsEmpty() bool {
	return set.Len() == 0
}

func (set Set[T]) Copy() Set[T] {
	newSet := Set[T]{}
	for k := range set {
		newSet[k] = struct{}{}
	}
	return newSet
}

func (set Set[T]) Contains(elem T) bool {
	if _, ok := set[elem]; ok {
		return true
	}
	return false
}

func (set Set[T]) Add(elem T) {
	set[elem] = struct{}{}
}

func (set Set[T]) Remove(elem T) {
	delete(set, elem)
}

func (set *Set[T]) Clear() {
	*set = Set[T]{}
}

func (set Set[T]) Union(iter Iterator[T]) {
	iter.Scan(func(elem T) {
		set[elem] = struct{}{}
	})
}

func (set Set[T]) Intersect(iter Iterator[T]) {
	toDel := make([]T, set.Len())
	rhs := SetOf(iter)
	for key := range set {
		if _, ok := rhs[key]; !ok {
			toDel = append(toDel, key)
		}
	}
	for _, elem := range toDel {
		delete(set, elem)
	}
}

func (set Set[T]) Substract(iter Iterator[T]) {
	iter.Scan(func(key T) {
		delete(set, key)
	})
}

func (set Set[T]) SubstractBy(f func(T) bool) {
	for key := range set {
		if f(key) {
			delete(set, key)
		}
	}
}

func (set Set[T]) Unioned(iter Iterator[T]) Set[T] {
	var result = Set[T]{}
	for key := range set {
		result[key] = struct{}{}
	}
	iter.Scan(func(key T) {
		result[key] = struct{}{}
	})
	return result
}

func (set Set[T]) Intersected(iter Iterator[T]) Set[T] {
	var result = Set[T]{}
	rhs := SetOf(iter)
	for key := range set {
		if _, ok := rhs[key]; ok {
			result.Add(key)
		}
	}
	return result
}

func (set Set[T]) Substracted(iter Iterator[T]) Set[T] {
	var result = Set[T]{}
	rhs := SetOf(iter)
	for key := range set {
		if _, ok := rhs[key]; !ok {
			result.Add(key)
		}
	}
	return result
}

func (set Set[T]) SubstractedBy(pred func(T) bool) Set[T] {
	var result = Set[T]{}
	for key := range set {
		if !pred(key) {
			result.Add(key)
		}
	}
	return result
}

func (set Set[T]) WithKey(key T, fn func(key T)) int {
	if _, ok := set[key]; ok {
		fn(key)
		return 1
	}
	return 0
}
