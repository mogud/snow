package container

var _ = Iterator[Pair[int, struct{}]]((Map[int, struct{}])(nil))

type Map[T comparable, U any] map[T]U

func NewMap[T comparable, U any](args ...Pair[T, U]) Map[T, U] {
	result := Map[T, U]{}
	if len(args) > 0 {
		for _, entry := range args {
			result.Add(entry.First, entry.Second)
		}
	}
	return result
}

func (m Map[T, U]) ScanIf(fn func(entry Pair[T, U]) bool) {
	for k, v := range m {
		if !fn(Pair[T, U]{k, v}) {
			break
		}
	}
}

func (m Map[T, U]) Scan(fn func(entry Pair[T, U])) {
	for k, v := range m {
		fn(Pair[T, U]{k, v})
	}
}

func (m Map[T, U]) ScanKVIf(fn func(k T, v U) bool) {
	for k, v := range m {
		if !fn(k, v) {
			break
		}
	}
}

func (m Map[T, U]) ScanKV(fn func(k T, v U)) {
	for k, v := range m {
		fn(k, v)
	}
}

func (m Map[T, U]) Len() int {
	return len(m)
}

func (m Map[T, U]) IsEmpty() bool {
	return m.Len() == 0
}

func (m Map[T, U]) Copy() Map[T, U] {
	newMap := Map[T, U]{}
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func (m Map[T, U]) Contains(key T) bool {
	if _, ok := m[key]; ok {
		return true
	}
	return false
}

func (m Map[T, U]) Get(key T) (U, bool) {
	v, ok := m[key]
	return v, ok
}

func (m Map[T, U]) Add(key T, value U) {
	m[key] = value
}

func (m Map[T, U]) AddAll(iter Iterator[Pair[T, U]]) {
	iter.Scan(func(entry Pair[T, U]) {
		m.Add(entry.First, entry.Second)
	})
}

func (m Map[T, U]) Remove(key T) {
	delete(m, key)
}

func (m Map[T, U]) RemoveAll(iter Iterator[T]) {
	iter.Scan(func(key T) {
		m.Remove(key)
	})
}

func (m *Map[T, U]) Clear() {
	*m = Map[T, U]{}
}

func (m Map[T, U]) Keys() List[T] {
	result := make(List[T], 0, m.Len())
	m.ScanKV(func(k T, v U) { result.Add(k) })
	return result
}

func (m Map[T, U]) Values() List[U] {
	result := make(List[U], 0, m.Len())
	m.ScanKV(func(k T, v U) { result.Add(v) })
	return result
}

func (m Map[T, U]) WithKey(key T, fn func(key T, value U)) int {
	if v, ok := m[key]; ok {
		fn(key, v)
		return 1
	}
	return 0
}
