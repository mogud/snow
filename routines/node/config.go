package node

import (
	"github.com/mogud/snow/core/logging/slog"
	"net"
	"sort"
)

type nodeElementOption struct {
	Host     string   `snow:"host"`     // 节点地址
	Port     int      `snow:"port"`     // 节点端口
	Order    int      `snow:"order"`    // 节点排序
	Services []string `snow:"services"` // 具名服务
}

type NodeBootOption struct {
	BootName string                        `snow:"boot_name"` // 启动节点名
	Nodes    map[string]*nodeElementOption `snow:"nodes"`     // 所有节点数据
}

type nodeInfo struct {
	Name     string
	Order    int
	NodeAddr NodeAddr
	Services []string
}

type NodeConfigStruct struct {
	Nodes       []*nodeInfo
	CurNode     []string
	CurNodeMap  map[string]bool
	CurNodeAddr NodeAddr
	CurNodePort int
}

var NodeConfig = &NodeConfigStruct{
	CurNodeMap: map[string]bool{},
}

func InitOptions(opt *NodeBootOption) {
	for name, nc := range opt.Nodes {
		naddr, err := NewNodeAddr(nc.Host, nc.Port)
		if err != nil {
			slog.Fatalf("invalid node(%s) address: %+v", name, err)
		}

		cfg := &nodeInfo{
			Name:     name,
			Order:    nc.Order,
			NodeAddr: naddr,
		}
		for _, s := range nc.Services {
			cfg.Services = append(cfg.Services, s)
		}
		NodeConfig.Nodes = append(NodeConfig.Nodes, cfg)

		if name == opt.BootName {
			NodeConfig.CurNodeAddr = naddr
			for _, s := range nc.Services {
				if NodeConfig.CurNodeMap[s] {
					slog.Fatalf("duplicate service(%s) in node(%s) config", s, name)
				}

				NodeConfig.CurNodeMap[s] = true
				NodeConfig.CurNode = append(NodeConfig.CurNode, s)
			}
		}
	}
	sort.Slice(NodeConfig.Nodes, func(i, j int) bool {
		return NodeConfig.Nodes[i].Order < NodeConfig.Nodes[j].Order
	})

	addr := NodeConfig.CurNodeAddr.String()
	var err error
	gNode.listener, err = net.Listen("tcp4", addr)
	if err != nil {
		slog.Fatalf("node listen: %+v", err)
	}
	laddr := gNode.listener.Addr().(*net.TCPAddr)
	NodeConfig.CurNodePort = laddr.Port
	NodeConfig.CurNodeAddr, _ = NewNodeAddr(laddr.IP.String(), laddr.Port)
	slog.Infof("node tcp listen at %s", laddr)
}
