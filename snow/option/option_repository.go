package option

import (
	"reflect"
	"unsafe"
)

type Repository struct {
	// key : type : option path
	binding map[string]map[reflect.Type]string
	// key : type : option value
	values map[string]map[reflect.Type]any
}

func NewOptionRepository() *Repository {
	return &Repository{
		binding: make(map[string]map[reflect.Type]string),
		values:  make(map[string]map[reflect.Type]any),
	}
}

func (ss *Repository) BindByPath(key string, ty reflect.Type, path string) {
	kv, ok := ss.binding[key]
	if !ok {
		kv = make(map[reflect.Type]string)
		ss.binding[key] = kv
	}
	kv[ty] = path
}

func (ss *Repository) BindByValue(key string, ty reflect.Type, value any) {
	kv, ok := ss.values[key]
	if !ok {
		kv = make(map[reflect.Type]any)
		ss.values[key] = kv
	}
	kv[ty] = value
}

func (ss *Repository) GetOption(ty reflect.Type) any {
	factoryTy := ty.Elem().Field(0).Type
	instanceValue := reflect.New(ty.Elem())

	field := reflect.NewAt(factoryTy, unsafe.Pointer(instanceValue.Elem().Field(0).UnsafeAddr())).Elem()
	field.Set(reflect.ValueOf(func(key string) any {
		optTy := ty.Elem().Field(1).Type

		var optValue reflect.Value
		var ok bool
		var vkv map[reflect.Type]any
		if vkv, ok = ss.values[key]; ok {
			var v any
			if v, ok = vkv[optTy]; ok {
				optValue = reflect.ValueOf(v)
			}
		}

		if !ok {
			optValue = reflect.New(optTy.Elem())
			var pkv map[reflect.Type]string
			if pkv, ok = ss.binding[key]; ok {
				var p string
				if p, ok = pkv[optTy]; ok {
					_ = GetByKey(p, optValue.Interface())
				}
			}
		}

		return optValue.Interface()
	}))
	return instanceValue.Interface()
}

func BindOptionPath[T any](repo *Repository, path string) {
	BindKeyedOptionPath[T](repo, "", path)
}

func BindKeyedOptionPath[T any](repo *Repository, key string, path string) {
	repo.BindByPath(key, reflect.TypeOf((*T)(nil)).Elem(), path)
}

func BindOptionValue[T any](repo *Repository, value T) {
	BindKeyedOptionValue[T](repo, "", value)
}

func BindKeyedOptionValue[T any](repo *Repository, key string, value T) {
	repo.BindByValue(key, reflect.TypeOf((*T)(nil)).Elem(), value)
}
