package internal

import (
	"gitee.com/mogud/snow/injection"
	"reflect"
	"sync"
)

var _ injection.IRoutineScope = (*RoutineScope)(nil)

type RoutineScope struct {
	lock sync.Mutex

	provider  *RoutineProvider
	instances map[any]map[reflect.Type]any
}

func NewRoutineScope(provider *RoutineProvider) *RoutineScope {
	return &RoutineScope{
		provider:  provider,
		instances: make(map[any]map[reflect.Type]any),
	}
}

func (ss *RoutineScope) GetScopedRoutine(ty reflect.Type) any {
	return ss.GetKeyedScopedRoutine(injection.DefaultKey, ty)
}

func (ss *RoutineScope) GetKeyedScopedRoutine(key any, ty reflect.Type) any {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	instances, ok := ss.instances[key]
	if !ok {
		return nil
	}
	return instances[ty]
}

func (ss *RoutineScope) SetScopedRoutine(ty reflect.Type, value any) {
	ss.SetKeyedScopedRoutine(injection.DefaultKey, ty, value)
}

func (ss *RoutineScope) SetKeyedScopedRoutine(key any, ty reflect.Type, value any) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	instances, ok := ss.instances[key]
	if !ok {
		instances = make(map[reflect.Type]any)
		ss.instances[key] = instances
	}

	instances[ty] = value
}

func (ss *RoutineScope) GetRoot() injection.IRoutineScope {
	return ss.provider.getRootProvider().scope
}

func (ss *RoutineScope) GetProvider() injection.IRoutineProvider {
	return ss.provider
}
