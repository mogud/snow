package handler

import (
	"github.com/mogud/snow/core/logging"
	"reflect"
	"unsafe"
)

var _ logging.ILogHandler = (*RootHandler)(nil)

type RootHandler struct {
	proxy logging.ILogHandler
}

func NewRootHandler(proxy logging.ILogHandler) *RootHandler {
	return &RootHandler{
		proxy: proxy,
	}
}

func (ss *RootHandler) Log(data *logging.LogData) {
	ss.proxy.Log(data)
}

func (ss *RootHandler) WrapToContainer(ty reflect.Type) any {
	keyTy := ty.Elem().Field(0).Type

	instanceValue := reflect.New(ty.Elem())
	instance := instanceValue.Interface()

	fieldValue := reflect.ValueOf(ss)

	field := reflect.NewAt(keyTy, unsafe.Pointer(instanceValue.Elem().Field(0).UnsafeAddr())).Elem()
	field.Set(fieldValue)
	return instance
}
