package ticker

import (
	"github.com/mogud/snow/core/task"
	"sync"
	"time"
)

var TickInterval int64 = 33_333_333

var (
	lock sync.Mutex
	chs  map[int64]chan<- int64
)

func init() {
	chs = make(map[int64]chan<- int64, 200)
	task.Execute(func() {
		prevUnixNano := time.Now().UnixNano()
		for {
			lock.Lock()
			for _, ch := range chs {
				select {
				case ch <- prevUnixNano:
				default:
				}
			}
			lock.Unlock()

			nowUnixNano := time.Now().UnixNano()
			nextUnixNano := prevUnixNano + TickInterval
			delta := nextUnixNano - nowUnixNano
			if delta > 0 {
				prevUnixNano = nextUnixNano
				time.Sleep(time.Duration(delta))
			} else {
				prevUnixNano = nowUnixNano
			}
		}
	})
}

func Subscribe(id int64, ch chan<- int64) {
	lock.Lock()
	defer lock.Unlock()
	chs[id] = ch
}

func Unsubscribe(id int64) {
	lock.Lock()
	defer lock.Unlock()
	delete(chs, id)
}
