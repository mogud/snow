package internal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"snow/core/host"
	"snow/core/logging"
	sync2 "snow/core/sync"
	"sync"
	"syscall"
	"unsafe"
)

var _ host.IHostedRoutine = (*ConsoleLifetimeRoutine)(nil)

type ConsoleLifetimeRoutine struct {
	logger      logging.ILogger
	ctx         context.Context
	cancel      func()
	wg          *sync.WaitGroup
	application host.IHostApplication
}

func (ss *ConsoleLifetimeRoutine) Construct(application host.IHostApplication, logger *logging.Logger[ConsoleLifetimeRoutine]) {
	ss.application = application
	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = "ConsoleLifetime"
		data.ID = fmt.Sprintf("%X", unsafe.Pointer(ss))
	})
}

func (ss *ConsoleLifetimeRoutine) Start(_ context.Context, wg *sync2.TimeoutWaitGroup) {
	ss.ctx, ss.cancel = context.WithCancel(context.Background())
	ss.wg = &sync.WaitGroup{}
	ss.wg.Add(1)
	wg.Add(1)
	go func() {
		wg.Done()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		select {
		case <-sigs:
			ss.logger.Infof("SHUTDOWN APPLICATION BY SIGNAL...")
		case <-ss.ctx.Done():
			ss.logger.Infof("SHUTDOWN APPLICATION")
		}
		ss.wg.Done()

		ss.application.StopApplication()
	}()
}

func (ss *ConsoleLifetimeRoutine) Stop(_ context.Context, wg *sync2.TimeoutWaitGroup) {
	wg.Add(1)
	defer wg.Done()

	ss.cancel()
	ss.wg.Wait()
}
