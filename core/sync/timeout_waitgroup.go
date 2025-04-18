package sync

import (
	"github.com/mogud/snow/core/meta"
	"sync/atomic"
	"time"
)

type TimeoutWaitGroup struct {
	noCopy meta.NoCopy

	c       chan struct{}
	counter atomic.Int32
}

func NewTimeoutWaitGroup() *TimeoutWaitGroup {
	return &TimeoutWaitGroup{
		c: make(chan struct{}),
	}
}

func (ss *TimeoutWaitGroup) Done() {
	var v int32
	for {
		v = ss.counter.Load()
		switch {
		case v <= 0:
			return
		case v == 1:
			if ss.counter.CompareAndSwap(v, -1) {
				close(ss.c)
				return
			}
		default:
			if ss.counter.CompareAndSwap(v, v-1) {
				return
			}
		}
	}
}

func (ss *TimeoutWaitGroup) Add(n int) bool {
	for {
		v := ss.counter.Load()
		if v < 0 { // finished
			return false
		}

		if ss.counter.CompareAndSwap(v, v+int32(n)) {
			return true
		}
	}
}

func (ss *TimeoutWaitGroup) WaitTimeout(dur time.Duration) bool {
	for {
		v := ss.counter.Load()
		if v != 0 {
			break
		}

		if ss.counter.CompareAndSwap(v, -1) {
			close(ss.c)
			return true
		}
	}

	select {
	case <-ss.c:
		return true
	case <-time.After(dur):
		for {
			v := ss.counter.Load()
			if v < 0 {
				return false
			}

			if ss.counter.CompareAndSwap(v, -1) {
				close(ss.c)
				return false
			}
		}
	}
}

func (ss *TimeoutWaitGroup) Wait() {
	<-ss.c
}
