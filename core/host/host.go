package host

import (
	"reflect"
	"snow/core/injection"
)

type IHost interface {
	IHostedRoutine

	GetRoutineProvider() injection.IRoutineProvider
}

func GetRoutine[T any](provider injection.IRoutineProvider) T {
	ty := reflect.TypeOf((*T)(nil)).Elem()
	return provider.GetRoutine(ty).(T)
}
