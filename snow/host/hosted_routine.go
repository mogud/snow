package host

import (
	"context"
	"gitee.com/mogud/snow/injection"
	"gitee.com/mogud/snow/sync"
)

type IHostedRoutine interface {
	Start(ctx context.Context, wg *sync.TimeoutWaitGroup)
	Stop(ctx context.Context, wg *sync.TimeoutWaitGroup)
}

type IHostedRoutineContainer interface {
	AddHostedRoutine(factory func() IHostedRoutine)
	BuildHostedRoutines()
	GetHostedRoutines() []IHostedRoutine
}

func AddHostedRoutine[U IHostedRoutine](builder IBuilder) {
	provider := builder.GetRoutineProvider()
	container := GetRoutine[IHostedRoutineContainer](provider)
	s := NewStruct[U]()
	AddSingletonFactory[U](builder, func(scope injection.IRoutineScope) U {
		return s
	})

	container.AddHostedRoutine(func() IHostedRoutine {
		Inject(provider.GetRootScope(), s)
		return s
	})
}
