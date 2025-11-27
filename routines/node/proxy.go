package node

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"time"
)

var (
	ErrServiceNotExist      = fmt.Errorf("service not exist")
	ErrNodeMessageChanFull  = fmt.Errorf("note message chan full")
	ErrRequestTimeoutRemote = fmt.Errorf("session timeout from remote")
	ErrRequestTimeoutLocal  = fmt.Errorf("session timeout from local")
)

type iProxy interface {
	IProxy

	doCall(*promise)
}

var _ = iProxy((*serviceProxy)(nil))

type serviceProxy struct {
	srv          *Service
	nAddr        Addr
	nAddrUpdater *AddrUpdater
	sAddr        int32
	sender       iMessageSender

	bufferFullCB func()
	buffer       []*promise
}

func (ss *serviceProxy) Call(fName string, args ...any) IPromise {
	if ss.sAddr == 0 {
		return (*dumbPromise)(nil)
	}

	return newPromise(ss, fName, args)
}

func (ss *serviceProxy) GetNodeAddr() INodeAddr {
	if ss.sAddr == 0 {
		return AddrInvalid
	}

	if ss.nAddrUpdater != nil {
		return ss.nAddrUpdater.GetNodeAddr()
	}

	return ss.nAddr
}

func (ss *serviceProxy) Avail() bool {
	return ss.sAddr != 0
}

func (ss *serviceProxy) doCall(p *promise) {
	if ss.sAddr == 0 {
		if ss.buffer != nil {
			if cap(ss.buffer) == len(ss.buffer) && ss.bufferFullCB != nil {
				ss.bufferFullCB()
				return
			}
			ss.buffer = append(ss.buffer, p)
		}
		return
	}

	if p.timeout == -1 {
		p.timeout = 30 * time.Second
	}
	srv := ss.srv

	m := &message{
		timeout: p.timeout,
		src:     srv.GetAddr(),
		dst:     ss.sAddr,
		// TODO trace id
	}
	m.writeRequest(p.fName, p.args)

	if ss.sender == nil || ss.sender.closed() {
		var ch chan<- bool
		if ss.nAddrUpdater != nil {
			ch = ss.nAddrUpdater.getSigChan()
		}
		ss.sender = nodeGetMessageSender(ss.GetNodeAddr().(Addr), ss.sAddr, true, ch)
	}
	if ss.sender == nil {
		if p.errCb != nil {
			srv.Fork("proxy.err.cb", func() {
				p.errCb(ErrServiceNotExist)
			})
		} else {
			srv.Errorf("rpc(%s) uncatched error: %+v", p.fName, ErrServiceNotExist)
		}
		if p.finalCb != nil {
			srv.Fork("proxy.err.finalCb", func() {
				p.finalCb()
				p.clear()
			})
		} else {
			p.clear()
		}
		m.clear()
		return
	}

	if p.successCb == nil {
		if p.finalCb != nil {
			srv.Fork("proxy.post.finalCb", func() {
				p.finalCb()
				p.clear()
			})
		} else {
			p.clear()
		}
	} else {
		sess := nodeGenSessionID()
		trace := m.trace
		cb := func(mm *message) {
			ss.callThen(mm, srv, p, sess)
		}
		m.cb = cb
		m.sess = sess
		timeout := p.timeout
		if timeout > 0 {
			srv.Fork("proxy.timeoutCallBack", func() {
				srv.After(timeout, func() {
					om := &message{
						trace: trace,
						err:   ErrRequestTimeoutLocal,
					}
					cb(om)
				})
			})
		}
	}

	ss.sender.send(m)
}

func (ss *serviceProxy) callThen(mm *message, srv *Service, p *promise, sess int32) {
	srv.Fork("proxy.forkCb", func() {
		if p.timeout == -1 {
			return
		}
		p.timeout = -1

		defer func() {
			if p.finalCb != nil {
				p.finalCb()
			}
			p.clear()
		}()

		if mm.src == 0 { // error occurs
			err := mm.getError()
			if p.errCb != nil {
				p.errCb(err)
			} else {
				srv.Errorf("rpc(%s:%v) uncatched error: %+v", p.fName, sess, err)
			}
			return
		}

		fv := reflect.ValueOf(p.successCb)
		if !fv.IsValid() {
			srv.Errorf("rpc(%s:%v) invalid success callback", p.fName, sess)
			return
		}

		ft := fv.Type()
		fArgs, err := mm.getResponse(ft)
		if err != nil {
			srv.Errorf("rpc(%s:%v) response error: %+v", p.fName, sess, err)
			return
		}

		panicked := true
		defer func() {
			if panicked {
				if e := recover(); e != nil {
					srv.Errorf("rpc(%s:%v) response got panic: %v => %v", p.fName, sess, e, string(debug.Stack()))
				} else {
					srv.Errorf("rpc(%s:%v) response got panic: %v", p.fName, sess, string(debug.Stack()))
				}
			}
		}()

		fRet := fv.Call(fArgs)

		for _, arg := range fArgs {
			if arg.CanAddr() {
				arg.SetZero()
			}
		}
		fArgs = nil
		for _, arg := range fRet {
			if arg.CanAddr() {
				arg.SetZero()
			}
		}
		fRet = nil

		panicked = false
	})
}
