package node

import (
	"github.com/mogud/snow/core/task"
	"sync/atomic"
)

type NodeAddrUpdater struct {
	naddr   int64
	updatef func(chan<- *NodeAddr)
	running int32
	sigChan chan bool
}

func NewNodeAddrUpdater(naddr NodeAddr, updateFunc func(chan<- *NodeAddr)) *NodeAddrUpdater {
	return &NodeAddrUpdater{
		naddr:   int64(naddr),
		updatef: updateFunc,
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

func (ss *NodeAddrUpdater) GetNodeAddr() NodeAddr {
	return NodeAddr(atomic.LoadInt64(&ss.naddr))
}

func (ss *NodeAddrUpdater) getSigChan() chan<- bool {
	return ss.sigChan
}

func (ss *NodeAddrUpdater) retryUpdateAddr() {
	if !atomic.CompareAndSwapInt32(&ss.running, 0, 1) {
		return
	}

	addrChan := make(chan *NodeAddr, 1)
	ss.updatef(addrChan)

	task.Execute(func() {
		newaddr := <-addrChan
		if newaddr != nil {
			atomic.StoreInt64(&ss.naddr, int64(*newaddr))
		}
		atomic.StoreInt32(&ss.running, 0)
	})
}
