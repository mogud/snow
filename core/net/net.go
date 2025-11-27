package net

import (
	"net"
)

type IPreprocessor interface {
	Process(conn net.Conn) error
}
