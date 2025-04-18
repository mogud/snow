package host

import (
	"context"
	"snow/core/sync"
)

type IHostedLifecycleRoutine interface {
	IHostedRoutine

	BeforeStart(ctx context.Context, wg *sync.TimeoutWaitGroup)
	AfterStart(ctx context.Context, wg *sync.TimeoutWaitGroup)
	BeforeStop(ctx context.Context, wg *sync.TimeoutWaitGroup)
	AfterStop(ctx context.Context, wg *sync.TimeoutWaitGroup)
}

type IHostedLifecycleRoutineContainer interface {
	AddHostedLifecycleRoutine(factory func() IHostedLifecycleRoutine)
	BuildHostedLifecycleRoutines()
	GetHostedLifecycleRoutines() []IHostedLifecycleRoutine
}

func AddHostedLifecycleRoutine[U IHostedLifecycleRoutine](builder IBuilder) {
	provider := builder.GetRoutineProvider()
	container := GetRoutine[IHostedLifecycleRoutineContainer](provider)

	container.AddHostedLifecycleRoutine(func() IHostedLifecycleRoutine {
		s := NewStruct[U]()
		Inject(provider.GetRootScope(), s)
		return s
	})
}
