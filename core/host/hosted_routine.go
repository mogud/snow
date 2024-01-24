package host

import (
	"context"
	"gitee.com/mogud/snow/core/injection"
	"gitee.com/mogud/snow/core/sync"
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

	AddSingleton[U](builder)
	container.AddHostedRoutine(func() IHostedRoutine {
		return injection.GetRoutine[U](provider)
	})
}
