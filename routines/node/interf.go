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

type ITimeWheelHandle interface {
	Stop()
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

type IMetricCollector interface {
	// Gauge 仪表，设置值
	Gauge(name string, val int64)
	// Counter 计数器，累加值
	Counter(name string, val uint64)
	// Histogram 直方图，累加，但值为浮点数，可为正负
	Histogram(name string, val float64)
}
