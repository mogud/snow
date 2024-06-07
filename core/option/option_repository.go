package option

import (
	"github.com/mogud/snow/core/configuration"
	"reflect"
	"unsafe"
)

type repositoryBinding struct {
	path  string
	value any
}

type Repository struct {
	// key: type: binding
	mapping map[string]map[reflect.Type]*repositoryBinding
	config  configuration.IConfiguration
}

func NewOptionRepository(config configuration.IConfiguration) *Repository {
	return &Repository{
		mapping: make(map[string]map[reflect.Type]*repositoryBinding),
		config:  config,
	}
}

func (ss *Repository) BindByPath(key string, ty reflect.Type, path string) {
	kv, ok := ss.mapping[key]
	if !ok {
		kv = make(map[reflect.Type]*repositoryBinding)
		ss.mapping[key] = kv
	}
	kv[ty] = &repositoryBinding{path: path}
}

func (ss *Repository) BindByValue(key string, ty reflect.Type, value any) {
	kv, ok := ss.mapping[key]
	if !ok {
		kv = make(map[reflect.Type]*repositoryBinding)
		ss.mapping[key] = kv
	}
	kv[ty] = &repositoryBinding{value: value}
}

func (ss *Repository) GetOptionWrapper(ty reflect.Type) any {
	wrapperTy := ty.Elem()

	factoryTy := wrapperTy.Field(0).Type
	registerCallBackTy := wrapperTy.Field(1).Type
	optTy := wrapperTy.Field(2).Type
	instanceValue := reflect.New(wrapperTy)

	factoryField := reflect.NewAt(factoryTy, unsafe.Pointer(instanceValue.Elem().Field(0).UnsafeAddr())).Elem()
	factoryField.Set(reflect.ValueOf(func(key string) any {
		var optValue reflect.Value

		var ok bool
		var tkv map[reflect.Type]*repositoryBinding
		var binding *repositoryBinding
		tkv, ok = ss.mapping[key]
		if !ok {
			optValue = reflect.New(optTy.Elem())
		} else if binding, ok = tkv[optTy]; !ok {
			optValue = reflect.New(optTy.Elem())
		} else if binding.value != nil {
			optValue = reflect.ValueOf(binding.value)
		} else {
			optValue = reflect.ValueOf(configuration.GetByType(optTy, ss.config.GetSection(binding.path), ""))
		}

		return optValue.Interface()
	}))

	registerCallBackField := reflect.NewAt(registerCallBackTy, unsafe.Pointer(instanceValue.Elem().Field(1).UnsafeAddr())).Elem()
	registerCallBackField.Set(reflect.ValueOf(func(key string, cb func()) {
		if tkv, ok := ss.mapping[key]; !ok {
			return
		} else if binding, ok := tkv[optTy]; !ok {
			return
		} else if binding.value != nil {
			return
		} else if len(binding.path) == 0 {
			return
		} else {
			ss.config.GetSection(binding.path).GetReloadNotifier().RegisterNotifyCallback(cb)
		}
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
