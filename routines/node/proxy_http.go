package node

import (
	"bytes"
	"crypto/tls"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/mogud/snow/core/task"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime/debug"
	"time"
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

func newHttpProxy(srv *Service, name string) *httpProxy {
	var urlBase string
	if Config.CurNodeMap[name] {
		urlBase = fmt.Sprintf("http://%s:%v", Config.CurNodeIP, Config.CurNodeHttpPort)
	} else {
	loop:
		for _, ni := range Config.Nodes {
			if ni.Name == Config.CurNodeName {
				continue
			}

			for _, n := range ni.Services {
				if n == name && len(ni.Host) > 0 && ni.HttpPort > 0 {
					protocol := "http"
					if ni.UseHttps {
						protocol = "https"
					}
					urlBase = fmt.Sprintf("%v://%s:%v", protocol, ni.Host, ni.HttpPort)
					break loop
				}
			}
		}
	}

	if len(urlBase) == 0 {
		return nil
	}

	// TODO by mogu: Golang HTTP2 有 bug，会导致超时访问，使用 HTTP1 可以绕过
	tr := &http.Transport{}
	tr.TLSClientConfig = &tls.Config{
		NextProtos: []string{"h1"},
	}

	res, _ := url.JoinPath(urlBase, httpRpcPathPrefix, name)
	return &httpProxy{
		srv: srv,
		url: res,
		httpClient: &http.Client{
			Timeout:   time.Second * 8,
			Transport: tr,
		},
	}
}

func (ss *httpProxy) onError(p *promise, err error) {
	if p.errCb != nil {
		ss.srv.Fork("httpProxy.doCall.errCb", func() {
			p.errCb(err)
		})
	} else {
		ss.srv.Errorf("httpRpc(%s) uncatched error: %+v", p.fName, err)
	}

	if p.finalCb != nil {
		ss.srv.Fork("httpProxy.doCall.finalCb", p.finalCb)
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
			Post: len(p.successCb) == 0,
			Args: argsStr,
		}

		var httpClient *http.Client
		if p.timeout == -1 {
			httpClient = ss.httpClient
		} else {
			httpClient = &*httpClient
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

	if len(p.successCb) == 0 {
		if p.finalCb != nil {
			ss.srv.Fork("httpProxy.post.finalCb", p.finalCb)
		}
		return
	}

	fv := reflect.ValueOf(p.successCb[0])
	if !fv.IsValid() {
		if p.finalCb != nil {
			ss.srv.Fork("httpProxy.post.finalCb", p.finalCb)
		}
		return
	}

	ft := fv.Type()
	p.successCb = p.successCb[1:]

	var rsp httpResponse
	err = jsoniter.Unmarshal(rspBody, &rsp)
	if err != nil {
		ss.onError(p, err)
		return
	}

	resArgs := make([]any, 0, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		resArgs = append(resArgs, reflect.New(ft.In(i)).Interface())
	}
	if err = jsoniter.Unmarshal(rsp.Result, &resArgs); err != nil {
		ss.onError(p, err)
		return
	}

	rArgs := make([]reflect.Value, len(resArgs))
	for i, v := range resArgs {
		if v == nil {
			rArgs[i] = reflect.Zero(ft.In(i))
		} else {
			rArgs[i] = reflect.ValueOf(v).Elem()
		}
	}

	ss.callThen(p, ft, fv, rArgs)
}

func (ss *httpProxy) callThen(p *promise, ft reflect.Type, fv reflect.Value, rArgs []reflect.Value) {
	ss.srv.Fork("httpProxy.fork", func() {
		nextProxy := false
		if p.finalCb != nil {
			defer func() {
				if !nextProxy {
					ss.srv.Fork("httpProxy.req.finalCb", p.finalCb)
				}
			}()
		}

		panicked := true
		defer func() {
			if panicked {
				ss.srv.Errorf("httpRpc(%s) response got panic: %v", p.fName, string(debug.Stack()))
			}
		}()

		fret := fv.Call(rArgs)
		for len(p.successCb) > 0 {
			if ft.NumOut() == 1 && ft.Out(0) == reflect.TypeOf((*IPromise)(nil)).Elem() {
				if fret[0].IsNil() {
					break
				}
				np := fret[0].Interface().(iPromise)
				np.setCallBacks(p.successCb, p.errCb, p.finalCb)
				np.Done()
				nextProxy = true
				break
			}

			fv = reflect.ValueOf(p.successCb[0])
			ft = fv.Type()
			fret = fv.Call(fret)
			p.successCb = p.successCb[1:]
		}
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
	ss.ch <- &httpResponse{
		StatusCode: http.StatusBadRequest,
		Result:     jsoniter.RawMessage(err.Error()),
	}
}

func (ss *httpRpcContext) onError(err error) {
	ss.errF(err)
}
