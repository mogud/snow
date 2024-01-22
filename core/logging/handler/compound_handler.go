package handler

import (
	"gitee.com/mogud/snow/core/container"
	"gitee.com/mogud/snow/core/logging"
	"reflect"
	"unsafe"
)

var _ logging.ILogHandler = (*CompoundHandler)(nil)

type CompoundHandler struct {
	proxy container.List[logging.ILogHandler]
}

func NewCompoundHandler() *CompoundHandler {
	return &CompoundHandler{
		proxy: container.NewList[logging.ILogHandler](),
	}
}

func (ss *CompoundHandler) Log(data *logging.LogData) {
	for _, handler := range ss.proxy {
		handler.Log(data)
	}
}

func (ss *CompoundHandler) AddHandler(logger logging.ILogHandler) {
	ss.proxy.Add(logger)
}

func (ss *CompoundHandler) WrapToContainer(ty reflect.Type) any {
	keyTy := ty.Elem().Field(0).Type

	instanceValue := reflect.New(ty.Elem())
	instance := instanceValue.Interface()

	fieldValue := reflect.ValueOf(ss)

	field := reflect.NewAt(keyTy, unsafe.Pointer(instanceValue.Elem().Field(0).UnsafeAddr())).Elem()
	field.Set(fieldValue)
	return instance
}
