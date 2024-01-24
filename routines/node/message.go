package node

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type iMessageSender interface {
	send(msg *message) bool
	closed() bool
}

type message struct {
	naddr   NodeAddr         // do not marshal
	cb      func(m *message) // do not marshal, used by inner node rpc
	timeout time.Duration    // do not marshal
	src     int32            // 0 if error occurs
	dst     int32            // kind or address, 0 if is ping package
	sess    int32            // req: > 0, post: == 0, resp: < 0
	err     error            // ok: nil
	fname   string           // req: len() > 0; resp: len() == 0
	args    []reflect.Value  // not nil: request
	data    []byte           // remote call: not nil; local call: nil(marshal not needed)
}

func (ss *message) marshalArgs(args []reflect.Value) ([]byte, error) {
	margs := make([]interface{}, 0, len(args))
	for _, arg := range args {
		margs = append(margs, arg.Interface())
	}

	bs, err := jsoniter.ConfigDefault.Marshal(margs)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (ss *message) unmarshalArgs(bs []byte, argi int, ft reflect.Type) ([]reflect.Value, error) {
	targs := make([]interface{}, 0, ft.NumIn()-argi)
	for i := argi; i < ft.NumIn(); i++ {
		targs = append(targs, reflect.New(ft.In(i)).Interface())
	}
	if err := jsoniter.ConfigDefault.Unmarshal(bs, &targs); err != nil {
		return nil, err
	}
	ret := make([]reflect.Value, 0, len(targs))
	for _, arg := range targs {
		ret = append(ret, reflect.ValueOf(arg).Elem())
	}
	return ret, nil
}

func (ss *message) marshal() ([]byte, error) {
	if ss == nil {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, 4)
		return buf, nil
	}

	if ss.err != nil { // error
		ss.data = []byte(fmt.Sprintf("%+v", ss.err))
	} else if len(ss.fname) > 0 { // request
		lf := len(ss.fname)
		data := make([]byte, 2+lf)
		binary.LittleEndian.PutUint16(data[:2], uint16(2+len(ss.fname)))
		copy(data[2:], []byte(ss.fname))

		bs, err := ss.marshalArgs(ss.args)
		if err != nil {
			return nil, err
		}
		data = append(data, bs...)
		ss.data = data
	} else if ss.args != nil { // response
		bs, err := ss.marshalArgs(ss.args)
		if err != nil {
			return nil, err
		}
		ss.data = bs
	} else {
		_ = 0
		// forward message only has data
	}

	ml := 16 + len(ss.data)
	buf := make([]byte, ml)
	binary.LittleEndian.PutUint32(buf[:4], uint32(ml))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(ss.src))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(ss.dst))
	binary.LittleEndian.PutUint32(buf[12:16], uint32(ss.sess))
	copy(buf[16:], ss.data)
	return buf, nil
}

func (ss *message) unmarshal(bytes []byte) error {
	bl := len(bytes)
	if bl == 4 {
		return nil
	}

	if bl < 16 {
		return fmt.Errorf("message decode length expected >= 16, got %d", bl)
	}

	// bytes[:4] is length that must be checked before decode
	src := int32(binary.LittleEndian.Uint32(bytes[4:8]))
	dst := int32(binary.LittleEndian.Uint32(bytes[8:12]))
	sess := int32(binary.LittleEndian.Uint32(bytes[12:16]))
	ss.src = src
	ss.dst = dst
	ss.sess = sess
	ss.data = make([]byte, bl-16)
	copy(ss.data, bytes[16:])

	return nil
}

func (ss *message) writeError(err error) {
	ss.err = err
}

func (ss *message) getError() error {
	if ss.err != nil {
		return ss.err
	}
	return fmt.Errorf("%s", string(ss.data))
}

func (ss *message) writeRequest(fname string, args []interface{}) {
	ss.fname = fname
	for _, arg := range args {
		ss.args = append(ss.args, reflect.ValueOf(arg))
	}
}

func (ss *message) getRequestFuncLen() (int, error) {
	if len(ss.data) < 2 {
		return 0, fmt.Errorf("getRequestFuncLen: message length < 2: %d", len(ss.data))
	}
	lf := int(binary.LittleEndian.Uint16(ss.data[:2]))
	if len(ss.data) < lf {
		return 0, fmt.Errorf("getRequestFuncLen: message length < %d: %d", lf, len(ss.data))
	}
	return lf, nil
}

func (ss *message) getRequestFunc() (string, error) {
	if len(ss.fname) > 0 {
		return ss.fname, nil
	}

	lf, err := ss.getRequestFuncLen()
	if err != nil {
		return "", err
	}
	return string(ss.data[2:lf]), nil
}

func (ss *message) getRequestFuncArgs(ft reflect.Type) ([]reflect.Value, error) {
	if len(ss.fname) > 0 {
		return ss.args, nil
	}

	lf, err := ss.getRequestFuncLen()
	if err != nil {
		return nil, err
	}
	bs := ss.data[lf:]

	args, err := ss.unmarshalArgs(bs, 2, ft)
	if err != nil {
		return nil, err
	}
	return args, nil
}

func (ss *message) writeResponse(args ...interface{}) {
	ss.args = []reflect.Value{}
	for _, arg := range args {
		ss.args = append(ss.args, reflect.ValueOf(arg))
	}
}

func (ss *message) getResponse(ft reflect.Type) ([]reflect.Value, error) {
	if ss.args != nil {
		return ss.args, nil
	}

	bs := ss.data
	args, err := ss.unmarshalArgs(bs, 0, ft)
	if err != nil {
		return nil, err
	}
	return args, nil
}
