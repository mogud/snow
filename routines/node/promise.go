package node

import "time"

var _ IPromise = (*dumbPromise)(nil)

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

var _ IPromise = (*promise)(nil)

func _emptyThen() {}

type promise struct {
	proxy     iProxy
	fName     string
	timeout   time.Duration
	args      []any
	successCb any
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
	if f == nil {
		f = _emptyThen
	}
	ss.successCb = f
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

func (ss *promise) clear() {
	ss.args = nil
	ss.successCb = nil
	ss.errCb = nil
	ss.finalCb = nil
}
