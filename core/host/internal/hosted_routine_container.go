package internal

import (
	"github.com/mogud/snow/core/host"
)

var _ host.IHostedRoutineContainer = (*HostedRoutineContainer)(nil)

type HostedRoutineContainer struct {
	routines []host.IHostedRoutine
	factory  []func() host.IHostedRoutine
}

func (ss *HostedRoutineContainer) AddHostedRoutine(factory func() host.IHostedRoutine) {
	ss.factory = append(ss.factory, factory)
}

func (ss *HostedRoutineContainer) BuildHostedRoutines() {
	for _, f := range ss.factory {
		ss.routines = append(ss.routines, f())
	}
}

func (ss *HostedRoutineContainer) GetHostedRoutines() []host.IHostedRoutine {
	return ss.routines
}
