package internal

import (
	"context"
	"fmt"
	"gitee.com/mogud/snow/host"
	"gitee.com/mogud/snow/injection"
	"gitee.com/mogud/snow/logging"
	"gitee.com/mogud/snow/option"
	"gitee.com/mogud/snow/sync"
	"time"
	"unsafe"
)

var _ host.IHost = (*Host)(nil)

type HostOption struct {
	StartWaitTimeoutSeconds int
	StopWaitTimeoutSeconds  int
}

type Host struct {
	option                          *HostOption
	logger                          logging.ILogger
	provider                        injection.IRoutineProvider
	app                             *HostApplication
	hostedRoutineContainer          host.IHostedRoutineContainer
	hostedRoutines                  []host.IHostedRoutine
	hostedLifecycleRoutineContainer host.IHostedLifecycleRoutineContainer
	hostedLifecycleRoutines         []host.IHostedLifecycleRoutine
}

func NewHost(provider injection.IRoutineProvider) *Host {
	return &Host{provider: provider}
}

func (ss *Host) Construct(option *option.Option[*HostOption], logger *logging.Logger[Host]) {
	ss.option = option.Get()
	if ss.option.StartWaitTimeoutSeconds == 0 {
		ss.option.StartWaitTimeoutSeconds = 5
	}
	if ss.option.StopWaitTimeoutSeconds == 0 {
		ss.option.StopWaitTimeoutSeconds = 8
	}

	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = "Host"
		data.ID = fmt.Sprintf("%X", unsafe.Pointer(ss))
	})
}

func (ss *Host) Start(ctx context.Context, wg *sync.TimeoutWaitGroup) {
	wg.Add(1)
	defer wg.Done()

	if ss.app == nil {
		ss.app = injection.GetRoutine[host.IHostApplication](ss.provider).(*HostApplication)
	}
	if ss.hostedRoutineContainer == nil {
		ss.hostedRoutineContainer = injection.GetRoutine[host.IHostedRoutineContainer](ss.provider)
		ss.hostedRoutineContainer.BuildHostedRoutines()
		ss.hostedRoutines = ss.hostedRoutineContainer.GetHostedRoutines()
	}
	if ss.hostedLifecycleRoutineContainer == nil {
		ss.hostedLifecycleRoutineContainer = injection.GetRoutine[host.IHostedLifecycleRoutineContainer](ss.provider)
		ss.hostedLifecycleRoutineContainer.BuildHostedLifecycleRoutines()
		ss.hostedLifecycleRoutines = ss.hostedLifecycleRoutineContainer.GetHostedLifecycleRoutines()
	}

	combinedCtx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
		case <-ss.app.ctx.Done():
		}
		cancel()
	}()

	if len(ss.hostedLifecycleRoutines) > 0 {
		routineWg := sync.NewTimeoutWaitGroup()
		routineWg.Add(len(ss.hostedLifecycleRoutines))
		for _, routine := range ss.hostedLifecycleRoutines {
			routine := routine
			go func() {
				routine.BeforeStart(combinedCtx, routineWg)
				routineWg.Done()
			}()
		}
		if !routineWg.WaitTimeout(time.Duration(ss.option.StartWaitTimeoutSeconds) * time.Second) {
			ss.logger.Warnf("'BeforeStart' wait timeout in hosted lifecycle routines")
		}
	}

	if len(ss.hostedLifecycleRoutines) > 0 || len(ss.hostedRoutines) > 0 {
		routineWg := sync.NewTimeoutWaitGroup()
		if len(ss.hostedLifecycleRoutines) > 0 {
			routineWg.Add(len(ss.hostedLifecycleRoutines))
			for _, routine := range ss.hostedLifecycleRoutines {
				routine := routine
				go func() {
					routine.Start(combinedCtx, routineWg)
					routineWg.Done()
				}()
			}
		}
		if len(ss.hostedRoutines) > 0 {
			routineWg.Add(len(ss.hostedRoutines))
			for _, routine := range ss.hostedRoutines {
				routine := routine
				go func() {
					routine.Start(combinedCtx, routineWg)
					routineWg.Done()
				}()
			}
		}
		if !routineWg.WaitTimeout(time.Duration(ss.option.StartWaitTimeoutSeconds) * time.Second) {
			ss.logger.Warnf("'Start' wait timeout in hosted routines")
		}
	}

	if len(ss.hostedLifecycleRoutines) > 0 {
		routineWg := sync.NewTimeoutWaitGroup()
		routineWg.Add(len(ss.hostedLifecycleRoutines))
		for _, routine := range ss.hostedLifecycleRoutines {
			routine := routine
			go func() {
				routine.AfterStart(combinedCtx, routineWg)
				routineWg.Done()
			}()
		}

		if !routineWg.WaitTimeout(time.Duration(ss.option.StartWaitTimeoutSeconds) * time.Second) {
			ss.logger.Warnf("'AfterStart' wait timeout in hosted lifecycle routines")
		}
	}

	for _, listener := range ss.app.startedListeners {
		listener()
	}
}

func (ss *Host) Stop(ctx context.Context, wg *sync.TimeoutWaitGroup) {
	wg.Add(1)
	defer wg.Done()

	if len(ss.hostedLifecycleRoutines) > 0 {
		routineWg := sync.NewTimeoutWaitGroup()
		routineWg.Add(len(ss.hostedLifecycleRoutines))
		for _, routine := range ss.hostedLifecycleRoutines {
			routine := routine
			go func() {
				routine.BeforeStop(ctx, routineWg)
				routineWg.Done()
			}()
		}
		if !routineWg.WaitTimeout(time.Duration(ss.option.StopWaitTimeoutSeconds) * time.Second) {
			ss.logger.Warnf("'BeforeStop' wait timeout in hosted lifecycle routines")
		}
	}

	if len(ss.hostedLifecycleRoutines) > 0 || len(ss.hostedRoutines) > 0 {
		routineWg := sync.NewTimeoutWaitGroup()
		if len(ss.hostedLifecycleRoutines) > 0 {
			routineWg.Add(len(ss.hostedLifecycleRoutines))
			for _, routine := range ss.hostedLifecycleRoutines {
				routine := routine
				go func() {
					routine.Stop(ctx, routineWg)
					routineWg.Done()
				}()
			}
		}
		if len(ss.hostedRoutines) > 0 {
			routineWg.Add(len(ss.hostedRoutines))
			for _, routine := range ss.hostedRoutines {
				routine := routine
				go func() {
					routine.Stop(ctx, routineWg)
					routineWg.Done()
				}()
			}
		}
		if !routineWg.WaitTimeout(time.Duration(ss.option.StopWaitTimeoutSeconds) * time.Second) {
			ss.logger.Warnf("'Stop' wait timeout in hosted routines")
		}
	}

	if len(ss.hostedLifecycleRoutines) > 0 {
		routineWg := sync.NewTimeoutWaitGroup()
		routineWg.Add(len(ss.hostedLifecycleRoutines))
		for _, routine := range ss.hostedLifecycleRoutines {
			routine := routine
			go func() {
				routine.AfterStop(ctx, routineWg)
				routineWg.Done()
			}()
		}
		if !routineWg.WaitTimeout(time.Duration(ss.option.StopWaitTimeoutSeconds) * time.Second) {
			ss.logger.Warnf("'AfterStop' wait timeout in hosted lifecycle routines")
		}
	}

	if ss.app != nil {
		for _, listener := range ss.app.stoppedListeners {
			listener()
		}
	}
}

func (ss *Host) GetRoutineProvider() injection.IRoutineProvider {
	return ss.provider
}
