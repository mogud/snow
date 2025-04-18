package node

import "time"

type IPromise interface {
	Then(f any) IPromise
	Catch(f func(error)) IPromise
	Final(f func()) IPromise
	Timeout(timeout time.Duration) IPromise
	Done()
}

type IProxy interface {
	Call(name string, args ...any) IPromise
	GetNodeAddr() INodeAddr
	Avail() bool
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
	Catch(f func(error)) IRpcContext
	Return(args ...any)
	Error(error)
}

type INodeAddr interface {
	IsLocalhost() bool
	GetIPString() string
	String() string
}
