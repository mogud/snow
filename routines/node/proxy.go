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
	dumbServiceProxyObj = newEmptyServiceProxy()
	dumbPromiseObj      = iPromise((*dumbPromise)(nil))
)

type iPromise interface {
	IPromise

	setCallBacks(succCb []interface{}, errCb func(error), finalCb func())
}

type dumbPromise struct {
}

func (ss *dumbPromise) Then(f interface{}) IPromise {
	return ss
}

func (ss *dumbPromise) Catch(f func(error)) IPromise {
	return ss
}

func (ss *dumbPromise) Final(f func()) IPromise {
	return ss
}

func (ss *dumbPromise) Timeout(timeout time.Duration) IPromise {
	return ss
}

func (ss *dumbPromise) Done() {
}

func (ss *dumbPromise) setCallBacks(succCb []interface{}, errCb func(error), finalCb func()) {
}

var _ iPromise = (*promise)(nil)

type promise struct {
	proxy   *serviceProxy
	fname   string
	timeout time.Duration
	args    []interface{}
	succCb  []interface{}
	errCb   func(error)
	finalCb func()
}

func newPromise(proxy *serviceProxy, fname string, args []interface{}) *promise {
	return &promise{
		proxy:   proxy,
		fname:   fname,
		timeout: -1,
		args:    args,
	}
}

func (ss *promise) Then(f interface{}) IPromise {
	ss.succCb = append(ss.succCb, f)
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

func (ss *promise) setCallBacks(succCb []interface{}, errCb func(error), finalCb func()) {
	ss.succCb = succCb
	ss.errCb = errCb
	ss.finalCb = finalCb
}

var _ = IProxy((*serviceProxy)(nil))

type serviceProxy struct {
	srv          *Service
	naddr        NodeAddr
	naddrUpdater *NodeAddrUpdater
	saddr        int32
	sender       iMessageSender

	bufferFullCB func()
	buffer       []*promise
}

func newEmptyServiceProxy() *serviceProxy {
	return &serviceProxy{}
}

func newServiceProxy(srv *Service, updater *NodeAddrUpdater, naddr NodeAddr, saddr int32) *serviceProxy {
	if naddr.IsLocalhost() {
		naddr = 0
	}

	if saddr < 0 && naddr == -1 { // 根据节点列表自动查找指定类型的服务
		kind := -saddr
		name := gNode.kind2Info[kind].Name
		if NodeConfig.CurNodeMap[name] {
			naddr = 0
		} else {
			for _, ni := range NodeConfig.Nodes {
				for _, n := range ni.Services {
					if n == name {
						naddr = ni.NodeAddr
						break
					}
				}
			}
		}
		if naddr == -1 {
			srv.Fatalf("service (%s) not found in current node config", name)
		}
	}

	return &serviceProxy{
		srv:          srv,
		naddr:        naddr,
		naddrUpdater: updater,
		saddr:        saddr,
	}
}

func (ss *serviceProxy) Call(fname string, args ...interface{}) IPromise {
	if ss.saddr == 0 {
		return dumbPromiseObj
	}

	return newPromise(ss, fname, args)
}

func (ss *serviceProxy) GetNodeAddr() INodeAddr {
	if ss.saddr == 0 {
		return NodeAddrInvalid
	}

	if ss.naddrUpdater != nil {
		return ss.naddrUpdater.GetNodeAddr()
	}

	return ss.naddr
}

func (ss *serviceProxy) Avail() bool {
	return ss.saddr != 0
}

func (ss *serviceProxy) Reset(proxy IProxy) {
	if proxy == nil {
		ss.saddr = 0
		return
	}

	p := proxy.(*serviceProxy)
	ss.srv = p.srv
	ss.naddr = p.naddr
	ss.naddrUpdater = p.naddrUpdater
	ss.saddr = p.saddr
	ss.sender = p.sender
}

func (ss *serviceProxy) AddBuffer(n int, fullcb func()) {
	ss.bufferFullCB = fullcb
	ss.buffer = make([]*promise, 0, n)
}

func (ss *serviceProxy) doCall(p *promise) {
	if ss.saddr == 0 {
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
		dst:     ss.saddr,
	}
	m.writeRequest(p.fname, p.args)

	if ss.sender == nil || ss.sender.closed() {
		var ch chan<- bool
		if ss.naddrUpdater != nil {
			ch = ss.naddrUpdater.getSigChan()
		}
		ss.sender = nodeGetMessageSender(ss.GetNodeAddr().(NodeAddr), ss.saddr, true, ch)
	}
	if ss.sender == nil {
		if p.errCb != nil {
			srv.Fork("proxy.err.cb", func() {
				p.errCb(ErrServiceNotExist)
			})
		} else {
			srv.Errorf("rpc(%s) uncatched error: %+v", p.fname, ErrServiceNotExist)
		}
		if p.finalCb != nil {
			srv.Fork("proxy.err.finalcb", p.finalCb)
		}
		return
	}

	if len(p.succCb) == 0 {
		if p.finalCb != nil {
			srv.Fork("proxy.post.finalcb", p.finalCb)
		}
	} else {
		m.sess = nodeGenSessionID()
		m.cb = func(mm *message) {
			srv.Fork("proxy.forkcb", func() {
				if p.timeout == -1 {
					return
				}
				p.timeout = -1

				nextProxy := false
				if p.finalCb != nil {
					defer func() {
						if !nextProxy {
							srv.Fork("proxy.req.finalcb", p.finalCb)
						}
					}()
				}

				if mm.src == 0 { // error occurs

					err := mm.getError()
					if p.errCb != nil {
						p.errCb(err)
					} else {
						srv.Errorf("rpc(%s:%v) uncatched error: %+v", p.fname, m.sess, err)
					}
					return
				}

				fv := reflect.ValueOf(p.succCb[0])
				if !fv.IsValid() {
					return
				}

				ft := fv.Type()
				p.succCb = p.succCb[1:]

				ret, err := mm.getResponse(ft)
				if err != nil {
					srv.Fatalf("rpc(%s:%v) response error: %+v", p.fname, m.sess, err)
				}

				panicked := true
				defer func() {
					if panicked {
						srv.Errorf("rpc(%s:%v) response got panic: %v", p.fname, m.sess, string(debug.Stack()))
					}
				}()
				fret := fv.Call(ret)
				for len(p.succCb) > 0 {
					if ft.NumOut() == 1 && ft.Out(0) == reflect.TypeOf((*IPromise)(nil)).Elem() {
						if fret[0].IsNil() {
							break
						}
						np := fret[0].Interface().(iPromise)
						np.setCallBacks(p.succCb, p.errCb, p.finalCb)
						np.Done()
						nextProxy = true
						break
					}

					fv = reflect.ValueOf(p.succCb[0])
					ft = fv.Type()
					fret = fv.Call(fret)
					p.succCb = p.succCb[1:]
				}
				panicked = false
			})
		}
		if p.timeout > 0 {
			srv.Fork("proxy.timeoutcb", func() {
				srv.After(p.timeout, func() {
					om := &message{}
					om.writeError(ErrRequestTimeoutLocal)
					m.cb(om)
				})
			})
		}
	}

	ss.sender.send(m)
}
