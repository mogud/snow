package internal

import (
	"github.com/mogud/snow/core/host"
)

var _ host.IHostedLifecycleRoutineContainer = (*HostedLifecycleRoutineContainer)(nil)

type HostedLifecycleRoutineContainer struct {
	routines []host.IHostedLifecycleRoutine
	factory  []func() host.IHostedLifecycleRoutine
}

func (ss *HostedLifecycleRoutineContainer) AddHostedLifecycleRoutine(factory func() host.IHostedLifecycleRoutine) {
	ss.factory = append(ss.factory, factory)
}

func (ss *HostedLifecycleRoutineContainer) BuildHostedLifecycleRoutines() {
	for _, f := range ss.factory {
		ss.routines = append(ss.routines, f())
	}
}

func (ss *HostedLifecycleRoutineContainer) GetHostedLifecycleRoutines() []host.IHostedLifecycleRoutine {
	return ss.routines
}
