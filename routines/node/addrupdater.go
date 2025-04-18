package node

import (
	"snow/core/task"
	"sync/atomic"
)

type NodeAddrUpdater struct {
	nAddr   int64
	updateF func(chan<- *Addr)
	running int32
	sigChan chan bool
}

func NewNodeAddrUpdater(nAddr Addr, updateFunc func(chan<- *Addr)) *NodeAddrUpdater {
	return &NodeAddrUpdater{
		nAddr:   int64(nAddr),
		updateF: updateFunc,
		sigChan: make(chan bool, 1024),
	}
}

func (ss *NodeAddrUpdater) Start() {
	task.Execute(func() {
		for {
			select {
			case <-ss.sigChan:
				ss.retryUpdateAddr()
			}
		}
	})
}

func (ss *NodeAddrUpdater) GetNodeAddr() Addr {
	return Addr(atomic.LoadInt64(&ss.nAddr))
}

func (ss *NodeAddrUpdater) getSigChan() chan<- bool {
	return ss.sigChan
}

func (ss *NodeAddrUpdater) retryUpdateAddr() {
	if !atomic.CompareAndSwapInt32(&ss.running, 0, 1) {
		return
	}

	addrChan := make(chan *Addr, 1)
	ss.updateF(addrChan)

	task.Execute(func() {
		newAddr := <-addrChan
		if newAddr != nil {
			atomic.StoreInt64(&ss.nAddr, int64(*newAddr))
		}
		atomic.StoreInt32(&ss.running, 0)
	})
}
