package node

import (
	"context"
	"errors"
	"fmt"
	http2 "github.com/mogud/snow/routines/http"
	"io"
	"math"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/host"
	"github.com/mogud/snow/core/injection"
	"github.com/mogud/snow/core/kvs"
	"github.com/mogud/snow/core/logging"
	"github.com/mogud/snow/core/logging/slog"
	net2 "github.com/mogud/snow/core/net"
	"github.com/mogud/snow/core/option"
	sync2 "github.com/mogud/snow/core/sync"
	"github.com/mogud/snow/core/task"
)

type nodeElementOption struct {
	Host     string   `snow:"Host"`     // 节点地址
	Port     int      `snow:"Port"`     // 节点端口
	Order    int      `snow:"Order"`    // 节点排序
	Services []string `snow:"Services"` // 具名服务
}

type Option struct {
	LocalIP  string                        `snow:"LocalIP"`  // 内网 ip，用于判断 RPC 连接是否是本地
	BootName string                        `snow:"BootName"` // 启动节点名
	Nodes    map[string]*nodeElementOption `snow:"Nodes"`    // 当前关注的节点信息
}

type RegisterOption struct {
	ServiceRegisterInfos     []*ServiceRegisterInfo
	ClientHandlePreprocessor net2.IPreprocessor
	ServerHandlePreprocessor net2.IPreprocessor
	PostInitializer          func()
}

type ServiceRegisterInfo struct {
	Kind int32
	Name string
	Type any
}

var defaultHandlePreprocessor net2.IPreprocessor = &defaultHandlePreprocessorImpl{}

type defaultHandlePreprocessorImpl struct{}

func (d defaultHandlePreprocessorImpl) Process(conn net.Conn) (io.Reader, io.Writer, error) {
	return conn, conn, nil
}

func AddNode(b host.IBuilder, registerFactory func() *RegisterOption) {
	host.AddOptionFactory[*RegisterOption](b, registerFactory)
	host.AddHostedRoutine[*Node](b)
}

// -------------------------------------------------------------------------------

var _ host.IHostedRoutine = (*Node)(nil)

type Node struct {
	sync.Mutex

	logger  logging.ILogger
	nodeOpt *Option
	regOpt  *RegisterOption

	nodeScope      injection.IRoutineScope
	chPreprocessor net2.IPreprocessor
	shPreprocessor net2.IPreprocessor
	kind2Info      map[int32]*ServiceRegisterInfo
	name2Info      map[string]*ServiceRegisterInfo
	name2Addr      map[string]int32

	sessID        int32
	sAddr         int32
	proto         map[int32]reflect.Type
	methodMap     map[int32]map[string]reflect.Value
	httpMethodMap map[int32]map[string]reflect.Value
	services      map[int32]*Service
	handle        map[Addr]*remoteHandle // node address: handle
	listener      net.Listener

	ctx    context.Context
	cancel func()

	closeWait *sync.WaitGroup

	httpServer *http2.Server
}

func (ss *Node) Construct(host host.IHost, logger *logging.Logger[Node], nodeOpt *option.Option[*Option], registerOpt *option.Option[*RegisterOption], httpServer *http2.Server) {
	ss.regOpt = registerOpt.Get()

	ss.httpServer = httpServer
	ss.nodeScope = host.GetRoutineProvider().GetRootScope()
	ss.chPreprocessor = ss.regOpt.ClientHandlePreprocessor
	ss.shPreprocessor = ss.regOpt.ServerHandlePreprocessor
	if ss.chPreprocessor == nil {
		ss.chPreprocessor = defaultHandlePreprocessor
	}
	if ss.shPreprocessor == nil {
		ss.shPreprocessor = defaultHandlePreprocessor
	}

	ss.kind2Info = make(map[int32]*ServiceRegisterInfo)
	ss.name2Info = make(map[string]*ServiceRegisterInfo)
	ss.name2Addr = make(map[string]int32)

	ss.sAddr = 0xffff
	ss.proto = make(map[int32]reflect.Type)
	ss.methodMap = make(map[int32]map[string]reflect.Value)
	ss.httpMethodMap = make(map[int32]map[string]reflect.Value)
	ss.services = make(map[int32]*Service)
	ss.handle = make(map[Addr]*remoteHandle) // node address: handle

	ss.ctx, ss.cancel = context.WithCancel(context.Background())

	ss.closeWait = &sync.WaitGroup{}

	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = "Node"
		data.ID = fmt.Sprintf("%X", unsafe.Pointer(ss))
	})

	ss.nodeOpt = nodeOpt.Get()

	if v, ok := kvs.Get[string]("NODE_TO_START"); ok && len(v) > 0 {
		ss.nodeOpt.BootName = v
	}

	if v, ok := kvs.Get[string]("NODE_LISTEN_HOST"); ok && len(v) > 0 {
		for name, nc := range ss.nodeOpt.Nodes {
			if name == ss.nodeOpt.BootName {
				nc.Host = v
				break
			}
		}
	}

	if v, ok := kvs.Get[int]("NODE_LISTEN_PORT"); ok {
		for name, nc := range ss.nodeOpt.Nodes {
			if name == ss.nodeOpt.BootName {
				nc.Port = v
				break
			}
		}
	}

	for name, nc := range ss.nodeOpt.Nodes {
		if name == ss.nodeOpt.BootName {
			ss.logger.Infof("read => name: %v host: %v port: %v", ss.nodeOpt.BootName, nc.Host, nc.Port)
			break
		}
	}

	srvInfos := ss.regOpt.ServiceRegisterInfos
	for _, info := range srvInfos {
		kind, srv, name := info.Kind, info.Type, info.Name

		// TODO by mogu: 检查 kind name 是否重复
		// TODO by mogu: 检查类型是否有效

		ss.kind2Info[kind] = info
		ss.name2Info[name] = info

		if _, ok := ss.proto[kind]; ok {
			ss.logger.Errorf("service proto kind(%d) already registered", kind)
			return
		}
		st := reflect.TypeOf(srv)
		if st == nil {
			continue
		}

		methods := make(map[string]reflect.Value)
		for i := 0; i < st.NumMethod(); i++ {
			m := st.Method(i)
			if strings.HasPrefix(m.Name, "Rpc") {
				methods[strings.TrimPrefix(m.Name, "Rpc")] = m.Func
			}
		}

		httpMethods := make(map[string]reflect.Value)
		for i := 0; i < st.NumMethod(); i++ {
			m := st.Method(i)
			if strings.HasPrefix(m.Name, "HttpRpc") {
				httpMethods[strings.TrimPrefix(m.Name, "HttpRpc")] = m.Func
			}
		}

		ss.proto[kind] = st
		ss.methodMap[kind] = methods
		ss.httpMethodMap[kind] = httpMethods
	}

	gNode = ss
}

func (ss *Node) Start(ctx context.Context, wg *sync2.TimeoutWaitGroup) {
	wg.Add(1)

	ss.httpServer.AddWhiteListIP("127.0.0.1")
	ss.httpServer.AddWhiteListIP(ss.nodeOpt.LocalIP)
	ss.httpServer.OnReady(func() {
		task.Execute(func() {
			httpPort := ss.httpServer.GetPort()

			ss.initOptions(ss.nodeOpt, httpPort)

			if ss.regOpt.PostInitializer != nil {
				ss.regOpt.PostInitializer()
			}

			var services []*container.Pair[string, int32]
			for _, sn := range Config.CurNodeServices {
				sAddr, err := newService(sn, 204800)
				if err != nil {
					ss.logger.Fatalf("create service(%s) error: %+v", sn, err)
				}
				ss.name2Addr[sn] = sAddr
				services = append(services, &container.Pair[string, int32]{
					First:  sn,
					Second: sAddr,
				})
			}

			task.Execute(func() {
				for _, service := range services {
					sn, sAddr := service.First, service.Second
					if !StartService(sAddr, nil) {
						ss.logger.Fatalf("start service(%s:%#8x) failed", sn, sAddr)
					}
				}

				wg.Done()

				ss.logger.Infof("%v services started", len(services))
			})

			task.Execute(ss.nodeStartListen)
		})
	})
}

func (ss *Node) Stop(ctx context.Context, wg *sync2.TimeoutWaitGroup) {
	wg.Add(1)
	ss.cancel()

	for i := len(Config.CurNodeServices) - 1; i >= 0; i-- {
		sn := Config.CurNodeServices[i]
		if addr, ok := ss.name2Addr[sn]; ok {
			swg := &sync.WaitGroup{}
			swg.Add(1)
			task.Execute(func() {
				StopService(addr)
				swg.Done()
			})
			swg.Wait()
		}
	}

	_ = gNode.listener.Close()

	gNode.Lock()
	for _, h := range gNode.handle {
		h.cancel()
	}
	gNode.Unlock()

	gNode.closeWait.Wait()

	wg.Done()
}

var gNode *Node

func NewService(name string) (int32, error) {
	return newService(name, 8192)
}

func newService(name string, chanSize int) (int32, error) {
	info := gNode.name2Info[name]
	if info == nil {
		return 0, fmt.Errorf("service proto kind(%s) is not registered", name)
	}

	kind := info.Kind

	gNode.Lock()
	defer gNode.Unlock()

	pt := gNode.proto[kind]
	gNode.sAddr++

	nsi := reflect.New(pt.Elem()).Interface()
	nss := nsi.(iService)
	ns := nss.getService()
	ns.init(name, kind, gNode.sAddr, chanSize, nss, gNode.methodMap[kind], gNode.httpMethodMap[kind])

	host.Inject(gNode.nodeScope, nsi)

	ns.afterInject()

	gNode.services[gNode.sAddr] = ns
	gNode.services[-kind] = ns
	return gNode.sAddr, nil
}

// StartService 快速启动一个服务，保证异步调用到 Service 的 Start，由用户保证完整、正确启动
func StartService(sAddr int32, arg any) bool {
	gNode.Lock()
	defer gNode.Unlock()

	srv := gNode.services[sAddr]
	if srv == nil {
		return false
	}

	srv.start(arg)

	return true
}

// StopService 关闭一个服务，阻塞执行
func StopService(sAddr int32) bool {
	gNode.Lock()
	srv := gNode.services[sAddr]
	if srv == nil {
		gNode.Unlock()
		return false
	}

	delete(gNode.services, srv.GetAddr())
	gNode.Unlock()

	srv.stop()

	return true
}

func nodeGenSessionID() int32 {
	atomic.CompareAndSwapInt32(&gNode.sessID, math.MaxInt32, 0) // 保证+1之后不会出现负数。否则rpc会一直有问题
	return atomic.AddInt32(&gNode.sessID, 1)
}

func nodeGetService(sAddr int32) *Service {
	gNode.Lock()
	defer gNode.Unlock()

	return gNode.services[sAddr]
}

func (ss *Node) nodeStartListen() {
	defer func() {
		_ = gNode.listener.Close()
	}()
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		conn, err := ss.listener.Accept()
		if err != nil {
			select {
			case <-ss.ctx.Done():
				return
			default:
			}

			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if maxDelay := 1 * time.Second; tempDelay > maxDelay {
					tempDelay = maxDelay
				}
				slog.Errorf("accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			slog.Fatalf("accept error: %v", err)
		}
		tempDelay = 0

		ipaddr := conn.RemoteAddr().(*net.TCPAddr)
		nAddr, err := NewNodeAddr(ipaddr.IP.String(), ipaddr.Port)
		if err != nil {
			slog.Fatalf("node new remote handle: %+v", err)
		}

		task.Execute(func() {
			h := newServerHandle(ss, nAddr, conn)
			if h == nil {
				return
			}
			ss.nodeAddRemoteHandle(nAddr, h)

			ss.closeWait.Add(1)
			defer ss.closeWait.Done()
			h.startServer()
			slog.Infof("node remote(%v) disconnected, handle closed", nAddr)
		})
	}
}

func (ss *Node) nodeAddRemoteHandle(nAddr Addr, h *remoteHandle) {
	gNode.Lock()
	defer gNode.Unlock()

	if old, ok := gNode.handle[nAddr]; ok {
		task.Execute(func() {
			old.cancel()
		})
	}
	gNode.handle[nAddr] = h
}

func nodeDelRemoteHandle(nAddr Addr) {
	gNode.Lock()
	defer gNode.Unlock()

	delete(gNode.handle, nAddr)
}

func nodeGetMessageSender(nAddr Addr, sAddr int32, retry bool, retrySigChan chan<- bool) iMessageSender {
	gNode.Lock()
	defer gNode.Unlock()

	if nAddr == 0 {
		return gNode.services[sAddr]
	}

	h := gNode.handle[nAddr]
	if h != nil {
		return h
	}

	if !retry {
		return nil
	}

	h = newRemoteHandle(gNode, nAddr, nil)
	gNode.handle[nAddr] = h

	task.Execute(func() {
		slog.Infof("node conntect to %v...", nAddr)
		conn, err := net.Dial("tcp4", nAddr.String())
		if err != nil {
			slog.Warnf("node get remote handle failed: %+v", err)
			h.safeDelete()
			if retrySigChan != nil {
				retrySigChan <- true
			}
			return
		}

		h.conn = conn

		if h.r, h.w, err = gNode.chPreprocessor.Process(conn); err != nil {
			slog.Warnf("send identity to server(%v) failed: %v", nAddr, err)
			h.safeDelete()
			_ = conn.Close()
			return
		}

		gNode.closeWait.Add(1)
		defer gNode.closeWait.Done()
		slog.Infof("node conntect to %v sucess", nAddr)
		h.startClient()
	})
	return h
}
