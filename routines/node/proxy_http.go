package node

import (
	"bytes"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/mogud/snow/core/task"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"
)

const httpRpcPathPrefix = "/node/rpc/"

type httpRequest struct {
	Func string              `json:"Func"`
	Post bool                `json:"Post"`
	Args jsoniter.RawMessage `json:"Args"`
}

type httpResponse struct {
	StatusCode int                 `json:"-"`
	Result     jsoniter.RawMessage `json:"Result"`
}

var _ iProxy = (*httpProxy)(nil)

type httpProxy struct {
	srv        *Service
	url        string
	httpClient *http.Client
}

func (ss *httpProxy) onError(p *promise, err error) {
	defer func() {
		ss.srv.Fork("httpProxy.doCall.finalCb", func() {
			if p.finalCb != nil {
				p.finalCb()
			}
			p.clear()
		})
	}()

	if p.errCb != nil {
		ss.srv.Fork("httpProxy.doCall.errCb", func() {
			p.errCb(err)
		})
	} else {
		ss.srv.Errorf("httpRpc(%s) uncatched error: %+v", p.fName, err)
	}
}

func (ss *httpProxy) doCall(p *promise) {
	task.Execute(func() {
		argsStr, err := jsoniter.Marshal(p.args)
		if err != nil {
			ss.onError(p, err)
			return
		}

		req := &httpRequest{
			Func: p.fName,
			Post: p.successCb == nil,
			Args: argsStr,
		}

		var httpClient *http.Client
		if p.timeout == -1 {
			httpClient = ss.httpClient
		} else {
			c := *ss.httpClient
			httpClient = &c
			httpClient.Timeout = p.timeout
		}

		bs, _ := jsoniter.ConfigDefault.Marshal(req)
		resp, err := httpClient.Post(ss.url, "application/json", bytes.NewBuffer(bs))
		if err != nil {
			ss.onError(p, err)
			return
		}

		ss.prepareThen(p, resp)
	})
}

func (ss *httpProxy) prepareThen(p *promise, resp *http.Response) {
	rspBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if err != nil {
		ss.onError(p, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		ss.onError(p, fmt.Errorf("http response code(%v): %v", resp.StatusCode, string(rspBody)))
		return
	}

	if p.successCb == nil {
		ss.srv.Fork("httpProxy.post.finalCb.noSuccessor", func() {
			if p.finalCb != nil {
				p.finalCb()
			}
			p.clear()
		})
		return
	}

	fv := reflect.ValueOf(p.successCb)
	if !fv.IsValid() {
		ss.onError(p, fmt.Errorf("invalid success callback"))
		return
	}

	var rsp httpResponse
	err = jsoniter.Unmarshal(rspBody, &rsp)
	if err != nil {
		ss.onError(p, err)
		return
	}

	ft := fv.Type()
	resArgs := make([]any, 0, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		resArgs = append(resArgs, reflect.New(ft.In(i)).Interface())
	}
	if err = jsoniter.Unmarshal(rsp.Result, &resArgs); err != nil {
		ss.onError(p, err)
		return
	}

	fArgs := make([]reflect.Value, len(resArgs))
	for i, v := range resArgs {
		if v == nil {
			fArgs[i] = reflect.Zero(ft.In(i))
		} else {
			fArgs[i] = reflect.ValueOf(v).Elem()
		}
	}

	ss.callThen(p, fv, fArgs)
}

func (ss *httpProxy) callThen(p *promise, fv reflect.Value, fArgs []reflect.Value) {
	ss.srv.Fork("httpProxy.fork", func() {
		panicked := true
		defer func() {
			if panicked {
				ss.srv.Errorf("httpRpc(%s) response got panic: %v", p.fName, string(debug.Stack()))
			}

			if p.finalCb != nil {
				p.finalCb()
			}
			p.clear()
		}()

		fRet := fv.Call(fArgs)

		for _, arg := range fArgs {
			if arg.CanAddr() {
				arg.SetZero()
			}
		}
		fArgs = nil
		for _, arg := range fRet {
			if arg.CanAddr() {
				arg.SetZero()
			}
		}
		fRet = nil

		panicked = false
	})
}

func (ss *httpProxy) Call(fName string, args ...any) IPromise {
	return newPromise(ss, fName, args)
}

func (ss *httpProxy) GetNodeAddr() INodeAddr {
	return Addr(0)
}

func (ss *httpProxy) Avail() bool {
	return true
}
