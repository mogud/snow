package node

var _ = (IRpcContext)((*rpcContext)(nil))

type rpcContext struct {
	mRsp    *message
	mReq    *message
	srv     *Service
	flushed bool
	flushCb func()
}

func newRpcContext(srv *Service, mRsp, mReq *message, flushCb func()) *rpcContext {
	return &rpcContext{
		mRsp:    mRsp,
		mReq:    mReq,
		srv:     srv,
		flushCb: flushCb,
	}
}

func (ss *rpcContext) GetRemoteNodeAddr() INodeAddr {
	return ss.mReq.nAddr
}

func (ss *rpcContext) GetRemoteServiceAddr() int32 {
	return ss.mReq.src
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

	mReq := ss.mReq
	mRsp := ss.mRsp
	if mReq.sess > 0 {
		if mReq.cb != nil { // local service message
			mReq.cb(mRsp)
		} else if mReq.nAddr != 0 { // must be remote message
			sender := nodeGetMessageSender(mReq.nAddr, mReq.src, false, nil)
			if sender != nil {
				sender.send(mRsp)
			} else {
				ss.srv.Errorf("service at nAddr(%v) sAddr(%#8x) not found when rpc return", mReq.nAddr, mReq.src)
			}
		} // else is a local post
	}
}
