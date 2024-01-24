package node

import (
	"container/heap"
	"gitee.com/mogud/snow/core/debug"
	"gitee.com/mogud/snow/core/logging/slog"
	"time"
)

type timeWheel struct {
	idx    int64
	timeMs int64
	queue  *timerQueue
}

func (ss *timeWheel) Update(timeMs int64) {
	ss.timeMs = timeMs
	q := ss.queue
	for q.Len() > 0 && (*q)[0].timeMs <= ss.timeMs {
		it := heap.Pop(q).(*timerItem)
		if !it.start || it.fun == nil {
			continue
		}

		func() {
			defer func() {
				if err := recover(); err != nil {
					buf := debug.StackInfo()
					slog.Errorf("timewheel update error: %s\n%s", err, buf)
				}
			}()
			it.fun()
		}()

		if it.start {
			if it.intervalMs > 0 {
				it.timeMs = ss.timeMs + it.intervalMs
				heap.Push(ss.queue, it)
			} else {
				it.start = false
			}
		}
	}
}

func (ss *timeWheel) CreateTimer() ITimer {
	return &timer{
		tw: ss,
		ti: &timerItem{
			idx: ss.genIdx(),
		},
	}
}

func (ss *timeWheel) Tick(interval, delay time.Duration, f func()) ITimer {
	t := ss.CreateTimer()
	t.Set(interval, delay, f)
	t.Start()
	return t
}

func (ss *timeWheel) After(delay time.Duration, f func()) ITimer {
	t := ss.CreateTimer()
	t.Set(0, delay, f)
	t.Start()
	return t
}

func (ss *timeWheel) genIdx() int64 {
	ss.idx++
	return ss.idx
}
