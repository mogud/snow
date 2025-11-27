package node

import (
	"crypto/tls"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/mogud/snow/core/debug"
	"github.com/mogud/snow/core/logging"
	"github.com/mogud/snow/core/task"
	"github.com/mogud/snow/core/ticker"
	"github.com/valyala/fasthttp"
	"math/rand/v2"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var _ iService = (*Service)(nil)
var _ iMessageSender = (*Service)(nil)
var _ ticker.PoolItem = (*Service)(nil)

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

type tagFunc struct {
	Tag string
	F   func()
}

type Service struct {
	node *Node

	methodMap     map[string]reflect.Value
	httpMethodMap map[string]reflect.Value

	realSrv iService
	name    string
	kind    int32
	sAddr   int32

	funcBufferLock sync.Mutex
	funcBuffer     []*tagFunc
	msgBufferLock  sync.Mutex
	msgBuffer      []*message

	nowNs           int64
	tw              *timeWheel
	delayedRpc      []func()
	allowedRpc      map[string]bool
	delayedHttpRpc  []chan struct{}
	httpRpcLock     sync.Mutex
	httpForwardAddr []int32

	loggerPath string
	logger     logging.ILogger

	closedLock int32
	wg         *sync.WaitGroup

	metricNameFuncPrefix string
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
		ss.Errorf("doDispatch: get args error: %+v name = %v", err, funcName)
		return nil
	}
	fArgs := append([]reflect.Value{reflect.ValueOf(ss.realSrv), reflect.ValueOf(ctx)}, args...)
	return func() {
		f.Call(fArgs)

		for _, arg := range fArgs {
			if arg.CanAddr() {
				arg.SetZero()
			}
		}
		fArgs = nil
	}
}

// Construct 注入构造函数
func (ss *Service) Construct(logger *logging.Logger[Service]) {
	selfPtr := unsafe.Pointer(ss)
	path := ss.loggerPath
	name := ss.name
	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = "*" + name
		data.ID = fmt.Sprintf("%X", selfPtr)
		data.Path = path
	})
}

func (ss *Service) Paused() bool {
	return false
}

func (ss *Service) Closed() bool {
	return ss.closed()
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

// SetAllowedRPC 设置允许调用的 RPC 函数，不含 "Rpc" 头，非线程安全
func (ss *Service) SetAllowedRPC(names []string) {
	for _, n := range names {
		ss.allowedRpc[n] = true
	}
}

// EnableRpc 设置 RPC 可用，此时队列中的 RPC 会开始执行，非线程安全
func (ss *Service) EnableRpc() {
	for _, f := range ss.delayedRpc {
		f()
	}
	ss.allowedRpc = nil
	ss.delayedRpc = nil
}

// EnableHttpRpc 设置 HTTP RPC 可用，此时队列中的 RPC 会开始执行，非线程安全
func (ss *Service) EnableHttpRpc() {
	var delayedRpc []chan struct{}
	ss.httpRpcLock.Lock()
	delayedRpc = ss.delayedHttpRpc
	ss.delayedHttpRpc = nil
	ss.httpRpcLock.Unlock()

	for _, ch := range delayedRpc {
		close(ch)
	}
}

// EnableHttpRPCForward 设置 HTTP RPC 转发可用，此时队列中的 RPC 会开始执行转发，非线程安全
func (ss *Service) EnableHttpRPCForward(srvAddresses []int32) {
	if len(srvAddresses) == 0 {
		ss.Fatalf("empty forward services")
		return
	}

	ss.httpForwardAddr = srvAddresses

	ss.EnableHttpRpc()
}

// Fork 将函数放入主线程并在下一帧中执行，线程安全
func (ss *Service) Fork(tag string, f func()) bool {
	return ss.fork(tag, f)
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
func (ss *Service) Tick(interval, delay time.Duration, f func()) ITimeWheelHandle {
	return ss.tw.createTickItem(interval, delay, f)
}

// TickDelayRandom 定时器，延迟 [0, interval) 随机时间并在后续以 interval 时间间隔执行函数 f，间隔时间与函数执行时间无关，非线程安全
func (ss *Service) TickDelayRandom(interval time.Duration, f func()) ITimeWheelHandle {
	delay := time.Duration(rand.Int64N(int64(interval)))
	return ss.tw.createTickItem(interval, delay, f)
}

// TickAfter 定时器，延迟 interval 时间后执行函数 f，若传入 f 的函数参数被执行，则延迟 interval 后继续执行函数 f，非线程安全
func (ss *Service) TickAfter(interval time.Duration, f func(func())) ITimeWheelHandle {
	return ss.tw.createAfterItem(interval, func() {
		f(func() {
			ss.TickAfter(interval, f)
		})
	})
}

// After 定时器，延迟 delay 时间后执行函数 f，非线程安全
func (ss *Service) After(delay time.Duration, f func()) ITimeWheelHandle {
	return ss.tw.createAfterItem(delay, f)
}

// CreateEmptyProxy 创建空代理，线程安全
func (ss *Service) CreateEmptyProxy() IProxy {
	return &serviceProxy{}
}

// CreateProxy 根据服务名自动创建代理，线程安全
func (ss *Service) CreateProxy(name string) IProxy {
	return ss.createProxy(nil, AddrInvalid, 0, name)
}

// CreateProxyByNodeKind 根据节点地址及服务名创建代理，线程安全
func (ss *Service) CreateProxyByNodeKind(nAddr INodeAddr, name string) IProxy {
	return ss.createProxy(nil, nAddr.(Addr), 0, name)
}

// CreateProxyByNodeAddr 根据节点地址及服务地址创建代理，线程安全
func (ss *Service) CreateProxyByNodeAddr(nAddr INodeAddr, sAddr int32) IProxy {
	return ss.createProxy(nil, nAddr.(Addr), sAddr, "")
}

// CreateProxyByUpdaterKind 根据节点地址更新器及服务名创建代理，线程安全
func (ss *Service) CreateProxyByUpdaterKind(nAddrUpdater *AddrUpdater, name string) IProxy {
	return ss.createProxy(nAddrUpdater, AddrInvalid, 0, name)
}

func (ss *Service) CreateHttpProxy(httpUrl, name string) IProxy {
	// TODO by mogu: Golang HTTP2 有 bug，会导致超时访问，使用 HTTP1 可以绕过
	tr := &http.Transport{}
	tr.TLSClientConfig = &tls.Config{
		NextProtos: []string{"h1"},
	}

	res, _ := url.JoinPath(httpUrl, httpRpcPathPrefix, name)
	return &httpProxy{
		srv: ss,
		url: res,
		httpClient: &http.Client{
			Timeout:   time.Second * 8,
			Transport: tr,
		},
	}
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

func (ss *Service) init(node *Node, name string, kind int32, sAddr int32,
	realService iService, methodMap map[string]reflect.Value, httpMethodMap map[string]reflect.Value) {

	ss.node = node
	ss.name = name
	ss.loggerPath = "Service/" + name
	ss.kind = kind
	ss.sAddr = sAddr
	ss.realSrv = realService
	ss.methodMap = methodMap
	ss.httpMethodMap = httpMethodMap
	ss.wg = &sync.WaitGroup{}

	ss.tw = newTimeWheel(time.Now(), 10*time.Millisecond)
	ss.delayedRpc = make([]func(), 0, 4)
	ss.allowedRpc = make(map[string]bool)
	ss.delayedHttpRpc = make([]chan struct{}, 0, 4)

	ss.metricNameFuncPrefix = "[ServiceFunc] " + ss.name + "::"
}

func (ss *Service) afterInject() {
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

		defer ss.wg.Done()

		if ss.closed() {
			return
		}

		// 这里即等待 onTickStop 完成
		ss.wg.Add(1)
		ss.nowNs = time.Now().UnixNano()
		ss.node.serviceTickerPool.Add(ss)
	})
}

func (ss *Service) onTick() {
	now := time.Now()
	ss.nowNs = now.UnixNano()
	ss.tw.update(now)

	ss.funcBufferLock.Lock()
	funcList := ss.funcBuffer
	ss.funcBuffer = nil
	ss.funcBufferLock.Unlock()

	mc := ss.node.regOpt.MetricCollector
	for _, f := range funcList {
		if mc != nil {
			start := time.Now().UnixNano()
			ss.doFunc(f)
			dur := time.Now().UnixNano() - start

			mc.Histogram(ss.metricNameFuncPrefix+f.Tag, float64(dur))
		} else {
			ss.doFunc(f)
		}
	}

	ss.msgBufferLock.Lock()
	msgList := ss.msgBuffer
	ss.msgBuffer = nil
	ss.msgBufferLock.Unlock()
	for _, msg := range msgList {
		ss.doDispatch(msg)
	}
}

func (ss *Service) onTickStop() {
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
		func() {
			defer func() {
				if err := recover(); err != nil {
					buf := debug.StackInfo()
					ss.Errorf("service 'Stop' execute error: %v\n%s", err, buf)
				}
			}()
			ss.realSrv.Stop(wg)
		}()

		wg.Done()
	})

	wg.Wait()

	atomic.StoreInt32(&ss.closedLock, 2)

	// 等待初始化或 ticker 结束，也即等待 service 的主线程结束
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

	// 清空 buffer
	ss.send(nil)
	ss.fork("", nil)

	ss.node = nil

	ss.methodMap = nil
	ss.httpMethodMap = nil

	ss.realSrv = nil
	ss.tw = nil
	ss.delayedRpc = nil
	ss.allowedRpc = nil
	ss.delayedHttpRpc = nil
	ss.httpForwardAddr = nil

	ss.wg = nil
}

func (ss *Service) closed() bool {
	return atomic.LoadInt32(&ss.closedLock) == 2
}

func (ss *Service) fork(tag string, f func()) bool {
	if ss.closed() {
		ss.funcBufferLock.Lock()
		defer ss.funcBufferLock.Unlock()

		if ss.funcBuffer != nil {
			for _, tagF := range ss.funcBuffer {
				tagF.F = nil
			}
			ss.funcBuffer = nil
		}

		return false
	}

	ss.funcBufferLock.Lock()
	defer ss.funcBufferLock.Unlock()
	ss.funcBuffer = append(ss.funcBuffer, &tagFunc{Tag: tag, F: f})
	return true
}

func (ss *Service) send(msg *message) bool {
	if ss.closed() {
		ss.msgBufferLock.Lock()
		defer ss.msgBufferLock.Unlock()

		if ss.msgBuffer != nil {
			for _, m := range ss.msgBuffer {
				m.clear()
			}
			ss.msgBuffer = nil
		}

		return false
	}

	ss.msgBufferLock.Lock()
	defer ss.msgBufferLock.Unlock()
	ss.msgBuffer = append(ss.msgBuffer, msg)
	return true
}

func (ss *Service) doFunc(f *tagFunc) {
	defer func() {
		if err := recover(); err != nil {
			buf := debug.StackInfo()
			ss.Errorf("service execute function(%v) error: %v\n%s", f.Tag, err, buf)
		}
		f.F = nil
	}()
	f.F()
}

func (ss *Service) doDispatch(mReq *message) {
	var funcName string
	defer func() {
		mReq.clear()

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
		src:   ss.sAddr,
		dst:   mReq.src,
		sess:  -mReq.sess,
		trace: mReq.trace,
	}

	if mc := ss.node.regOpt.MetricCollector; mc != nil {
		mc.Counter("[ServiceRpc] "+ss.name, 1)

		start := time.Now().UnixNano()
		var cb func()
		isRequest := mReq.sess != 0
		if isRequest {
			cb = func() {
				// request
				dur := time.Now().UnixNano() - start
				mc.Histogram("[ServiceRequest] "+ss.name+"::"+funcName, float64(dur))
			}
		}

		ctx := newRpcContext(ss, mRsp, mReq.sess, mReq.src, mReq.nAddr, mReq.cb, cb)
		if ss.delayedRpc == nil || ss.allowedRpc[funcName] {
			ss.entry(ctx, funcName, mReq.getRequestFuncArgs)
		} else {
			ss.delayEntry(ctx, funcName, mReq.getRequestFuncArgs)
		}

		if !isRequest {
			dur := time.Now().UnixNano() - start
			mc.Histogram("[ServicePost] "+ss.name+"::"+funcName, float64(dur))
		}
	} else {
		ctx := newRpcContext(ss, mRsp, mReq.sess, mReq.src, mReq.nAddr, mReq.cb, nil)
		if ss.delayedRpc == nil || ss.allowedRpc[funcName] {
			ss.entry(ctx, funcName, mReq.getRequestFuncArgs)
		} else {
			ss.delayEntry(ctx, funcName, mReq.getRequestFuncArgs)
		}
	}
}

func (ss *Service) delayEntry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) {
	if f := ss.realSrv.Entry(ctx, funcName, argGetter); f != nil {
		ss.delayedRpc = append(ss.delayedRpc, f)
	}
}

func (ss *Service) entry(ctx IRpcContext, funcName string, argGetter func(ft reflect.Type) ([]reflect.Value, error)) {
	if f := ss.realSrv.Entry(ctx, funcName, argGetter); f != nil {
		f()
	}
}

func (ss *Service) handleHttpRpc(ctx *fasthttp.RequestCtx) {
	ss.httpRpcLock.Lock()
	if ss.delayedHttpRpc == nil {
		ss.httpRpcLock.Unlock()
		ss.httpEntry(ctx)
		return
	}

	ch := make(chan struct{})
	ss.delayedHttpRpc = append(ss.delayedHttpRpc, ch)
	ss.httpRpcLock.Unlock()

	select {
	case <-ch:
	case <-time.After(30 * time.Second):
		res, err := jsoniter.MarshalToString(&httpResponse{
			StatusCode: http.StatusRequestTimeout,
			Result:     jsoniter.RawMessage("no response from remote service, is 'EnableHttpRpc' called?"),
		})
		if err != nil {
			ctx.Error("http rpc response marshal error", http.StatusInternalServerError)
			return
		}

		ctx.SuccessString("application/json", res)
		return
	}
	ss.httpEntry(ctx)
}

func (ss *Service) httpEntry(ctx *fasthttp.RequestCtx) {
	if len(ss.httpForwardAddr) == 0 {
		processHttpRpc(ss, ctx)
	} else {
		srv := nodeGetService(ss.httpForwardAddr[rand.IntN(len(ss.httpForwardAddr))])
		if srv == nil {
			ctx.Error("invalid service address", http.StatusInternalServerError)
			return
		}

		processHttpRpc(srv, ctx)
	}
}

func (ss *Service) createProxy(updater *AddrUpdater, nAddr Addr, sAddr int32, name string) IProxy {
	if nAddr.IsLocalhost() || nAddr == Config.CurNodeAddr {
		nAddr = AddrLocal
	}

	var urlBase string
	if len(name) > 0 {
		regInfo, ok := ss.node.name2Info[name]
		if !ok {
			ss.Errorf("[createProxy] invalid service name %v", name)
			return nil
		}
		sAddr = -regInfo.Kind

		if updater == nil {
			if nAddr == AddrLocal {
				if !Config.CurNodeMap[name] {
					ss.Errorf("[createProxy] cannot found local service name %v", name)
					return nil
				}
			} else if nAddr == AddrInvalid && Config.CurNodeMap[name] {
				// 自动查找且本地存在需要的服务
				nAddr = AddrLocal
			} else if nAddr == AddrInvalid || nAddr == AddrRemote {
			loop:
				for _, ni := range Config.Nodes {
					if ni.Name == Config.CurNodeName {
						continue
					}

					for _, n := range ni.Services {
						if n == name && len(ni.Host) > 0 {
							// 远端节点的服务，优先使用 Http
							if ni.HttpPort > 0 {
								protocol := "http"
								if ni.UseHttps {
									protocol = "https"
								}
								urlBase = fmt.Sprintf("%v://%s:%v", protocol, ni.Host, ni.HttpPort)
								break loop
							} else if ni.Port > 0 {
								nAddr = ni.NodeAddr
								break loop
							}
						}
					}
				}
			}
		}
	}

	if len(urlBase) > 0 {
		// TODO by mogu: Golang HTTP2 有 bug，会导致超时访问，使用 HTTP1 可以绕过
		tr := &http.Transport{}
		tr.TLSClientConfig = &tls.Config{
			NextProtos: []string{"h1"},
		}

		res, _ := url.JoinPath(urlBase, httpRpcPathPrefix, name)
		return &httpProxy{
			srv: ss,
			url: res,
			httpClient: &http.Client{
				Timeout:   time.Second * 8,
				Transport: tr,
			},
		}
	}

	if nAddr == AddrRemote || sAddr == 0 || (nAddr == AddrInvalid && updater == nil) {
		ss.Errorf("[createProxy] invalid arguments nAddr: %v sAddr: %v name: %v", nAddr, sAddr, name)
		return nil
	}

	return &serviceProxy{
		srv:          ss,
		nAddr:        nAddr,
		nAddrUpdater: updater,
		sAddr:        sAddr,
	}
}

func processHttpRpc(srv *Service, ctx *fasthttp.RequestCtx) {
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

		for _, arg := range rArgs {
			if arg.CanAddr() {
				arg.SetZero()
			}
		}
		rArgs = nil
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
