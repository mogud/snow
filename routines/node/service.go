package node

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"snow/core/debug"
	"snow/core/logging"
	"snow/core/task"
	"snow/core/ticker"
	http2 "snow/routines/http"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var _ iService = (*Service)(nil)
var _ iMessageSender = (*Service)(nil)

// Methods of Service overridable
type iService interface {
	// Start 服务启动时调用
	Start(arg any)
	// Stop 服务关闭时调用，保证与服务器的消息处理在同一线程
	Stop(wg *sync.WaitGroup)
	// AfterStop 服务关闭后调用，此时任何消息都已关闭
	AfterStop()
	// Entry 用于自定义 RPC 处理
	Entry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) func()

	getService() *Service
}

type Service struct {
	methodMap     map[string]reflect.Value
	httpMethodMap map[string]reflect.Value

	realSrv  iService
	name     string
	kind     int32
	sAddr    int32
	chanSize int
	msgChan  chan *message
	funcChan chan *tagFunc

	nowNs       int64
	timeWheel   *timeWheel
	delayedRPCs []func()
	allowedRPCs map[string]bool
	logName     string
	loggerPath  string
	logger      logging.ILogger
	load        int64

	closedLock int32
	wg         *sync.WaitGroup

	sMeter IProxy

	httpServer *http2.Server
	httpPort   int
}

func (ss *Service) Start(_ any) {
}

func (ss *Service) Stop(_ *sync.WaitGroup) {
}

func (ss *Service) AfterStop() {
}

func (ss *Service) Entry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) func() {
	f := ss.methodMap[funcName]
	args, err := argGetter(f.Type())
	if err != nil {
		ss.Errorf("doDispatch: get args error: %+v  name = %v", err, funcName)
		return nil
	}
	rArgs := []reflect.Value{reflect.ValueOf(ss.realSrv), reflect.ValueOf(ctx)}
	return func() {
		f.Call(append(rArgs, args...))
	}
}

// Construct 注入构造函数
func (ss *Service) Construct(logger *logging.Logger[Service], httpServer *http2.Server) {
	ss.httpServer = httpServer
	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = "*" + ss.GetName()
		data.ID = fmt.Sprintf("%X", unsafe.Pointer(ss))
		data.Path = ss.loggerPath
	})
}

// GetName 获取服务名称，线程安全
func (ss *Service) GetName() string {
	return ss.name
}

// GetKind 获取服务类型，线程安全
func (ss *Service) GetKind() int32 {
	return ss.kind
}

// GetAddr 获取服务地址，线程安全
func (ss *Service) GetAddr() int32 {
	return ss.sAddr
}

// GetMetric 获取服务指标系统，线程安全
func (ss *Service) GetMetric() IProxy {
	return ss.sMeter
}

// GetLoad 获取负载，非线程安全
func (ss *Service) GetLoad() float64 {
	return float64(ss.load) / 1000
}

// SetChannelSize 设置消息队列大小，注入构造时才可使用
func (ss *Service) SetChannelSize(size int) {
	ss.chanSize = size
}

// SetAllowedRPC 设置允许调用的 RPC 函数，不含 "Rpc" 头，非线程安全
func (ss *Service) SetAllowedRPC(names []string) {
	for _, n := range names {
		ss.allowedRPCs[n] = true
	}
}

// EnableRPC 设置 RPC 可用，此时队列中的 RPC 会开始执行，非线程安全
func (ss *Service) EnableRPC() {
	for _, f := range ss.delayedRPCs {
		f()
	}
	ss.allowedRPCs = nil
	ss.delayedRPCs = nil
}

// EnableHttpRPC 设置 HTTP RPC 可用，此时队列中的 RPC 会开始执行，非线程安全
func (ss *Service) EnableHttpRPC() {
	if ss.httpServer == nil {
		ss.Fatalf("no http server found in host")
		return
	}

	path := ss.getHttpPath()
	ss.httpServer.HandleRequest(path, func(ctx *fasthttp.RequestCtx) {
		handleHttpRpc(ss, ctx)
	})
}

// EnableHttpRPCForward 设置 HTTP RPC 转发可用，此时队列中的 RPC 会开始执行转发，非线程安全
func (ss *Service) EnableHttpRPCForward(srvAddresses []int32) {
	if ss.httpServer == nil {
		ss.Fatalf("no http server found in host")
		return
	}

	if len(srvAddresses) == 0 {
		ss.Fatalf("empty forward services")
		return
	}

	path := ss.getHttpPath()
	ss.httpServer.HandleRequest(path, func(ctx *fasthttp.RequestCtx) {
		srv := nodeGetService(srvAddresses[rand.Intn(len(srvAddresses))])
		if srv == nil {
			ctx.Error("invalid service address", http.StatusInternalServerError)
			return
		}

		handleHttpRpc(srv, ctx)
	})
}

// Fork 将函数放入主线程并在下一帧中执行，线程安全
func (ss *Service) Fork(tag string, f func()) bool {
	tf := &tagFunc{Tag: tag, F: f}
	select {
	case ss.funcChan <- tf:
		return true
	default:
		return false
	}
}

// GetTime 获取当前时间，非线程安全
func (ss *Service) GetTime() time.Time {
	return time.Unix(0, ss.nowNs)
}

// GetSecond 获取当前时间秒，非线程安全
func (ss *Service) GetSecond() int64 {
	return ss.nowNs / 1000_000_000
}

// GetMillisecond 获取当前时间毫秒，非线程安全
func (ss *Service) GetMillisecond() int64 {
	return ss.nowNs / 1000_000
}

// Tick 定时器，延迟 delay 时间并在后续以 interval 时间间隔执行函数 f，间隔时间与函数执行时间无关，非线程安全
func (ss *Service) Tick(interval, delay time.Duration, f func()) ITimer {
	return ss.timeWheel.Tick(interval, delay, f)
}

// TickDelayRandom 定时器，延迟 [0, interval) 随机时间并在后续以 interval 时间间隔执行函数 f，间隔时间与函数执行时间无关，非线程安全
func (ss *Service) TickDelayRandom(interval time.Duration, f func()) ITimer {
	delay := time.Duration(rand.Int63n(int64(interval)))
	return ss.timeWheel.Tick(interval, delay, f)
}

// TickAfter 定时器，延迟 interval 时间后执行函数 f，若传入 f 的函数参数被执行，则延迟 interval 后继续执行函数 f，非线程安全
func (ss *Service) TickAfter(interval time.Duration, f func(func())) ITimer {
	return ss.timeWheel.After(interval, func() {
		f(func() {
			ss.TickAfter(interval, f)
		})
	})
}

// After 定时器，延迟 delay 时间后执行函数 f，非线程安全
func (ss *Service) After(delay time.Duration, f func()) ITimer {
	return ss.timeWheel.After(delay, f)
}

// CreateEmptyProxy 创建空代理，线程安全
func (ss *Service) CreateEmptyProxy() IProxy {
	return dumbProxyObj
}

// CreateProxy 根据服务名以及在配置中的顺序依次在配置的当前节点及其他节点查询并创建代理，线程安全
func (ss *Service) CreateProxy(name string) IProxy {
	kind := gNode.name2Info[name].Kind
	return newServiceProxy(ss, nil, addrAutoSearch, -kind)
}

// CreateProxyByNodeKind 根据节点地址及服务名创建代理，线程安全
func (ss *Service) CreateProxyByNodeKind(nAddr INodeAddr, name string) IProxy {
	kind := gNode.name2Info[name].Kind
	return newServiceProxy(ss, nil, nAddr.(Addr), -kind)
}

// CreateProxyByNodeAddr 根据节点地址及服务地址创建代理，线程安全
func (ss *Service) CreateProxyByNodeAddr(nAddr INodeAddr, sAddr int32) IProxy {
	return newServiceProxy(ss, nil, nAddr.(Addr), sAddr)
}

// Deprecated: 废弃，未来移除
func (ss *Service) CreateProxyByUpdaterKind(nAddrUpdater *NodeAddrUpdater, name string) IProxy {
	kind := gNode.name2Info[name].Kind
	return newServiceProxy(ss, nAddrUpdater, AddrLocal, -kind)
}

// CreateHttpProxy 创建 HTTP 代理，线程安全
func (ss *Service) CreateHttpProxy(url string, name string) IProxy {
	return newHttpProxy(ss, url, name)
}

// RpcStatus 获取服务状态 RPC 的默认实现
func (ss *Service) RpcStatus(ctx IRpcContext) {
	ctx.Return("OK")
}

// RpcReload 重载服务配置 RPC 的默认实现
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

func (ss *Service) getService() *Service {
	return ss
}

func (ss *Service) init(name string, kind int32, sAddr int32, chanSize int, realService iService, methodMap map[string]reflect.Value, httpMethodMap map[string]reflect.Value) {
	ss.name = name
	ss.loggerPath = "Service/" + name
	ss.kind = kind
	ss.sAddr = sAddr
	ss.chanSize = chanSize
	ss.realSrv = realService
	ss.methodMap = methodMap
	ss.httpMethodMap = httpMethodMap
	ss.wg = &sync.WaitGroup{}

	ss.timeWheel = &timeWheel{
		queue:  &timerQueue{},
		timeMs: time.Now().UnixMilli(),
	}
	ss.delayedRPCs = make([]func(), 0, 4)
	ss.allowedRPCs = make(map[string]bool)

	if _, ok := Config.CurNodeMap["Metric"]; ok {
		ss.sMeter = ss.CreateProxy("Metric")
	} else {
		ss.sMeter = ss.CreateEmptyProxy()
	}
}

func (ss *Service) afterInject() {
	ss.msgChan = make(chan *message, ss.chanSize)
	ss.funcChan = make(chan *tagFunc, ss.chanSize)
}

func (ss *Service) start(arg any) {
	ss.wg.Add(1)

	ss.nowNs = time.Now().UnixNano()

	task.Execute(func() {
		isStandalone := Config.CurNodeMap[ss.name]
		if isStandalone {
			ss.Debugf("start...")
		}

		ss.realSrv.Start(arg)

		if isStandalone {
			ss.Infof("start success")
		}

		ss.mainLoop()
	})
}

func (ss *Service) mainLoop() {
	if ss.closed() {
		ss.wg.Done()
		return
	}

	tch := make(chan int64, 1024)
	ticker.Subscribe(int64(ss.sAddr), tch)
	unixNano := <-tch
	ss.nowNs = unixNano

	//meterMs := int64(0)
	//meterForkFuncCallCount := int64(0)
	//meterForkFuncCostMs := int64(0)
	//meterRpcFuncCallCount := int64(0)
	//meterRpcFuncCostMs := int64(0)

L:
	for {
		select {
		case unixNano = <-tch:
			n := len(tch)
			for i := 0; i < n; i++ {
				unixNano = <-tch
			}
			ss.nowNs = unixNano
			ss.timeWheel.Update(ss.GetMillisecond())
		case msg := <-ss.msgChan:
			ss.doDispatch(msg)
		case f := <-ss.funcChan:
			ss.doFunc(f)
		}

		//meterMs += (unixNano - ss.nowNs) / 1000_000
		//if meterMs/1000 > 1 {
		//	m := &metrics.MeterData{
		//		Type:  metrics.MetricGauge_RPCFunctionExecuteCount,
		//		Value: meterRpcFuncCostMs,
		//		Count: meterRpcFuncCallCount,
		//	}
		//	ss.sMeter.Call("Counter", m).Done()
		//
		//	m2 := &metrics.MeterData{
		//		Type:  metrics.MetricGauge_ForkFunctionExecuteCount,
		//		Value: meterForkFuncCostMs,
		//		Count: meterForkFuncCallCount,
		//	}
		//	ss.sMeter.Call("Counter", m2).Done()
		//
		//	meterMs = 0
		//	meterForkFuncCostMs = 0
		//	meterForkFuncCallCount = 0
		//	meterRpcFuncCallCount = 0
		//	meterRpcFuncCostMs = 0
		//}

		//var messages []*message
		//for msg := ss.msgChan.deq(); msg != nil; msg = ss.msgChan.deq() {
		//	messages = append(messages, msg)
		//}
		//
		//start := time.Now().UnixMilli()
		//for _, msg := range messages {
		//	ss.doDispatch(msg)
		//}
		//meterRpcFuncCallCount += int64(len(messages))
		//end := time.Now().UnixMilli()
		//cost := end - start
		//meterRpcFuncCostMs += cost
		//
		//var functions []*tagFunc
		//for f := ss.funcChan.deq(); f != nil; f = ss.funcChan.deq() {
		//	functions = append(functions, f)
		//}
		//
		//start = time.Now().UnixMilli()
		//for _, f := range functions {
		//	ss.doFunc(f)
		//}
		//end = time.Now().UnixMilli()
		//cost = end - start
		//meterForkFuncCostMs += cost
		//meterForkFuncCallCount += int64(len(functions))

		if ss.closed() {
			break L
		}
	}

	ticker.Unsubscribe(int64(ss.sAddr))
	ss.wg.Done()
}

func (ss *Service) stop() {
	if !atomic.CompareAndSwapInt32(&ss.closedLock, 0, 1) {
		return
	}

	// 这里开始退出流程，当前还未处于关闭状态
	isStandalone := Config.CurNodeMap[ss.name]
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

	func() {
		defer func() {
			if err := recover(); err != nil {
				buf := debug.StackInfo()
				ss.Errorf("service 'AfterStop' execute error: %v\n%s", err, buf)
			}
		}()
		ss.realSrv.AfterStop()
	}()

	if isStandalone {
		ss.Infof("stop success")
	}
}

func (ss *Service) closed() bool {
	return atomic.LoadInt32(&ss.closedLock) == 2
}

func (ss *Service) send(msg *message) bool {
	select {
	case ss.msgChan <- msg:
		return true
	default:
		return false
	}
}

func (ss *Service) doFunc(f *tagFunc) {
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
		nAddr: mReq.nAddr,
		src:   ss.GetAddr(),
		dst:   mReq.src,
		sess:  -mReq.sess,
		trace: mReq.trace,
	}
	var cb func()

	cb = func() {
		if mRsp.sess != 0 {
		}
	}

	ctx := newRpcContext(ss, mRsp, mReq, cb)
	if ss.delayedRPCs == nil || ss.allowedRPCs[funcName] {
		ss.entry(ctx, funcName, mReq.getRequestFuncArgs)
	} else {
		ss.delayEntry(ctx, funcName, mReq.getRequestFuncArgs)
	}
}

func (ss *Service) getHttpPath() string {
	res, _ := url.JoinPath(httpRpcPathPrefix, ss.GetName())
	return res
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

func handleHttpRpc(srv *Service, ctx *fasthttp.RequestCtx) {
	var hc httpRequest
	if err := jsoniter.Unmarshal(ctx.Request.Body(), &hc); err != nil {
		ctx.Error("invalid http rpc structure", http.StatusBadRequest)
		return
	}

	f, ok := srv.httpMethodMap[hc.Func]
	if !ok {
		ctx.Error("invalid http rpc name", http.StatusBadRequest)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			srv.Errorf("prepare httpRpc(%v) failed: %v\n%v", hc.Func, r, debug.StackInfo())

			ctx.Error("invalid http rpc data", http.StatusBadRequest)
		}
	}()

	ft := f.Type()
	var args []any
	if ft.NumIn() > 2 {
		args = make([]any, 0, ft.NumIn()-2)
		for i := 2; i < ft.NumIn(); i++ {
			args = append(args, reflect.New(ft.In(i)).Interface())
		}
		if err := jsoniter.Unmarshal(hc.Args, &args); err != nil {
			ctx.Error("invalid http rpc arguments", http.StatusBadRequest)
			return
		}
		args = args[:ft.NumIn()-2]
	}

	var ch chan *httpResponse
	if !hc.Post {
		ch = make(chan *httpResponse, 1)
	}
	httpRpcCtx := newHttpRpcContext(ch)
	rArgs := make([]reflect.Value, 0, ft.NumIn())
	rArgs = append(rArgs, reflect.ValueOf(srv.realSrv))
	rArgs = append(rArgs, reflect.ValueOf(IRpcContext(httpRpcCtx)))
	for _, arg := range args {
		rArgs = append(rArgs, reflect.ValueOf(arg).Elem())
	}

	srv.Fork("Service.httpRpcHandler", func() {
		defer func() {
			if r := recover(); r != nil && ch != nil {
				srv.Errorf("handle httpRpc(%v) failed\n%v", hc.Func, debug.StackInfo())

				ch <- &httpResponse{
					StatusCode: http.StatusInternalServerError,
					Result:     jsoniter.RawMessage(fmt.Sprintf("internal game logic error")),
				}
			}
		}()

		f.Call(rArgs)
	})

	if ch != nil {
		rsp := <-ch

		if rsp.StatusCode != http.StatusOK {
			ctx.Error(string(rsp.Result), rsp.StatusCode)
			return
		}

		res, err := jsoniter.MarshalToString(rsp)
		if err != nil {
			srv.Fork("Service.httpRpcHandler.onError", func() {
				httpRpcCtx.onError(err)
			})
			ctx.Error("http rpc response marshal error", http.StatusInternalServerError)
			return
		}

		ctx.SuccessString("application/json", res)
	} else {
		ctx.SuccessString("text/plain", "")
	}
}
