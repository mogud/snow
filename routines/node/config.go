package node

import (
	"context"
	"fmt"
	"github.com/mogud/snow/core/task"
	"github.com/valyala/fasthttp"
	"net"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type nodeInfo struct {
	Name     string
	Order    int
	NodeAddr Addr
	Host     string
	Port     int
	HttpPort int
	UseHttps bool
	Services []string
}

type nodeConfig struct {
	Nodes           []*nodeInfo
	CurNodeServices []string
	CurNodeMap      map[string]bool
	CurNodeName     string
	CurNodeLocalIP  string
	CurNodeIP       string
	CurNodePort     int
	CurNodeHttpPort int
	CurNodeAddr     Addr
}

var Config = &nodeConfig{
	CurNodeMap: map[string]bool{},
}

func (ss *Node) initOptions() {
	if len(ss.nodeOpt.LocalIP) == 0 {
		panic("node local ip address empty")
	}

	var curHost string
	var curPort int
	var curHttpPort int
	for name, nc := range ss.nodeOpt.Nodes {
		nAddr, err := NewNodeAddr(nc.Host, nc.Port)
		if err != nil {
			ss.logger.Fatalf("invalid node(%s) address: %+v", name, err)
		}

		info := &nodeInfo{
			Name:     name,
			Order:    nc.Order,
			NodeAddr: nAddr,
			Host:     nc.Host,
			Port:     nc.Port,
			HttpPort: nc.HttpPort,
			UseHttps: nc.UseHttps,
		}
		for _, s := range nc.Services {
			info.Services = append(info.Services, s)
		}
		Config.Nodes = append(Config.Nodes, info)

		if name == ss.nodeOpt.BootName {
			curHost = nc.Host
			curPort = nc.Port
			curHttpPort = nc.HttpPort
			Config.CurNodeName = name
			for _, s := range nc.Services {
				if Config.CurNodeMap[s] {
					ss.logger.Fatalf("duplicate service(%s) in node(%s) config", s, name)
				}

				Config.CurNodeMap[s] = true
				Config.CurNodeServices = append(Config.CurNodeServices, s)

			}
		}
	}
	sort.Slice(Config.Nodes, func(i, j int) bool {
		return Config.Nodes[i].Order < Config.Nodes[j].Order
	})

	if len(curHost) == 0 {
		curHost = ss.nodeOpt.LocalIP
	}

	var err error
	ss.tcpListener, err = net.Listen("tcp4", curHost+":"+strconv.Itoa(curPort))
	if err != nil {
		panic(fmt.Sprintf("node tcp listen at port %v failed: %+v", curPort, err))
	}

	listenConfig := &net.ListenConfig{KeepAlive: time.Duration(ss.nodeOpt.HttpKeepAliveSeconds) * time.Second}
	ss.httpListener, err = listenConfig.Listen(context.Background(), "tcp", curHost+":"+strconv.Itoa(curHttpPort))
	if err != nil {
		panic(fmt.Sprintf("node http listen at port %v failed: %+v", curPort, err))
	}

	lAddr := ss.tcpListener.Addr().(*net.TCPAddr)
	Config.CurNodeLocalIP = ss.nodeOpt.LocalIP
	Config.CurNodeIP = lAddr.IP.String()
	Config.CurNodePort = lAddr.Port
	Config.CurNodeHttpPort = ss.httpListener.Addr().(*net.TCPAddr).Port
}

func (ss *Node) postInitOptions() {
	srv := &fasthttp.Server{
		IdleTimeout:  time.Duration(ss.nodeOpt.HttpKeepAliveSeconds) * time.Second,
		ReadTimeout:  time.Duration(ss.nodeOpt.HttpTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(ss.nodeOpt.HttpTimeoutSeconds) * time.Second,
		Handler:      ss.handler,
	}

	ss.handleRequestMethod("/", http.MethodPost, ss.notFound)

	task.Execute(func() {
		if err := srv.Serve(ss.httpListener); err != nil {
			ss.logger.Infof("http listener stopped: %+v", err)
		}
	})

	var err error
	Config.CurNodeAddr, err = NewNodeAddr(ss.nodeOpt.LocalIP, Config.CurNodePort)
	if err != nil {
		panic(fmt.Sprintf("invalid node local ip address: %v", err))
	}

	ss.logger.Infof("tcp listen at %v, http listen at %v, local IP: %v",
		ss.tcpListener.Addr(), ss.httpListener.Addr(), Config.CurNodeLocalIP)
}

func (ss *Node) handler(ctx *fasthttp.RequestCtx) {
	if h, ok := ss.httpHandlers[string(ctx.Path())]; ok {
		h(ctx)
	} else {
		ss.logger.Errorf("invalid http request route: %v", string(ctx.RequestURI()))
	}
}

func (ss *Node) handleRequestMethod(pattern string, method string, handler fasthttp.RequestHandler) {
	ss.httpHandlers[pattern] = func(ctx *fasthttp.RequestCtx) {
		m := string(ctx.Method())
		if method != m {
			if ss.nodeOpt.HttpDebug {
				ss.logger.Warnf("handleRequestMethod: invalid method type '%v' from remote(%v)", m, ctx.RemoteAddr())
			}
			return
		}

		handler(ctx)
	}
}

func (ss *Node) notFound(ctx *fasthttp.RequestCtx) {
	if ss.nodeOpt.HttpDebug {
		ss.logger.Warnf("http request(%s) of url(%s) not found", ctx.RemoteAddr(), ctx.Path())
	}
}
