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

var (
	dumbPromiseObj = iPromise((*dumbPromise)(nil))
	dumbProxyObj   = &serviceProxy{}
)

type iPromise interface {
	IPromise

	setCallBacks(successCb []any, errCb func(error), finalCb func())
}

type dumbPromise struct {
}

func (ss *dumbPromise) Then(_ any) IPromise {
	return ss
}

func (ss *dumbPromise) Catch(_ func(error)) IPromise {
	return ss
}

func (ss *dumbPromise) Final(_ func()) IPromise {
	return ss
}

func (ss *dumbPromise) Timeout(_ time.Duration) IPromise {
	return ss
}

func (ss *dumbPromise) Done() {
}

func (ss *dumbPromise) setCallBacks(_ []any, _ func(error), _ func()) {
}

var _ iPromise = (*promise)(nil)

type promise struct {
	proxy     iProxy
	fName     string
	timeout   time.Duration
	args      []any
	successCb []any
	errCb     func(error)
	finalCb   func()
}

func newPromise(proxy iProxy, fName string, args []any) *promise {
	return &promise{
		proxy:   proxy,
		fName:   fName,
		timeout: -1,
		args:    args,
	}
}

func (ss *promise) Then(f any) IPromise {
	ss.successCb = append(ss.successCb, f)
	return ss
}

func (ss *promise) Catch(f func(error)) IPromise {
	ss.errCb = f
	return ss
}

func (ss *promise) Final(f func()) IPromise {
	ss.finalCb = f
	return ss
}

func (ss *promise) Timeout(timeout time.Duration) IPromise {
	ss.timeout = timeout
	return ss
}

func (ss *promise) Done() {
	ss.proxy.doCall(ss)
}

func (ss *promise) setCallBacks(successCb []any, errCb func(error), finalCb func()) {
	ss.successCb = successCb
	ss.errCb = errCb
	ss.finalCb = finalCb
}

type iProxy interface {
	IProxy

	doCall(*promise)
}

var _ = iProxy((*serviceProxy)(nil))

type serviceProxy struct {
	srv          *Service
	nAddr        Addr
	nAddrUpdater *NodeAddrUpdater
	sAddr        int32
	sender       iMessageSender

	bufferFullCB func()
	buffer       []*promise
}

func newServiceProxy(srv *Service, updater *NodeAddrUpdater, nAddr Addr, sAddr int32) *serviceProxy {
	if nAddr.IsLocalhost() || nAddr == Config.CurNodeAddr {
		nAddr = AddrLocal
	}

	// 根据节点列表自动查找指定类型的服务
	if nAddr == addrAutoSearch && sAddr < 0 {
		kind := -sAddr
		name := gNode.kind2Info[kind].Name
		if Config.CurNodeMap[name] {
			nAddr = AddrLocal
		} else {
		loop:
			for _, ni := range Config.Nodes {
				if ni.Name == Config.CurNodeName {
					continue
				}

				for _, n := range ni.Services {
					if n == name && len(ni.Host) > 0 && ni.Port > 0 {
						nAddr = ni.NodeAddr
						break loop
					}
				}
			}
		}

		if nAddr == addrAutoSearch {
			return nil
		}
	}

	return &serviceProxy{
		srv:          srv,
		nAddr:        nAddr,
		nAddrUpdater: updater,
		sAddr:        sAddr,
	}
}

func (ss *serviceProxy) Call(fName string, args ...any) IPromise {
	if ss.sAddr == 0 {
		return dumbPromiseObj
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
			srv.Fork("proxy.err.finalCb", p.finalCb)
		}
		return
	}

	if len(p.successCb) == 0 {
		if p.finalCb != nil {
			srv.Fork("proxy.post.finalCb", p.finalCb)
		}
	} else {
		m.sess = nodeGenSessionID()
		m.cb = func(mm *message) {
			ss.callThen(mm, srv, p, m.sess)
		}
		if p.timeout > 0 {
			srv.Fork("proxy.timeoutCallBack", func() {
				srv.After(p.timeout, func() {
					om := &message{
						trace: m.trace,
						err:   ErrRequestTimeoutLocal,
					}
					m.cb(om)
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

		nextProxy := false
		if p.finalCb != nil {
			defer func() {
				if !nextProxy {
					srv.Fork("proxy.req.finalCb", p.finalCb)
				}
			}()
		}

		if mm.src == 0 { // error occurs
			err := mm.getError()
			if p.errCb != nil {
				p.errCb(err)
			} else {
				srv.Errorf("rpc(%s:%v) uncatched error: %+v", p.fName, sess, err)
			}
			return
		}

		fv := reflect.ValueOf(p.successCb[0])
		if !fv.IsValid() {
			return
		}

		ft := fv.Type()
		p.successCb = p.successCb[1:]

		ret, err := mm.getResponse(ft)
		if err != nil {
			srv.Fatalf("rpc(%s:%v) response error: %+v", p.fName, sess, err)
		}

		panicked := true
		defer func() {
			if panicked {
				srv.Errorf("rpc(%s:%v) response got panic: %v", p.fName, sess, string(debug.Stack()))
			}
		}()
		fret := fv.Call(ret)
		for len(p.successCb) > 0 {
			if ft.NumOut() == 1 && ft.Out(0) == reflect.TypeOf((*IPromise)(nil)).Elem() {
				if fret[0].IsNil() {
					break
				}
				np := fret[0].Interface().(iPromise)
				np.setCallBacks(p.successCb, p.errCb, p.finalCb)
				np.Done()
				nextProxy = true
				break
			}

			fv = reflect.ValueOf(p.successCb[0])
			ft = fv.Type()
			fret = fv.Call(fret)
			p.successCb = p.successCb[1:]
		}
		panicked = false
	})
}
