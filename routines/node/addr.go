package node

import (
	"fmt"
	"net"
	"strconv"
)

var _ = (INodeAddr)((*Addr)(nil))

var (
	AddrLocal   = Addr(0)
	AddrRemote  = Addr(-1)
	AddrInvalid = Addr(-2)
)

type Addr int64

func (ss Addr) IsLocalhost() bool {
	return ss == 0 || net.IP([]byte{byte(ss >> 56), byte(ss >> 48), byte(ss >> 40), byte(ss >> 32)}).IsLoopback()
}

func (ss Addr) GetIPString() string {
	return net.IP([]byte{byte(ss >> 56), byte(ss >> 48), byte(ss >> 40), byte(ss >> 32)}).String()
}

func (ss Addr) GetPort() int {
	return int(ss & 0xffff)
}

func (ss Addr) String() string {
	return (&net.TCPAddr{
		IP:   []byte{byte(ss >> 56), byte(ss >> 48), byte(ss >> 40), byte(ss >> 32)},
		Port: int(ss & 0xffff),
	}).String()
}

func NewNodeAddr(host string, port int) (Addr, error) {
	ipaddr, err := net.ResolveTCPAddr("tcp4", host+":"+strconv.Itoa(port))
	if err != nil {
		return 0, err
	}

	if len(ipaddr.IP) == 0 {
		return Addr(port), nil
	}

	v4 := ipaddr.IP.To4()
	if v4 == nil {
		return 0, fmt.Errorf("IPv6(%v) is not supported", host+":"+strconv.Itoa(port))
	}

	return Addr(v4[0])<<56 | Addr(v4[1])<<48 | Addr(v4[2])<<40 | Addr(v4[3])<<32 | Addr(port), nil
}
