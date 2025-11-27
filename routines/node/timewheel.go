package node

import (
	assert "github.com/arl/assertgo"
	"github.com/mogud/snow/core/debug"
	"github.com/mogud/snow/core/logging/slog"
	"time"
)

const maxTimeWheelDuration = 23*time.Hour + 59*time.Minute + 59*time.Second

var _ = ITimeWheelHandle((*timeWheelItem)(nil))

type timeWheelItem struct {
	interval time.Duration
	nextTime time.Time
	stopped  bool
	callback func()
}

func (ss *timeWheelItem) Stop() {
	ss.stopped = true
}

type timeWheel struct {
	hourWheel   [24][]*timeWheelItem
	minuteWheel [60][]*timeWheelItem
	secondWheel [60][]*timeWheelItem
	curTime     time.Time
	step        time.Duration
}

func newTimeWheel(curTime time.Time, step time.Duration) *timeWheel {
	assert.True(step > 0)

	return &timeWheel{
		curTime: curTime,
		step:    step,
	}
}

func (ss *timeWheel) update(nowTime time.Time) {
	curHour, curMinute, curSecond := ss.curTime.Clock()
	for {
		nextTime := ss.curTime.Add(ss.step)

		if nextTime.After(nowTime) {
			break
		}

		nextHour, nextMinute, nextSecond := nextTime.Clock()

		var pendingItem []*timeWheelItem
		if nextHour != curHour {
			pendingItem = ss.hourWheel[nextHour]
			ss.hourWheel[nextHour] = nil
		}

		if nextMinute != curMinute {
			if pendingItem == nil {
				pendingItem = ss.minuteWheel[nextMinute]
			} else {
				pendingItem = append(pendingItem, ss.minuteWheel[nextMinute]...)
			}
			ss.minuteWheel[nextMinute] = nil
		}

		curSecondList := ss.secondWheel[curSecond]
		ss.secondWheel[curSecond] = nil
		if pendingItem == nil {
			pendingItem = ss.process(nextTime, curSecondList)
		} else {
			pendingItem = append(pendingItem, ss.process(nextTime, curSecondList)...)
		}

		ss.curTime = nextTime
		curHour, curMinute, curSecond = nextHour, nextMinute, nextSecond

		for _, item := range pendingItem {
			ss.insertItem(item)
		}
	}
}

func (ss *timeWheel) process(now time.Time, list []*timeWheelItem) []*timeWheelItem {
	for i := 0; i < len(list); i++ {
		item := list[i]
		if !item.stopped {
			if item.nextTime.After(now) {
				continue
			}

			func() {
				defer func() {
					if err := recover(); err != nil {
						buf := debug.StackInfo()
						slog.Errorf("timewheel update error: %s\n%s", err, buf)
					}
				}()

				item.callback()
			}()

			if item.interval > 0 {
				item.nextTime = item.nextTime.Add(item.interval)
				continue
			}
		}

		list = append(list[:i], list[i+1:]...)
		i--
	}
	return list
}

func (ss *timeWheel) insertItem(item *timeWheelItem) {
	if item.nextTime.Before(ss.curTime) {
		item.nextTime = ss.curTime
	}

	curHour, curMinute, _ := ss.curTime.Clock()
	nextHour, nextMinute, nextSecond := item.nextTime.Clock()
	if nextHour != curHour {
		ss.hourWheel[nextHour] = append(ss.hourWheel[nextHour], item)
	} else if nextMinute != curMinute {
		ss.minuteWheel[nextMinute] = append(ss.minuteWheel[nextMinute], item)
	} else {
		ss.secondWheel[nextSecond] = append(ss.secondWheel[nextSecond], item)
	}
}

func (ss *timeWheel) createTickItem(interval, delay time.Duration, callback func()) ITimeWheelHandle {
	if interval < 0 {
		interval = 0
	} else if interval > maxTimeWheelDuration {
		interval = maxTimeWheelDuration
	} else if interval != 0 && interval < ss.step {
		interval = ss.step
	}

	if delay < 0 {
		delay = 0
	} else if delay > maxTimeWheelDuration {
		delay = maxTimeWheelDuration
	}

	item := &timeWheelItem{
		interval: interval,
		nextTime: ss.curTime.Add(delay),
		stopped:  false,
		callback: callback,
	}
	ss.insertItem(item)
	return item
}

func (ss *timeWheel) createAfterItem(delay time.Duration, callback func()) ITimeWheelHandle {
	if delay < 0 {
		delay = 0
	} else if delay > maxTimeWheelDuration {
		delay = maxTimeWheelDuration
	}

	item := &timeWheelItem{
		interval: 0,
		nextTime: ss.curTime.Add(delay),
		stopped:  false,
		callback: callback,
	}
	ss.insertItem(item)
	return item
}
