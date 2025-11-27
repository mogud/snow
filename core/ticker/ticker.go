package ticker

import (
	"context"
	"github.com/mogud/snow/core/task"
	"math"
	"runtime"
	"server/lib/uid"
	"sync"
	"sync/atomic"
	"time"
)

type PoolItem interface {
	Paused() bool
	Closed() bool
}

type Pool struct {
	name         string
	ctx          context.Context
	closeWait    *sync.WaitGroup
	itemChan     chan PoolItem
	tickDuration time.Duration
}

func NewPool(name string, ctx context.Context, wg *sync.WaitGroup, itemChanSize int, tickDuration time.Duration) *Pool {
	return &Pool{
		name:         name,
		ctx:          ctx,
		closeWait:    wg,
		itemChan:     make(chan PoolItem, itemChanSize),
		tickDuration: tickDuration,
	}
}

func (ss *Pool) Start(tickCallback func(item PoolItem), stopCallback func(item PoolItem)) {
	ss.closeWait.Add(1)
	defer ss.closeWait.Done()

	workerSize := runtime.NumCPU()
	type tickWorker struct {
		id    uid.Uid
		count atomic.Int32
		ch    chan PoolItem
	}
	workerMap := make([]*tickWorker, workerSize)
	for i := 0; i < workerSize; i++ {
		workerMap[i] = &tickWorker{
			id: uid.Gen(),
			ch: make(chan PoolItem, 100),
		}
	}

	ss.closeWait.Add(1)
	task.Execute(func() {
		defer ss.closeWait.Done()

		for {
			select {
			case item := <-ss.itemChan:
				var lowestLoadWorker *tickWorker
				var minLoadCount int32 = math.MaxInt32
				for _, worker := range workerMap {
					workerLoadCount := worker.count.Load()
					if workerLoadCount < minLoadCount {
						minLoadCount = workerLoadCount
						lowestLoadWorker = worker
					}
				}

				lowestLoadWorker.ch <- item
			case <-ss.ctx.Done():
				return
			}
		}
	})

	for _, worker := range workerMap {
		ss.closeWait.Add(1)
		task.Execute(func() {
			defer ss.closeWait.Done()

			t := time.NewTicker(ss.tickDuration)
			itemMap := make(map[PoolItem]bool)
			for {
				select {
				case <-t.C:
					var itemToClose []PoolItem
					for item := range itemMap {
						if item.Closed() {
							itemToClose = append(itemToClose, item)
							if stopCallback != nil {
								stopCallback(item)
							}
						} else if !item.Paused() {
							tickCallback(item)
						}
					}

					if len(itemToClose) > 0 {
						for _, item := range itemToClose {
							delete(itemMap, item)
						}
						worker.count.Store(int32(len(itemMap)))
					}
				case item := <-worker.ch:
					if _, ok := itemMap[item]; !ok {
						itemMap[item] = true
						worker.count.Store(int32(len(itemMap)))
					}
				case <-ss.ctx.Done():
					return
				}
			}
		})
	}
}

func (ss *Pool) Add(item PoolItem) {
	ss.itemChan <- item
}
