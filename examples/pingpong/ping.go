package main

import (
	"sync"
	"time"

	"github.com/mogud/snow/routines/node"
)

type ping struct {
	node.Service

	closeChan chan struct{}
	pongProxy node.IProxy
}

func (ss *ping) ConstructPing() {
	ss.Infof("ping construct")
}

func (ss *ping) Start(_ any) {
	ss.Infof("ping start")

	ss.closeChan = make(chan struct{})
	ss.pongProxy = ss.CreateProxy("Pong")

	go func() {
		ticker := time.NewTicker(3 * time.Second)
	loop:
		for {
			select {
			case <-ticker.C:
				ss.Fork("rpc", func() {
					ss.Infof("=====================")
					ss.pongProxy.Call("Hello", "ping").
						Then(func(ret string) {
							ss.Infof("received: %s", ret)
						}).Done()
				})
			case <-ss.closeChan:
				break loop
			}
		}
	}()
}

func (ss *ping) Stop(_ *sync.WaitGroup) {
	ss.Infof("ping stop")
	close(ss.closeChan)
}

func (ss *ping) AfterStop() {
	ss.Infof("ping after-stop")
}
