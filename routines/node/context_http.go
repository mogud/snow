package node

import (
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

var _ IRpcContext = (*httpRpcContext)(nil)

type httpRpcContext struct {
	ch   chan *httpResponse
	errF func(error)
}

func newHttpRpcContext(ch chan *httpResponse) *httpRpcContext {
	return &httpRpcContext{
		ch: ch,
	}
}

func (ss *httpRpcContext) GetRemoteNodeAddr() INodeAddr {
	return Addr(0)
}

func (ss *httpRpcContext) GetRemoteServiceAddr() int32 {
	return 0
}

func (ss *httpRpcContext) Catch(f func(error)) IRpcContext {
	ss.errF = f
	return ss
}

func (ss *httpRpcContext) Return(args ...any) {
	if ss.ch == nil {
		return
	}

	if args == nil {
		args = make([]any, 0)
	}
	argsStr, err := jsoniter.Marshal(args)
	if err != nil {
		ss.ch <- &httpResponse{
			StatusCode: http.StatusInternalServerError,
			Result:     jsoniter.RawMessage(err.Error()),
		}
		return
	}

	ss.ch <- &httpResponse{
		StatusCode: http.StatusOK,
		Result:     argsStr,
	}
}

func (ss *httpRpcContext) Error(err error) {
	if ss.ch == nil {
		return
	}

	ss.ch <- &httpResponse{
		StatusCode: http.StatusBadRequest,
		Result:     jsoniter.RawMessage(err.Error()),
	}
}

func (ss *httpRpcContext) onError(err error) {
	ss.errF(err)
}
