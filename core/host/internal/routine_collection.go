package internal

import (
	"reflect"
	"snow/core/injection"
	"sync"
)

var _ injection.IRoutineCollection = (*RoutineCollection)(nil)

type RoutineCollection struct {
	lock sync.Mutex

	descriptors map[any]map[reflect.Type]*injection.RoutineDescriptor
}

func NewRoutineCollection() *RoutineCollection {
	return &RoutineCollection{
		descriptors: make(map[any]map[reflect.Type]*injection.RoutineDescriptor),
	}
}

func (ss *RoutineCollection) AddDescriptor(descriptor *injection.RoutineDescriptor) {
	if descriptor.Key == nil {
		descriptor.Key = injection.DefaultKey
	}

	ss.lock.Lock()
	defer ss.lock.Unlock()

	descriptors, ok := ss.descriptors[descriptor.Key]
	if !ok {
		descriptors = make(map[reflect.Type]*injection.RoutineDescriptor)
		ss.descriptors[descriptor.Key] = descriptors
	}

	descriptors[descriptor.TyKey] = descriptor
}

func (ss *RoutineCollection) GetDescriptors() []*injection.RoutineDescriptor {
	var descriptors []*injection.RoutineDescriptor

	ss.lock.Lock()
	defer ss.lock.Unlock()

	for _, descriptorsRaw := range ss.descriptors {
		for _, descriptor := range descriptorsRaw {
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors
}

func (ss *RoutineCollection) GetDescriptor(ty reflect.Type) *injection.RoutineDescriptor {
	return ss.GetKeyedDescriptor(injection.DefaultKey, ty)
}

func (ss *RoutineCollection) GetKeyedDescriptor(key any, ty reflect.Type) *injection.RoutineDescriptor {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	descriptors, ok := ss.descriptors[key]
	if !ok {
		return nil
	}
	return descriptors[ty]
}
