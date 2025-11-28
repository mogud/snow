package ignore_input

import (
	"bufio"
	"context"
	"github.com/mogud/snow/core/host"
	"github.com/mogud/snow/core/sync"
	"os"
	"sync/atomic"
	"time"
)

var _ host.IHostedRoutine = (*IgnoreInput)(nil)

type IgnoreInput struct {
	closed atomic.Bool
}

func (ss *IgnoreInput) Start(ctx context.Context, wg *sync.TimeoutWaitGroup) {
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for !ss.closed.Load() {
			if !scanner.Scan() {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (ss *IgnoreInput) Stop(ctx context.Context, wg *sync.TimeoutWaitGroup) {
	ss.closed.Store(true)
}
