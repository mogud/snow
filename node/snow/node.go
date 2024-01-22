package snow

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/mogud/snow/core/logging"
	"gitee.com/mogud/snow/core/logging/slog"
	net2 "gitee.com/mogud/snow/core/net"
	"gitee.com/mogud/snow/core/option"
	"gitee.com/mogud/snow/core/syncext"
	"gitee.com/mogud/snow/core/task"
	"gitee.com/mogud/snow/host"
	"gitee.com/mogud/snow/injection"
	"io"
	"math"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type ServiceRegisterInfo struct {
	Kind int32
	Name string
	Type any
}

type NodeExOption struct {
	ServiceRegisterInfos     []*ServiceRegisterInfo
	ClientHandlePreprocessor net2.IPreprocessor
	ServerHandlePreprocessor net2.IPreprocessor
}

var defaultHandlePreprocessor net2.IPreprocessor = &defaultHandlePreprocessorImpl{}

type defaultHandlePreprocessorImpl struct{}

func (d defaultHandlePreprocessorImpl) Process(conn net.Conn) (io.Reader, io.Writer, error) {
	return conn, conn, nil
}

func AddNode(b host.IBuilder, registerFactory func() *NodeExOption) {
	host.AddOptionFactory[*NodeExOption](b, registerFactory)
	host.AddHostedRoutine[*Node](b)
}

// -------------------------------------------------------------------------------

var _ host.IHostedRoutine = (*Node)(nil)

type Node struct {
	sync.Mutex

	logger      logging.ILogger
	nodeBootOpt *NodeBootOption

	nodeScope      injection.IRoutineScope
	chPreprocessor net2.IPreprocessor
	shPreprocessor net2.IPreprocessor
	kind2Info      map[int32]*ServiceRegisterInfo
	name2Info      map[string]*ServiceRegisterInfo
	name2Addr      map[string]int32

	sessID    int32
	saddr     int32
	proto     map[int32]reflect.Type
	methodMap map[int32]map[string]reflect.Value
	services  map[int32]*Service
	handle    map[NodeAddr]*remoteHandle // node address: handle
	listener  net.Listener

	ctx    context.Context
	cancel func()

	closeWait *sync.WaitGroup
}

func (ss *Node) Construct(host host.IHost, logger *logging.Logger[Node], nbOpt *option.Option[*NodeBootOption], nodeOpt *option.Option[*NodeExOption]) {
	opt := nodeOpt.Get()

	ss.nodeScope = host.GetRoutineProvider().GetRootScope()
	ss.chPreprocessor = opt.ClientHandlePreprocessor
	ss.shPreprocessor = opt.ServerHandlePreprocessor
	if ss.chPreprocessor == nil {
		ss.chPreprocessor = defaultHandlePreprocessor
	}
	if ss.shPreprocessor == nil {
		ss.shPreprocessor = defaultHandlePreprocessor
	}

	ss.kind2Info = make(map[int32]*ServiceRegisterInfo)
	ss.name2Info = make(map[string]*ServiceRegisterInfo)
	ss.name2Addr = make(map[string]int32)

	ss.saddr = 0xffff
	ss.proto = make(map[int32]reflect.Type)
	ss.methodMap = make(map[int32]map[string]reflect.Value)
	ss.services = make(map[int32]*Service)
	ss.handle = make(map[NodeAddr]*remoteHandle) // node address: handle

	ss.ctx, ss.cancel = context.WithCancel(context.Background())

	ss.closeWait = &sync.WaitGroup{}

	ss.logger = logger.Get(func(data *logging.LogData) {
		data.Name = "Node"
		data.ID = fmt.Sprintf("%X", unsafe.Pointer(ss))
	})

	ss.nodeBootOpt = nbOpt.Get()
	srvInfos := opt.ServiceRegisterInfos
	for _, info := range srvInfos {
		kind, srv, name := info.Kind, info.Type, info.Name

		// TODO 检查 kind name 是否重复
		// TODO 检查类型是否有效

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

		ss.proto[kind] = st
		ss.methodMap[kind] = methods
	}

	gNode = ss
}

func (ss *Node) Start(ctx context.Context, wg *syncext.TimeoutWaitGroup) {
	InitOptions(ss.nodeBootOpt)

	sn := "Metrics"
	if NodeConfig.CurNodeMap[sn] {
		saddr, err := NewService(sn)
		if err != nil {
			ss.logger.Fatalf("create service(%s) error: %+v", sn, err)
		}

		if !StartService(saddr, nil) {
			ss.logger.Fatalf("start service(%s:%#8x) failed", sn, saddr)
		}
	}

	var saddrs []int32
	for _, sn := range NodeConfig.CurNode {
		if sn == "Metrics" {
			continue
		}

		saddr, err := NewService(sn)
		if err != nil {
			ss.logger.Fatalf("create service(%s) error: %+v", sn, err)
		}
		ss.name2Addr[sn] = saddr
		saddrs = append(saddrs, saddr)
	}

	go func() {
		for _, saddr := range saddrs {
			a := saddr
			if !StartService(a, nil) {
				ss.logger.Fatalf("start service(%s:%#8x) failed", sn, saddr)
			}
		}
	}()

	go ss.nodeStartListen()
}

func (ss *Node) Stop(ctx context.Context, wg *syncext.TimeoutWaitGroup) {
	wg.Add(1)
	ss.cancel()

	for i := len(NodeConfig.CurNode) - 1; i >= 0; i-- {
		sn := NodeConfig.CurNode[i]
		if addr, ok := ss.name2Addr[sn]; ok {
			swg := &sync.WaitGroup{}
			swg.Add(1)
			go func() {
				StopService(addr)
				swg.Done()
			}()
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
	info := gNode.name2Info[name]
	if info == nil {
		return 0, fmt.Errorf("service proto kind(%s) is not registered", name)
	}

	kind := info.Kind

	gNode.Lock()
	defer gNode.Unlock()

	pt := gNode.proto[kind]
	gNode.saddr++

	nsi := reflect.New(pt.Elem()).Interface()
	nss := nsi.(iService)
	ns := nss.getService()
	ns.init(name, kind, gNode.saddr, nss, gNode.methodMap[kind])

	host.Inject(gNode.nodeScope, nsi)

	gNode.services[gNode.saddr] = ns
	gNode.services[-kind] = ns
	return gNode.saddr, nil
}

func StartService(saddr int32, arg interface{}) bool {
	gNode.Lock()

	srv := gNode.services[saddr]
	if srv == nil {
		gNode.Unlock()
		return false
	}
	gNode.Unlock()

	srv.start(arg)

	return true
}

func StopService(saddr int32) bool {
	gNode.Lock()
	srv := gNode.services[saddr]
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

func nodeGetService(saddr int32) *Service {
	gNode.Lock()
	defer gNode.Unlock()

	return gNode.services[saddr]
}

func (ss *Node) nodeStartListen() {
	defer gNode.listener.Close()

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
		naddr, err := NewNodeAddr(ipaddr.IP.String(), ipaddr.Port)
		if err != nil {
			slog.Fatalf("node new remote handle: %+v", err)
		}

		task.Execute(func() {
			h := newServerHandle(ss, naddr, conn)
			if h == nil {
				return
			}
			ss.nodeAddRemoteHandle(naddr, h)

			ss.closeWait.Add(1)
			defer ss.closeWait.Done()
			h.startServer()
			slog.Infof("node remote(%v) disconnected, handle closed", naddr)
		})
	}
}

func (ss *Node) nodeAddRemoteHandle(naddr NodeAddr, h *remoteHandle) {
	gNode.Lock()
	defer gNode.Unlock()

	if old, ok := gNode.handle[naddr]; ok {
		task.Execute(func() {
			old.cancel()
		})
	}
	gNode.handle[naddr] = h
}

func nodeDelRemoteHandle(naddr NodeAddr) {
	gNode.Lock()
	defer gNode.Unlock()

	delete(gNode.handle, naddr)
}

func nodeGetMessageSender(naddr NodeAddr, saddr int32, retry bool, retrySigChan chan<- bool) iMessageSender {
	gNode.Lock()
	defer gNode.Unlock()

	if naddr == 0 {
		return gNode.services[saddr]
	}

	h := gNode.handle[naddr]
	if h != nil {
		return h
	}

	if !retry {
		return nil
	}

	h = newRemoteHandle(gNode, naddr, nil)
	gNode.handle[naddr] = h

	task.Execute(func() {
		slog.Infof("node conntect to %v...", naddr)
		conn, err := net.Dial("tcp4", naddr.String())
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
			slog.Warnf("send identity to server(%v) failed: %v", naddr, err)
			h.safeDelete()
			conn.Close()
			return
		}

		gNode.closeWait.Add(1)
		defer gNode.closeWait.Done()
		slog.Infof("node conntect to %v sucess", naddr)
		h.startClient()
	})
	return h
}
