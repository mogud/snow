package net

import (
	"io"
	"net"
)

type IDivider[T any] interface {
	Divide(reader io.Reader) (T, []byte, error)
}

type IPreprocessor interface {
	Process(conn net.Conn) (io.Reader, io.Writer, error)
}
