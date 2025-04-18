package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"
	"runtime/trace"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/host"
	"github.com/mogud/snow/core/logging"
	"github.com/mogud/snow/core/option"
	sync2 "github.com/mogud/snow/core/sync"
	"github.com/mogud/snow/core/task"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type fastHttpSrvMux struct {
	mu sync.RWMutex

	handlers map[string]fasthttp.RequestHandler
}

func newFastHttpSrvMux() *fastHttpSrvMux {
	return &fastHttpSrvMux{
		handlers: make(map[string]fasthttp.RequestHandler),
	}
}

func (ss *fastHttpSrvMux) getHandler(pattern string) fasthttp.RequestHandler {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	return ss.handlers[pattern]
}

func (ss *fastHttpSrvMux) setHandler(pattern string, handler fasthttp.RequestHandler) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.handlers[pattern] = handler
}

func (ss *fastHttpSrvMux) Register(pattern string, handler fasthttp.RequestHandler) {
	ss.setHandler(pattern, handler)
}

func (ss *fastHttpSrvMux) Handler(ctx *fasthttp.RequestCtx) {
	h := ss.getHandler(string(ctx.Path()))
	if h != nil {
		h(ctx)
	} else {
		ctx.Error("invalid http request route", http.StatusBadRequest)
	}
}

type IHttpServer interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	GetPort() int
	OnReady(cb func())
}

type Option struct {
	Host             string   `snow:"Host"`
	MinPort          int      `snow:"MinPort"`
	MaxPort          int      `snow:"MaxPort"`
	KeepAliveSeconds int      `snow:"KeepAliveSeconds"`
	TimeoutSeconds   int      `snow:"TimeoutSeconds"`
	WhiteList        []string `snow:"WhiteList"`
	UncheckedPath    []string `snow:"UncheckedPath"`
	Debug            bool     `snow:"Debug"`
}

var _ IHttpServer = (*Server)(nil)
var _ host.IHostedRoutine = (*Server)(nil)

type Server struct {
	opt     *Option
	logger  logging.ILogger
	ctx     context.Context
	cancel  func()
	port    int
	started int32

	srvMux *fastHttpSrvMux
	queue  *container.ThreadSafeQueue[func()]
}

func (ss *Server) Construct(opt *option.Option[*Option], logger *logging.Logger[Server]) {
	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = "HttpServer"
		data.ID = fmt.Sprintf("%X", unsafe.Pointer(ss))
	})
	ss.opt = opt.Get()
	if len(ss.opt.Host) == 0 {
		ss.opt.Host = "0.0.0.0"
	}
	if ss.opt.MinPort == 0 {
		ss.opt.MinPort = 10000
	}
	if ss.opt.MaxPort == 0 {
		ss.opt.MaxPort = 10099
	}
	if ss.opt.KeepAliveSeconds == 0 {
		ss.opt.KeepAliveSeconds = 60
	}
	if ss.opt.TimeoutSeconds == 0 {
		ss.opt.TimeoutSeconds = 5
	}

	ss.ctx, ss.cancel = context.WithCancel(context.Background())

	ss.srvMux = newFastHttpSrvMux()
	ss.queue = container.NewThreadSafeQueue[func()]()
}

func (ss *Server) Start(ctx context.Context, wg *sync2.TimeoutWaitGroup) {
	var listener net.Listener
	var err error
	listenConfig := &net.ListenConfig{KeepAlive: time.Duration(ss.opt.KeepAliveSeconds) * time.Second}
	for ss.port = ss.opt.MinPort; ss.port <= ss.opt.MaxPort; ss.port++ {
		listener, err = listenConfig.Listen(
			ss.ctx,
			"tcp",
			ss.opt.Host+":"+strconv.Itoa(ss.port),
		)
		if err == nil {
			break
		}
	}

	if ss.port > ss.opt.MaxPort {
		ss.logger.Fatalf("http listen failed: %v", err)
		return
	}

	srv := &fasthttp.Server{
		IdleTimeout:  time.Duration(ss.opt.KeepAliveSeconds) * time.Second,
		ReadTimeout:  time.Duration(ss.opt.TimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(ss.opt.TimeoutSeconds) * time.Second,
		Handler:      ss.srvMux.Handler,
	}

	task.Execute(func() {
		ss.logger.Infof("http server listen at %s", listener.Addr())
		if err := srv.Serve(listener); err != nil {
			ss.logger.Errorf("ListenAndServe: %+v", err)
		}
	})

	ss.HandleRequest("/", ss.notFound)

	if ss.opt.Debug {
		ss.HandleRequestMethod("/debug/pprof/", http.MethodGet, fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Index))
		ss.HandleRequestMethod("/debug/pprof/cmdline", http.MethodGet, fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Cmdline))
		ss.HandleRequestMethod("/debug/pprof/profile", http.MethodGet, fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Profile))
		ss.HandleRequestMethod("/debug/pprof/symbol", http.MethodGet, fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Symbol))
		ss.HandleRequestMethod("/debug/pprof/trace", http.MethodGet, fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Trace))
		ss.HandleRequestMethod("/debug/pprof/trace_start", http.MethodGet, ss.traceStart)
		ss.HandleRequestMethod("/debug/pprof/trace_stop", http.MethodGet, ss.traceStop)
		ss.HandleRequestMethod("/gc", http.MethodGet, ss.gc)
	}

	atomic.StoreInt32(&ss.started, 1)

	for {
		if ss.queue.Empty() {
			break
		}
		ss.queue.Deq()()
	}
}

func (ss *Server) Stop(ctx context.Context, wg *sync2.TimeoutWaitGroup) {
	ss.cancel()
}

// AddWhiteListIP 添加白名单 ip，为保证性能，此接口不支持初始化后的调用，不支持多线程
func (ss *Server) AddWhiteListIP(ip string) {
	ss.opt.WhiteList = append(ss.opt.WhiteList, ip)
}

func (ss *Server) GetPort() int {
	return ss.port
}

func (ss *Server) OnReady(callback func()) {
	if atomic.LoadInt32(&ss.started) == 1 {
		callback()
		return
	}

	ss.queue.Enq(callback)

	if atomic.LoadInt32(&ss.started) == 1 {
		for {
			if ss.queue.Empty() {
				break
			}
			ss.queue.Deq()()
		}
	}
}

func (ss *Server) HandleRequestMethod(pattern string, method string, handler fasthttp.RequestHandler) {
	ss.srvMux.Register(pattern, func(ctx *fasthttp.RequestCtx) {
		checkWhiteList := true
		for _, s := range ss.opt.UncheckedPath {
			if s == pattern {
				checkWhiteList = false
				break
			}
		}

		m := string(ctx.Method())
		if method != m {
			if ss.opt.Debug {
				ss.logger.Warnf("HandleFunc: invalid method type '%v' from remote(%v)", m, ctx.RemoteAddr())
			}
			return
		}

		if checkWhiteList {
			ip := ctx.RemoteIP().String()

			if !ss.isInWhiteList(ip) {
				if ss.opt.Debug {
					ss.logger.Warnf("HandleFunc: http request remote(%v) not in white list", ctx.RemoteAddr())
				}
				return
			}
		}

		handler(ctx)
	})
}

func (ss *Server) HandleRequest(pattern string, handler fasthttp.RequestHandler) {
	ss.HandleRequestMethod(pattern, http.MethodPost, handler)
}

// Deprecated: use HandleRequestMethod instead
func (ss *Server) HandleFuncMethod(pattern string, method string, handler func(http.ResponseWriter, *http.Request)) {
	ss.HandleRequestMethod(pattern, method, fasthttpadaptor.NewFastHTTPHandlerFunc(handler))
}

// Deprecated: use HandleRequest instead
func (ss *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	ss.HandleRequest(pattern, fasthttpadaptor.NewFastHTTPHandlerFunc(handler))
}

func (ss *Server) isInWhiteList(ip string) bool {
	for i := 0; i < len(ss.opt.WhiteList); i++ {
		if ss.opt.WhiteList[i] == ip {
			return true
		}
	}
	return false
}

// 手动GC
func (ss *Server) gc(ctx *fasthttp.RequestCtx) {
	debug.FreeOSMemory()
	_, _ = ctx.WriteString("force gc and free os memory executed")
}

// 运行trace
func (ss *Server) traceStart(ctx *fasthttp.RequestCtx) {
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}

	err = trace.Start(f)
	if err != nil {
		panic(err)
	}
	_, _ = ctx.WriteString("Trace Start")
}

// 停止trace
func (ss *Server) traceStop(ctx *fasthttp.RequestCtx) {
	trace.Stop()
	_, _ = ctx.WriteString("Trace Stop")
}

func (ss *Server) notFound(ctx *fasthttp.RequestCtx) {
	if ss.opt.Debug {
		ss.logger.Warnf("http request(%s) of url(%s) not found", ctx.RemoteAddr(), ctx.Path())
	}
}
