package main

import (
	"sync"

	"github.com/mogud/snow/routines/node"
)

type pong struct {
	node.Service
}

func (ss *pong) ConstructPong() {
	ss.Infof("pong construct")
}

func (ss *pong) Start(_ any) {
	ss.Infof("pong start")
	ss.EnableRpc()
}

func (ss *pong) Stop(_ *sync.WaitGroup) {
	ss.Infof("pong stop")
}

func (ss *pong) AfterStop() {
	ss.Infof("pong after-stop")
}

func (ss *pong) RpcHello(ctx node.IRpcContext, msg string) {
	ss.Infof("received: %s", msg)
	ctx.Return("pong")
}
