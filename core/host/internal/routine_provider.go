package internal

import (
	"github.com/mogud/snow/core/host"
	"github.com/mogud/snow/core/injection"
	"reflect"
	"sync/atomic"
)

var _ injection.IRoutineProvider = (*RoutineProvider)(nil)

type RoutineProvider struct {
	descriptors injection.IRoutineCollection
	root        *RoutineProvider

	scope *RoutineScope
}

func NewProvider(descriptors injection.IRoutineCollection, root *RoutineProvider) *RoutineProvider {
	provider := &RoutineProvider{
		descriptors: descriptors,
		root:        root,
	}
	provider.scope = NewRoutineScope(provider)
	return provider
}

func (ss *RoutineProvider) GetRoutine(ty reflect.Type) any {
	return ss.GetKeyedRoutine(injection.DefaultKey, ty)
}

func (ss *RoutineProvider) GetKeyedRoutine(key any, ty reflect.Type) any {
	descriptor := ss.descriptors.GetKeyedDescriptor(key, ty)
	if descriptor == nil {
		return nil
	}

	if descriptor.Lifetime == injection.Transient {
		return descriptor.Factory(ss.scope)
	}

	var scope *RoutineScope
	if descriptor.Lifetime == injection.Singleton {
		scope = ss.getRootProvider().scope
	} else {
		scope = ss.scope
	}

	for {
		instance := scope.GetKeyedScopedRoutine(key, ty)
		if instance != nil {
			return instance
		}

		if !atomic.CompareAndSwapInt32(&descriptor.InitLock, 0, 1) {
			continue
		}

		instance = scope.GetKeyedScopedRoutine(key, ty)
		if instance != nil {
			atomic.StoreInt32(&descriptor.InitLock, 0)
			return instance
		}

		instance = descriptor.Factory(scope)

		host.Inject(scope, instance)

		scope.SetKeyedScopedRoutine(key, ty, instance)
		atomic.StoreInt32(&descriptor.InitLock, 0)
		return instance
	}
}

func (ss *RoutineProvider) CreateScope() injection.IRoutineScope {
	provider := NewProvider(ss.descriptors, ss.getRootProvider())
	return provider.scope
}

func (ss *RoutineProvider) GetRootScope() injection.IRoutineScope {
	return ss.getRootProvider().scope
}

func (ss *RoutineProvider) getRootProvider() *RoutineProvider {
	if ss.root != nil {
		return ss.root
	}

	return ss
}
