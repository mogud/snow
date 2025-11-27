package task

import (
	"fmt"

	"github.com/panjf2000/ants/v2"
)

var p *ants.PoolWithFunc

func init() { // TODO
	var err error
	p, err = ants.NewPoolWithFunc(100000, func(f any) {
		(f.(func()))()
	}, ants.WithPreAlloc(true))

	if err != nil {
		panic(fmt.Sprintf("init goroutine pool: %v", err))
	}
}

func Execute(f func()) {
	//go f()
	_ = p.Invoke(f)
}
