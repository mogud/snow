package container

import (
	"strings"
)

type CaseInsensitiveStringMap[V any] struct {
	keyToValueMap    Map[string, V]
	upperKeyToKeyMap Map[string, string]
}

func NewCaseInsensitiveStringMap[V any]() *CaseInsensitiveStringMap[V] {
	return &CaseInsensitiveStringMap[V]{
		keyToValueMap:    NewMap[string, V](),
		upperKeyToKeyMap: NewMap[string, string](),
	}
}

func (ss *CaseInsensitiveStringMap[V]) Len() int {
	return len(ss.upperKeyToKeyMap)
}

func (ss *CaseInsensitiveStringMap[V]) Contains(key string) bool {
	return ss.upperKeyToKeyMap.Contains(strings.ToUpper(key))
}

func (ss *CaseInsensitiveStringMap[V]) Get(key string) V {
	v, _ := ss.TryGet(key)
	return v
}

func (ss *CaseInsensitiveStringMap[V]) TryGet(key string) (V, bool) {
	upperKey := strings.ToUpper(key)
	if realKey, ok := ss.upperKeyToKeyMap[upperKey]; ok {
		return ss.keyToValueMap[realKey], true
	}
	var res V
	return res, false
}

func (ss *CaseInsensitiveStringMap[V]) Add(key string, value V) {
	upperKey := strings.ToUpper(key)
	if oldKey, ok := ss.upperKeyToKeyMap[upperKey]; ok {
		ss.keyToValueMap.Remove(oldKey)
	}
	ss.keyToValueMap.Add(key, value)
	ss.upperKeyToKeyMap.Add(upperKey, key)
}

func (ss *CaseInsensitiveStringMap[V]) Remove(key string) {
	upperKey := strings.ToUpper(key)
	if oldKey, ok := ss.upperKeyToKeyMap[upperKey]; ok {
		ss.keyToValueMap.Remove(oldKey)
		ss.upperKeyToKeyMap.Remove(upperKey)
	}
}

func (ss *CaseInsensitiveStringMap[V]) Scan(cb func(key string, value V)) {
	for key, value := range ss.keyToValueMap {
		cb(key, value)
	}
}

func (ss *CaseInsensitiveStringMap[V]) ScanFull(cb func(upperKey, key string, value V)) {
	for upperKey, key := range ss.upperKeyToKeyMap {
		cb(upperKey, key, ss.keyToValueMap[key])
	}
}

func (ss *CaseInsensitiveStringMap[V]) ToMap() map[string]V {
	res := make(map[string]V)
	ss.Scan(func(key string, value V) {
		res[key] = value
	})
	return res
}
