package node

var _ = (IRpcContext)((*rpcContext)(nil))

type rpcContext struct {
	reqSess     int32
	reqSrc      int32
	reqCb       func(m *message)
	reqNodeAddr Addr

	mRsp    *message
	srv     *Service
	flushed bool
	flushCb func()
}

func newRpcContext(srv *Service, mRsp *message, reqSess, reqSrc int32, reqNodeAddr Addr, reqCb func(m *message), flushCb func()) *rpcContext {
	return &rpcContext{
		reqSess:     reqSess,
		reqSrc:      reqSrc,
		reqNodeAddr: reqNodeAddr,
		reqCb:       reqCb,

		mRsp:    mRsp,
		srv:     srv,
		flushCb: flushCb,
	}
}

func (ss *rpcContext) GetRemoteNodeAddr() INodeAddr {
	return ss.reqNodeAddr
}

func (ss *rpcContext) GetRemoteServiceAddr() int32 {
	return ss.reqSrc
}

func (ss *rpcContext) Catch(f func(error)) IRpcContext {
	ss.mRsp.cb = func(m *message) {
		f(m.err)
	}
	return ss
}

func (ss *rpcContext) Return(args ...any) {
	ss.mRsp.writeResponse(args...)
	ss.flush()
}

func (ss *rpcContext) Error(err error) {
	ss.mRsp.err = err
	ss.mRsp.src = 0
	ss.flush()
}

func (ss *rpcContext) flush() {
	if ss.flushed {
		return
	}
	ss.flushed = true
	if ss.flushCb != nil {
		ss.flushCb()
	}

	reqSess := ss.reqSess
	reqSrc := ss.reqSrc
	reqNodeAddr := ss.reqNodeAddr
	reqCb := ss.reqCb
	mRsp := ss.mRsp
	if reqSess > 0 {
		if reqCb != nil { // local service message
			reqCb(mRsp)
		} else if reqNodeAddr != 0 { // must be remote message
			sender := nodeGetMessageSender(reqNodeAddr, reqSrc, false, nil)
			if sender != nil {
				sender.send(mRsp)
			} else {
				ss.srv.Errorf("service at nAddr(%v) sAddr(%#8x) not found when rpc return", reqNodeAddr, reqSrc)
				mRsp.clear()
			}
		} // else is a local post
	}
}
