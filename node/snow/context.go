package snow

import (
	"gitee.com/mogud/snow/node"
)

var _ = (node.IRpcContext)((*rpcContext)(nil))
var EmptyContext = (node.IRpcContext)((*rpcContext)(nil))

type rpcContext struct {
	mRsp    *message
	mReq    *message
	srv     *Service
	flushed bool
	flushCb func()
}

func newContext(srv *Service, mRsp, mReq *message, flushCb func()) *rpcContext {
	return &rpcContext{
		mRsp:    mRsp,
		mReq:    mReq,
		srv:     srv,
		flushCb: flushCb,
	}
}

func (ss *rpcContext) GetRemoteNodeAddr() node.INodeAddr {
	return ss.mReq.naddr
}

func (ss *rpcContext) GetRemoteServiceAddr() int32 {
	return ss.mReq.src
}

func (ss *rpcContext) Return(args ...interface{}) {
	if ss == nil {
		return
	}
	ss.mRsp.writeResponse(args...)
	ss.flush()
}

func (ss *rpcContext) Error(err error) {
	if ss == nil {
		return
	}
	ss.mRsp.writeError(err)
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
		} else if mReq.naddr != 0 { // must be remote message
			sender := nodeGetMessageSender(mReq.naddr, mReq.src, false, nil)
			if sender != nil {
				sender.send(mRsp)
			} else {
				ss.srv.Errorf("service at naddr(%v) saddr(%#8x) not found when rpc return", mReq.naddr, mReq.src)
			}
		} // else is a local post
	}
}
