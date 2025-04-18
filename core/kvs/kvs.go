package kvs

import (
	"reflect"
	"sync"
)

var kvStore = sync.Map{}

func Set(key string, value any) {
	kvStore.Store(key, value)
}

func GetAny(key string) (any, bool) {
	return kvStore.Load(key)
}

func Get[T any](key string) (T, bool) {
	if v, ok := GetAny(key); ok {
		return v.(T), true
	}

	return reflect.New(reflect.TypeOf((*T)(nil)).Elem()).Elem().Interface().(T), false
}
