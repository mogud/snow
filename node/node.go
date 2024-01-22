package node

import "time"

type IPromise interface {
	Then(f interface{}) IPromise
	Catch(f func(error)) IPromise
	Final(f func()) IPromise
	Timeout(timeout time.Duration) IPromise
	Done()
}

type IProxy interface {
	Call(fname string, args ...interface{}) IPromise
	GetNodeAddr() INodeAddr
	Avail() bool
	Reset(proxy IProxy)
	AddBuffer(n int, fullcb func())
}

type ITimer interface {
	Start()
	Stop()
	SetInterval(interval time.Duration)
	SetDelay(delay time.Duration)
	SetFunc(f func())
	Set(interval, delay time.Duration, f func())
}

type IRpcContext interface {
	GetRemoteNodeAddr() INodeAddr
	GetRemoteServiceAddr() int32
	Return(args ...interface{})
	Error(error)
}

type INodeAddr interface {
	IsLocalhost() bool
	GetIPString() string
	String() string
}
