package injection

import (
	"reflect"
)

var DefaultKey = struct{}{}

type IRoutineCollection interface {
	AddDescriptor(descriptor *RoutineDescriptor)
	GetDescriptors() []*RoutineDescriptor
	GetDescriptor(ty reflect.Type) *RoutineDescriptor
	GetKeyedDescriptor(key any, ty reflect.Type) *RoutineDescriptor
}

type IRoutineScope interface {
	GetScopedRoutine(ty reflect.Type) any
	GetKeyedScopedRoutine(key any, ty reflect.Type) any
	SetScopedRoutine(ty reflect.Type, value any)
	SetKeyedScopedRoutine(key any, ty reflect.Type, value any)
	GetRoot() IRoutineScope
	GetProvider() IRoutineProvider
}

type IRoutineProvider interface {
	GetRoutine(ty reflect.Type) any
	GetKeyedRoutine(key any, ty reflect.Type) any

	CreateScope() IRoutineScope
	GetRootScope() IRoutineScope
}

func GetRoutine[T any](provider IRoutineProvider) T {
	ty := reflect.TypeOf((*T)(nil)).Elem()
	return provider.GetRoutine(ty).(T)
}

func GetKeyedRoutine[T any](provider IRoutineProvider, key any) T {
	ty := reflect.TypeOf((*T)(nil)).Elem()
	return provider.GetKeyedRoutine(key, ty).(T)
}

type RoutineDescriptor struct {
	Lifetime RoutineLifetime               // Routine 生命期
	InitLock int32                         // 初始化锁
	Key      any                           // 按 Key 注册
	TyKey    reflect.Type                  // 注册的接口类型 Key
	TyImpl   reflect.Type                  // 注册的实现类型
	Factory  func(scope IRoutineScope) any // 工厂方法，用于在 scope 中创建实例，方法必须返回新实例
}

type RoutineLifetime uint8

const (
	Singleton RoutineLifetime = iota
	Scoped
	Transient
)
