package snow

import (
	"fmt"
	"gitee.com/mogud/snow/node"
	"net"
	"strconv"
)

var _ = (node.INodeAddr)((*NodeAddr)(nil))

var (
	NodeAddrInvalid = NodeAddr(-1)
	NodeAddrLocal   = NodeAddr(0)
)

type NodeAddr int64

func (ss NodeAddr) IsLocalhost() bool {
	return ss == 0 || net.IP([]byte{byte(ss >> 56), byte(ss >> 48), byte(ss >> 40), byte(ss >> 32)}).IsLoopback()
}

func (ss NodeAddr) GetIPString() string {
	return net.IP([]byte{byte(ss >> 56), byte(ss >> 48), byte(ss >> 40), byte(ss >> 32)}).String()
}

func (ss NodeAddr) String() string {
	return (&net.TCPAddr{
		IP:   net.IP([]byte{byte(ss >> 56), byte(ss >> 48), byte(ss >> 40), byte(ss >> 32)}),
		Port: int(ss & 0xffff),
	}).String()
}

func NewNodeAddr(host string, port int) (NodeAddr, error) {
	ipaddr, err := net.ResolveTCPAddr("tcp4", host+":"+strconv.Itoa(port))
	if err != nil {
		return 0, err
	}

	if len(ipaddr.IP) == 0 {
		return NodeAddr(port), nil
	}

	v4 := ipaddr.IP.To4()
	if v4 == nil {
		return 0, fmt.Errorf("IPv6(%v) is not supported", host+":"+strconv.Itoa(port))
	}

	return NodeAddr(v4[0])<<56 | NodeAddr(v4[1])<<48 | NodeAddr(v4[2])<<40 | NodeAddr(v4[3])<<32 | NodeAddr(port), nil
}
