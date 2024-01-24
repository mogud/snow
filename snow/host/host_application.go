package host

import (
	"context"
	"gitee.com/mogud/snow/sync"
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

	wg := sync.NewTimeoutWaitGroup()
	h.Start(context.Background(), wg)
	wg.Wait()

	<-ctx.Done()

	wg = sync.NewTimeoutWaitGroup()
	h.Stop(context.Background(), wg)
	wg.Wait()
}
