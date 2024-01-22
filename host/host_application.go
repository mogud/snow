package host

import (
	"context"
	"gitee.com/mogud/snow/core/syncext"
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
	app.OnStopping(func() {
		cancel()
	})

	wg := syncext.NewTimeoutWaitGroup()
	h.Start(context.Background(), wg)
	wg.Wait()

	<-ctx.Done()

	wg = syncext.NewTimeoutWaitGroup()
	h.Stop(context.Background(), wg)
	wg.Wait()
}
