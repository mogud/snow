package node

import (
	"fmt"
	"net"
	"sort"
	"strconv"
)

type nodeInfo struct {
	Name     string
	Order    int
	NodeAddr Addr
	Services []string
}

type nodeConfig struct {
	Nodes           []*nodeInfo
	CurNodeServices []string
	CurNodeMap      map[string]bool
	CurNodeName     string
	CurNodeIP       string
	CurNodePort     int
	CurNodeHttpPort int
	CurNodeAddr     Addr
}

var Config = &nodeConfig{
	CurNodeMap: map[string]bool{},
}

func (ss *Node) initOptions(opt *Option, httpPort int) {
	if len(opt.LocalIP) == 0 {
		panic("node local ip address empty")
	}

	var curHost string
	var curPort int
	for name, nc := range opt.Nodes {
		nAddr, err := NewNodeAddr(nc.Host, nc.Port)
		if err != nil {
			ss.logger.Fatalf("invalid node(%s) address: %+v", name, err)
		}

		info := &nodeInfo{
			Name:     name,
			Order:    nc.Order,
			NodeAddr: nAddr,
		}
		for _, s := range nc.Services {
			info.Services = append(info.Services, s)
		}
		Config.Nodes = append(Config.Nodes, info)

		if name == opt.BootName {
			curHost = nc.Host
			curPort = nc.Port
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
		curHost = opt.LocalIP
	}

	var err error
	gNode.listener, err = net.Listen("tcp4", curHost+":"+strconv.Itoa(curPort))
	if err != nil {
		panic(fmt.Sprintf("node listen at port %v failed: %+v", curPort, err))
	}
	lAddr := gNode.listener.Addr().(*net.TCPAddr)

	Config.CurNodeIP = opt.LocalIP
	Config.CurNodePort = lAddr.Port
	Config.CurNodeHttpPort = httpPort
	Config.CurNodeAddr, err = NewNodeAddr(opt.LocalIP, lAddr.Port)
	if err != nil {
		panic(fmt.Sprintf("invalid node local ip address: %v", err))
	}

	ss.logger.Infof("node tcp listen at %s, current node address: %v", lAddr.String(), Config.CurNodeAddr.String())
}
