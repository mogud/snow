package host

import (
	"context"
	"snow/core/syncext"
)

type IHostedRoutine interface {
	Start(ctx context.Context, wg *syncext.TimeoutWaitGroup)
	Stop(ctx context.Context, wg *syncext.TimeoutWaitGroup)
}

type IHostedRoutineContainer interface {
	AddHostedRoutine(factory func() IHostedRoutine)
	BuildHostedRoutines()
	GetHostedRoutines() []IHostedRoutine
}

func AddHostedRoutine[U IHostedRoutine](builder IBuilder) {
	provider := builder.GetRoutineProvider()
	container := GetRoutine[IHostedRoutineContainer](provider)

	container.AddHostedRoutine(func() IHostedRoutine {
		s := NewStruct[U]()
		Inject(provider.GetRootScope(), s)
		return s
	})
}
