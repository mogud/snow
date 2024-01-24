package node

import (
	"container/heap"
	"time"
)

var _ = ITimer((*timer)(nil))

type timer struct {
	tw *timeWheel
	ti *timerItem
}

func (ss *timer) Start() {
	if !ss.ti.start {
		ss.ti.start = true
		heap.Push(ss.tw.queue, ss.ti)
	}
}

func (ss *timer) Stop() {
	if ss.ti.start {
		ss.ti.start = false
		old := ss.ti
		ss.ti = &timerItem{
			idx:        ss.tw.genIdx(),
			timeMs:     old.timeMs,
			intervalMs: old.intervalMs,
			fun:        old.fun,
		}
	}
}

func (ss *timer) SetInterval(interval time.Duration) {
	ss.ti.intervalMs = int64(interval / time.Millisecond)
}

func (ss *timer) SetDelay(delay time.Duration) {
	ss.ti.timeMs = ss.tw.timeMs + int64(delay/time.Millisecond)
}

func (ss *timer) SetFunc(f func()) {
	ss.ti.fun = f
}

func (ss *timer) Set(interval, delay time.Duration, f func()) {
	ss.SetInterval(interval)
	ss.SetDelay(delay)
	ss.SetFunc(f)
}
