package host

import (
	"context"
	"github.com/mogud/snow/core/sync"
)

type IHostApplication interface {
	OnStarted(listener func())
	OnStopped(listener func())
	OnStopping(listener func())

	StopApplication()
}

func Run(h IHost) {
	app := GetRoutine[IHostApplication](h.GetRoutineProvider())
	ctx, cancel := context.WithCancel(context.Background())

	started := false
	app.OnStopping(func() {
		cancel()
	})
	app.OnStarted(func() {
		started = true
	})

	wg := sync.NewTimeoutWaitGroup()
	h.Start(ctx, wg)
	wg.Wait()

	<-ctx.Done()

	if started {
		wg = sync.NewTimeoutWaitGroup()
		h.Stop(context.Background(), wg)
		wg.Wait()
	}
}
