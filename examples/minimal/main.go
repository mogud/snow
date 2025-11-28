package main

import (
	"context"
	"time"

	"github.com/mogud/snow/core/host"
	"github.com/mogud/snow/core/host/builder"
	"github.com/mogud/snow/core/logging/slog"
	"github.com/mogud/snow/core/sync"
	"github.com/mogud/snow/routines/ignore_input"
)

var _ host.IHostedRoutine = (*clock)(nil)

type clock struct {
	closeChan chan struct{}
}

func (ss *clock) Start(_ context.Context, wg *sync.TimeoutWaitGroup) {
	ss.closeChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Second)
	loop:
		for {
			select {
			case <-ticker.C:
				h, m, s := time.Now().Clock()
				slog.Infof("Now => %02v:%02v:%02v", h, m, s)
			case <-ss.closeChan:
				break loop
			}
		}
	}()
}

func (ss *clock) Stop(_ context.Context, wg *sync.TimeoutWaitGroup) {
	close(ss.closeChan)
}

func main() {
	b := builder.NewDefaultBuilder()
	host.AddHostedRoutine[*ignore_input.IgnoreInput](b)
	host.AddHostedRoutine[*clock](b)
	host.Run(b.Build())
}
