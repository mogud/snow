package snow

import (
	"net"
	"snow/core/logging/slog"
	"sort"
)

type nodeElementOption struct {
	Host     string   `koanf:"host"`     // 节点地址
	Port     int      `koanf:"port"`     // 节点端口
	Services []string `koanf:"services"` // 具名服务
}

type NodeBootOption struct {
	BootName string                        `koanf:"boot_name"` // 启动节点名
	Nodes    map[string]*nodeElementOption `koanf:"nodes"`     // 所有节点数据
}

type nodeInfo struct {
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
	records := make(map[NodeAddr]bool)
	for name, nc := range opt.Nodes {
		naddr, err := NewNodeAddr(nc.Host, nc.Port)
		if _, ok := records[naddr]; ok && naddr != 0 {
			slog.Fatalf("node(%s) address already exist", name)
		}
		records[naddr] = true

		if err != nil {
			slog.Fatalf("invalid node(%s) address: %+v", name, err)
		}

		cfg := &nodeInfo{
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
		return NodeConfig.Nodes[i].NodeAddr < NodeConfig.Nodes[j].NodeAddr
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
