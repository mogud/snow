package http

import (
	"context"
	"fmt"
	"gitee.com/mogud/snow/core/container"
	"gitee.com/mogud/snow/core/host"
	"gitee.com/mogud/snow/core/logging"
	"gitee.com/mogud/snow/core/logging/slog"
	"gitee.com/mogud/snow/core/option"
	"gitee.com/mogud/snow/core/sync"
	"gitee.com/mogud/snow/core/task"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"
	"runtime/trace"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"
)

type IHttpServer interface {
	SafeHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	GetPort() int
	OnReady(cb func())
}

type Option struct {
	Host             string   `koanf:"Host"`
	MinPort          int      `koanf:"MinPort"`
	MaxPort          int      `koanf:"MaxPort"`
	KeepAliveSeconds int      `koanf:"KeepAliveSeconds"`
	TimeoutSeconds   int      `koanf:"TimeoutSeconds"`
	WhiteList        []string `koanf:"WhiteList"`
	Debug            bool     `koanf:"Debug"`
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

	srvMux *http.ServeMux
	queue  container.ThreadSafeQueue[func()]
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

	ss.srvMux = http.NewServeMux()
}

func (ss *Server) Start(ctx context.Context, wg *sync.TimeoutWaitGroup) {
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

	srv := &http.Server{
		IdleTimeout:       time.Duration(ss.opt.KeepAliveSeconds) * time.Second,
		ReadTimeout:       time.Duration(ss.opt.TimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(ss.opt.TimeoutSeconds) * time.Second,
		ReadHeaderTimeout: time.Duration(ss.opt.TimeoutSeconds) * time.Second,
		Handler:           ss.srvMux,
	}

	wg.Add(2)
	task.Execute(func() {
		slog.Infof("http server listen at %s", listener.Addr())
		if err := srv.Serve(listener); err != nil {
			slog.Errorf("ListenAndServe: %+v", err)
		}
		wg.Done()
	})

	ss.HandleFunc("/", ss.notFound)

	if ss.opt.Debug {
		ss.SafeHandleFunc("/debug/pprof/", pprof.Index)
		ss.SafeHandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		ss.SafeHandleFunc("/debug/pprof/profile", pprof.Profile)
		ss.SafeHandleFunc("/debug/pprof/symbol", pprof.Symbol)
		ss.SafeHandleFunc("/debug/pprof/trace", pprof.Trace)
		ss.SafeHandleFunc("/debug/pprof/trace_start", ss.traceStart)
		ss.SafeHandleFunc("/debug/pprof/trace_stop", ss.traceStop)
		ss.SafeHandleFunc("/gc", ss.gc)
	}

	atomic.StoreInt32(&ss.started, 1)

	for {
		if ss.queue.Empty() {
			break
		}
		ss.queue.Deq()()
	}

	wg.Done()
}

func (ss *Server) SafeHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	ss.srvMux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		ipaddr, _ := net.ResolveTCPAddr("tcp", req.RemoteAddr)
		ip := ipaddr.IP.String()

		if !ss.isInWhiteList(ip) {
			if ss.opt.Debug {
				slog.Warnf("SafeHandleFunc: http request remote(%v) not in whitelist", req.RemoteAddr)
			}
			return
		}

		handler(w, req)
	})
}

func (ss *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	ss.srvMux.HandleFunc(pattern, handler)
}

func (ss *Server) Stop(ctx context.Context, wg *sync.TimeoutWaitGroup) {
	ss.cancel()
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
func (ss *Server) gc(w http.ResponseWriter, r *http.Request) {
	debug.FreeOSMemory()
	_, _ = w.Write([]byte("force gc and free os memory executed"))
}

// 运行trace
func (ss *Server) traceStart(w http.ResponseWriter, r *http.Request) {
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}

	err = trace.Start(f)
	if err != nil {
		panic(err)
	}
	_, _ = w.Write([]byte("Trace Start"))
}

// 停止trace
func (ss *Server) traceStop(w http.ResponseWriter, r *http.Request) {
	trace.Stop()
	_, _ = w.Write([]byte("Trace Stop"))
}

func (ss *Server) notFound(w http.ResponseWriter, r *http.Request) {
	if ss.opt.Debug {
		slog.Warnf("http request(%s) of url(%s) not found", r.RemoteAddr, r.URL.Path)
	}
}
