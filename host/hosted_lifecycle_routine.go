package host

import (
	"context"
	"snow/core/syncext"
)

type IHostedLifecycleRoutine interface {
	IHostedRoutine

	BeforeStart(ctx context.Context, wg *syncext.TimeoutWaitGroup)
	AfterStart(ctx context.Context, wg *syncext.TimeoutWaitGroup)
	BeforeStop(ctx context.Context, wg *syncext.TimeoutWaitGroup)
	AfterStop(ctx context.Context, wg *syncext.TimeoutWaitGroup)
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
