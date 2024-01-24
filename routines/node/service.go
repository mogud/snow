package node

import (
	"fmt"
	"gitee.com/mogud/snow/core/debug"
	"gitee.com/mogud/snow/core/logging"
	"gitee.com/mogud/snow/core/task"
	"gitee.com/mogud/snow/core/ticker"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const tickIntervalNs = 100_000_000

var _ iService = (*Service)(nil)
var _ iMessageSender = (*Service)(nil)

// Methods of Service overridable
type iService interface {
	Start(arg interface{})
	Stop(wg *sync.WaitGroup)
	Entry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) func()

	getService() *Service
}

type Service struct {
	MethodMap map[string]reflect.Value

	realSrv   iService
	name      string
	kind      int32
	saddr     int32
	msgQueue  *msgQueue
	msgChan   chan *message
	funcQueue *funcQueue
	funcChan  chan *TagFunc

	nowNs       int64
	timeWheel   *timeWheel
	delayedRPCs []func()
	allowedRPCs map[string]bool
	logger      logging.ILogger
	load        int64

	closedLock int32
	wg         *sync.WaitGroup
}

const chanSize = 1024

func (ss *Service) getService() *Service {
	return ss
}

func (ss *Service) Construct(logger *logging.Logger[Service]) {
	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = ss.GetName()
		data.ID = fmt.Sprintf("%X", unsafe.Pointer(ss))
	})
}

func (ss *Service) init(name string, kind int32, saddr int32, realService iService, methodMap map[string]reflect.Value) {
	ss.name = name
	ss.kind = kind
	ss.saddr = saddr
	ss.realSrv = realService
	ss.MethodMap = methodMap
	ss.wg = &sync.WaitGroup{}

	ss.msgQueue = newMsgQueue()
	ss.msgChan = make(chan *message, chanSize)
	ss.funcQueue = newFuncQueue()
	ss.funcChan = make(chan *TagFunc, chanSize)

	ss.timeWheel = &timeWheel{
		queue:  &timerQueue{},
		timeMs: time.Now().UnixMilli(),
	}
	ss.delayedRPCs = make([]func(), 0, 4)
	ss.allowedRPCs = make(map[string]bool)
}

func (ss *Service) start(arg interface{}) {
	ss.wg.Add(1)

	ss.nowNs = time.Now().UnixNano()

	// TODO
	// rnd := rand.Int63n(5)
	// prevTs := ss.GetSecond()

	ss.realSrv.Start(arg)

	isStandalone := NodeConfig.CurNodeMap[ss.GetName()]
	if isStandalone {
		ss.Infof("start success")
	}

	task.Execute(func() {
		if ss.closed() {
			ss.wg.Done()
			return
		}

		tch := make(chan int64, chanSize)
		ticker.Subscribe(int64(ss.GetAddr()), tch)

	L:
		for {
			select {
			case msg := <-ss.msgChan:
				ss.doDispatch(msg)
			case f := <-ss.funcChan:
				// TODO
				// ss.metric.Call("RecordTimer", f.Tag, int64(1), int64(100), int64(5)).Done()
				ss.doFunc(f)
			case unixNano := <-tch: // tick
				n := len(tch)
				for i := 0; i < n; i++ {
					unixNano = <-tch
				}

				ss.nowNs = unixNano
				ss.timeWheel.Update(ss.GetMillisecond())

				if unixNano-ss.nowNs >= tickIntervalNs {
					// TODO
					// if app.DevMode {
					// 	nowTs := ss.GetSecond() + rnd
					// 	deltaTs := nowTs - prevTs
					// 	if ss.load > 0 && deltaTs > 5 {
					// 		prevTs = nowTs
					// 		name := "Load(us) - " + ss.GetName()
					// 		ss.metric.Call("RecordHistogramTime", name, ss.load, int64(100*1000), int64(8)).Done()
					// 	}
					// }

					if len(ss.msgChan) == 0 {
						for msg := ss.msgQueue.deq(); msg != nil; msg = ss.msgQueue.deq() {
							ss.doDispatch(msg)
						}
					}

					if len(ss.funcChan) == 0 {
						for f := ss.funcQueue.deq(); f != nil; f = ss.funcQueue.deq() {
							// TODO
							// ss.metric.Call("RecordTimer", f.Tag, int64(1), int64(100), int64(5)).Done()
							ss.doFunc(f)
						}
					}
				}

				if ss.closed() {
					break L
				}
			}
		}

		ticker.Unsubscribe(int64(ss.GetAddr()))
		ss.wg.Done()
	})
}

func (ss *Service) stop() {
	if !atomic.CompareAndSwapInt32(&ss.closedLock, 0, 1) {
		return
	}

	// 这里开始退出流程，当前还未处于关闭状态
	isStandalone := NodeConfig.CurNodeMap[ss.GetName()]
	if isStandalone {
		ss.Debugf("stop...")
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	ss.Fork("service.stop", func() {
		ss.realSrv.Stop(wg)
		wg.Done()
	})
	wg.Wait()

	atomic.StoreInt32(&ss.closedLock, 2)
	ss.wg.Wait()

	if isStandalone {
		ss.Infof("stop success")
	}
}

func (ss *Service) closed() bool {
	return atomic.LoadInt32(&ss.closedLock) == 2
}

func (ss *Service) GetName() string {
	return ss.name
}

func (ss *Service) GetKind() int32 {
	return ss.kind
}

func (ss *Service) GetAddr() int32 {
	return ss.saddr
}

func (ss *Service) GetLoad() float64 {
	return float64(ss.load) / 1000
}

// TODO
// func (ss *Service) GetMetric() core.IProxy {
// 	return ss.metric
// }

func (ss *Service) send(msg *message) bool {
	if !ss.msgQueue.empty() {
		ss.msgQueue.enq(msg)
	} else {
		select {
		case ss.msgChan <- msg:
		default:
			ss.msgQueue.enq(msg)
		}
	}
	return true
}

func (ss *Service) doFunc(f *TagFunc) {
	defer func() {
		if err := recover(); err != nil {
			buf := debug.StackInfo()
			ss.Errorf("service execute anonymous function error: %v\n%s", err, buf)
		}
	}()
	f.F()
}

func (ss *Service) doDispatch(mReq *message) {
	var funcName string
	defer func() {
		if err := recover(); err != nil {
			buf := debug.StackInfo()
			if len(funcName) == 0 {
				ss.Errorf("service execute anonymous function error: %v\n%s", err, buf)
			} else {
				ss.Errorf("service execute function %s error: %v\n%s", funcName, err, buf)
			}
		}
	}()

	var err error
	funcName, err = mReq.getRequestFunc()
	if err != nil {
		ss.Errorf("doDispatch: function name error: %+v", err)
		return
	}

	mRsp := &message{
		naddr: mReq.naddr,
		src:   ss.GetAddr(),
		dst:   mReq.src,
		sess:  -mReq.sess,
	}
	var cb func()
	// TODO
	// if app.DevMode {
	// 	start := time.Now()
	// 	cb = func() {
	// 		if mRsp.sess != 0 {
	// 			name := "RPC(us) - " + ss.GetName() + " " + funcName
	// 			us := int64(time.Now().Sub(start) / time.Microsecond)
	// 			ss.metric.Call("RecordMeter", name, us).Done()
	// 		}
	// 	}
	// }

	ctx := newContext(ss, mRsp, mReq, cb)
	if ss.delayedRPCs == nil || ss.allowedRPCs[funcName] {
		ss.entry(ctx, funcName, mReq.getRequestFuncArgs)
	} else {
		ss.delayEntry(ctx, funcName, mReq.getRequestFuncArgs)
	}
}

func (ss *Service) Start(_ interface{}) {
}

func (ss *Service) Stop(_ *sync.WaitGroup) {
}

func (ss *Service) SetAllowedRPC(names []string) {
	for _, n := range names {
		ss.allowedRPCs[n] = true
	}
}

func (ss *Service) EnableRPC() {
	for _, f := range ss.delayedRPCs {
		f()
	}
	ss.allowedRPCs = nil
	ss.delayedRPCs = nil
}

func (ss *Service) delayEntry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) {
	if f := ss.realSrv.Entry(ctx, funcName, argGetter); f != nil {
		ss.delayedRPCs = append(ss.delayedRPCs, f)
	}
}

func (ss *Service) entry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) {
	if f := ss.realSrv.Entry(ctx, funcName, argGetter); f != nil {
		f()
	}
}

func (ss *Service) Entry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) func() {
	f := ss.MethodMap[funcName]
	args, err := argGetter(f.Type())
	if err != nil {
		ss.Errorf("doDispatch: get args error: %+v  name = %v", err, funcName)
		return nil
	}
	cargs := []reflect.Value{reflect.ValueOf(ss.realSrv), reflect.ValueOf(ctx)}
	return func() {
		f.Call(append(cargs, args...))
	}
}

func (ss *Service) Fork(tag string, f func()) {
	tf := &TagFunc{Tag: tag, F: f}
	if !ss.funcQueue.empty() {
		ss.funcQueue.enq(tf)
	} else {
		select {
		case ss.funcChan <- tf:
		default:
			ss.funcQueue.enq(tf)
		}
	}
}

func (ss *Service) GetTime() time.Time {
	return time.Unix(0, ss.nowNs)
}

func (ss *Service) GetSecond() int64 {
	return ss.nowNs / 1000_000_000
}

func (ss *Service) GetMillisecond() int64 {
	return ss.nowNs / 1000_000
}

func (ss *Service) Tick(interval, delay time.Duration, f func()) ITimer {
	return ss.timeWheel.Tick(interval, delay, f)
}

func (ss *Service) TickDelayRandom(interval time.Duration, f func()) ITimer {
	delay := time.Duration(rand.Int63n(int64(interval)))
	return ss.timeWheel.Tick(interval, delay, f)
}

func (ss *Service) TickAfter(interval time.Duration, f func(func())) ITimer {
	return ss.timeWheel.After(interval, func() {
		f(func() {
			ss.TickAfter(interval, f)
		})
	})
}

func (ss *Service) After(delay time.Duration, f func()) ITimer {
	return ss.timeWheel.After(delay, f)
}

func (ss *Service) NewService(name string) (int32, error) {
	return NewService(name)
}

func (ss *Service) CreateProxyDefault() IProxy {
	return newEmptyServiceProxy()
}

func (ss *Service) CreateProxy(name string) IProxy {
	kind := gNode.name2Info[name].Kind
	return newServiceProxy(ss, nil, -1, -kind)
}

func (ss *Service) CreateProxyByNodeKind(naddr INodeAddr, name string) IProxy {
	kind := gNode.name2Info[name].Kind
	return newServiceProxy(ss, nil, naddr.(NodeAddr), -kind)
}

func (ss *Service) CreateProxyByNodeAddr(naddr INodeAddr, saddr int32) IProxy {
	return newServiceProxy(ss, nil, naddr.(NodeAddr), saddr)
}

func (ss *Service) CreateProxyByUpdaterKind(naddrUpdater *NodeAddrUpdater, name string) IProxy {
	kind := gNode.name2Info[name].Kind
	return newServiceProxy(ss, naddrUpdater, 0, -kind)
}

func (ss *Service) CreateProxyByUpdaterAddr(naddrUpdater *NodeAddrUpdater, saddr int32) IProxy {
	return newServiceProxy(ss, naddrUpdater, 0, saddr)
}

func (ss *Service) RpcStatus(ctx IRpcContext) {
	ctx.Return("OK")
}

func (ss *Service) RpcReload(ctx IRpcContext) {
	ctx.Return()
}

func (ss *Service) Tracef(format string, args ...any) {
	ss.logger.Tracef(format, args...)
}

func (ss *Service) Debugf(format string, args ...any) {
	ss.logger.Debugf(format, args...)
}

func (ss *Service) Infof(format string, args ...any) {
	ss.logger.Infof(format, args...)
}

func (ss *Service) Warnf(format string, args ...any) {
	ss.logger.Warnf(format, args...)
}

func (ss *Service) Errorf(format string, args ...any) {
	ss.logger.Errorf(format, args...)
}

func (ss *Service) Fatalf(format string, args ...any) {
	ss.logger.Fatalf(format, args...)
}
