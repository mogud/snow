package node

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const messageHeaderLen = 24

type iMessageSender interface {
	send(msg *message) bool
	closed() bool
}

type message struct {
	nAddr   Addr             // do not marshal
	cb      func(m *message) // do not marshal, used by inner node rpc
	timeout time.Duration    // do not marshal
	src     int32            // 0 if error occurs
	dst     int32            // kind or address, 0 if is ping package
	sess    int32            // req: > 0, post: == 0, resp: < 0
	trace   int64            // trace id
	err     error            // ok: nil
	fName   string           // req: len() > 0; resp: len() == 0
	args    []reflect.Value  // not nil: request
	data    []byte           // remote call: not nil; local call: nil(marshal not needed)
}

func (ss *message) marshalArgs(args []reflect.Value) ([]byte, error) {
	mArgs := make([]any, 0, len(args))
	for _, arg := range args {
		mArgs = append(mArgs, arg.Interface())
	}

	bs, err := jsoniter.ConfigDefault.Marshal(mArgs)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (ss *message) unmarshalArgs(bs []byte, argI int, ft reflect.Type) ([]reflect.Value, error) {
	tArgs := make([]any, 0, ft.NumIn()-argI)
	for i := argI; i < ft.NumIn(); i++ {
		tArgs = append(tArgs, reflect.New(ft.In(i)).Interface())
	}
	if err := jsoniter.ConfigDefault.Unmarshal(bs, &tArgs); err != nil {
		return nil, err
	}
	ret := make([]reflect.Value, 0, len(tArgs))
	for _, arg := range tArgs {
		ret = append(ret, reflect.ValueOf(arg).Elem())
	}
	return ret, nil
}

func (ss *message) marshal() ([]byte, error) {
	if ss == nil {
		// 发送 ping 包

		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, 4)
		return buf, nil
	}

	if ss.err != nil { // error
		// 若存在错误，则 data 中是错误的信息

		errBytes := []byte(fmt.Sprintf("%+v", ss.err))
		ss.data = make([]byte, messageHeaderLen+len(errBytes))
		copy(ss.data[messageHeaderLen:], errBytes)
	} else if len(ss.fName) > 0 { // request
		// fName 大于 0 代表是请求
		// 请求的格式：len(fName,2bytes) + fName + args

		lof := len(ss.fName)
		bs, err := ss.marshalArgs(ss.args)
		if err != nil {
			return nil, err
		}

		ss.data = make([]byte, messageHeaderLen+2+lof+len(bs))
		binary.LittleEndian.PutUint16(ss.data[messageHeaderLen:], uint16(2+lof))
		copy(ss.data[messageHeaderLen+2:], ss.fName)
		copy(ss.data[messageHeaderLen+2+lof:], bs)
	} else if ss.args != nil { // response
		// 没有 fName 但存在 args，即为 response

		bs, err := ss.marshalArgs(ss.args)
		if err != nil {
			return nil, err
		}
		ss.data = make([]byte, messageHeaderLen+len(bs))
		copy(ss.data[messageHeaderLen:], bs)
	} else {
		_ = 0
		// forward message only has data
	}

	binary.LittleEndian.PutUint32(ss.data[0:4], uint32(len(ss.data)))
	binary.LittleEndian.PutUint32(ss.data[4:8], uint32(ss.src))
	binary.LittleEndian.PutUint32(ss.data[8:12], uint32(ss.dst))
	binary.LittleEndian.PutUint32(ss.data[12:16], uint32(ss.sess))
	binary.LittleEndian.PutUint64(ss.data[16:24], uint64(ss.trace))
	return ss.data, nil
}

func (ss *message) unmarshal(bytes []byte) error {
	bl := len(bytes)
	if bl == 4 {
		// 收到的是 ping 包

		return nil
	}

	if bl < messageHeaderLen {
		return fmt.Errorf("message decode length expected >= %d, got %d", messageHeaderLen, bl)
	}

	// bytes[:4] is length that must be checked before decode
	src := int32(binary.LittleEndian.Uint32(bytes[4:8]))
	dst := int32(binary.LittleEndian.Uint32(bytes[8:12]))
	sess := int32(binary.LittleEndian.Uint32(bytes[12:16]))
	trace := int64(binary.LittleEndian.Uint64(bytes[16:24]))
	ss.src = src
	ss.dst = dst
	ss.sess = sess
	ss.trace = trace
	ss.data = bytes
	return nil
}

func (ss *message) getError() error {
	if ss.err != nil {
		return ss.err
	}
	return fmt.Errorf("%s", string(ss.data[messageHeaderLen:]))
}

func (ss *message) writeRequest(fName string, args []any) {
	ss.fName = fName
	for _, arg := range args {
		ss.args = append(ss.args, reflect.ValueOf(arg))
	}
}

func (ss *message) getRequestFuncLen() (int, error) {
	if len(ss.data) < messageHeaderLen+2 {
		return 0, fmt.Errorf("getRequestFuncLen: message length < %d: %d", messageHeaderLen+2, len(ss.data))
	}
	lof := int(binary.LittleEndian.Uint16(ss.data[messageHeaderLen:]))
	if len(ss.data) < messageHeaderLen+lof {
		return 0, fmt.Errorf("getRequestFuncLen: message length < %d: %d", messageHeaderLen+lof, len(ss.data))
	}
	return lof, nil
}

func (ss *message) getRequestFunc() (string, error) {
	if len(ss.fName) > 0 {
		return ss.fName, nil
	}

	lof, err := ss.getRequestFuncLen()
	if err != nil {
		return "", err
	}
	return string(ss.data[messageHeaderLen+2 : messageHeaderLen+lof]), nil
}

func (ss *message) getRequestFuncArgs(ft reflect.Type) ([]reflect.Value, error) {
	if len(ss.fName) > 0 {
		return ss.args, nil
	}

	lof, err := ss.getRequestFuncLen()
	if err != nil {
		return nil, err
	}
	bs := ss.data[messageHeaderLen+lof:]

	args, err := ss.unmarshalArgs(bs, 2, ft)
	if err != nil {
		return nil, err
	}
	return args, nil
}

func (ss *message) writeResponse(args ...any) {
	ss.args = []reflect.Value{}
	for _, arg := range args {
		ss.args = append(ss.args, reflect.ValueOf(arg))
	}
}

func (ss *message) getResponse(ft reflect.Type) ([]reflect.Value, error) {
	if ss.args != nil {
		return ss.args, nil
	}

	bs := ss.data[messageHeaderLen:]
	args, err := ss.unmarshalArgs(bs, 0, ft)
	if err != nil {
		return nil, err
	}
	return args, nil
}
