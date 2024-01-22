package internal

import (
	"context"
	"snow/host"
)

var _ host.IHostApplication = (*HostApplication)(nil)

type HostApplication struct {
	ctx               context.Context
	cancel            func()
	startedListeners  []func()
	stoppedListeners  []func()
	stoppingListeners []func()
}

func (ss *HostApplication) OnStarted(listener func()) {
	ss.startedListeners = append(ss.startedListeners, listener)
}

func (ss *HostApplication) OnStopped(listener func()) {
	ss.stoppedListeners = append(ss.stoppedListeners, listener)
}

func (ss *HostApplication) OnStopping(listener func()) {
	ss.stoppingListeners = append(ss.stoppingListeners, listener)
}

func NewHostApplication() *HostApplication {
	app := &HostApplication{}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	return app
}

func (ss *HostApplication) StopApplication() {
	for _, listener := range ss.stoppingListeners {
		listener()
	}

	ss.cancel()
}
