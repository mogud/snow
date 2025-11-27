package node

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/mogud/snow/core/logging/slog"
	"github.com/mogud/snow/core/task"
	"github.com/mogud/snow/core/ticker"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var _ iMessageSender = (*remoteHandle)(nil)
var _ ticker.PoolItem = (*remoteHandle)(nil)

var ErrRemoteDisconnected = fmt.Errorf("remote disconnected")

type session struct {
	timeout time.Time
	cb      func(m *message)
	trace   int64
}

type remoteHandle struct {
	node   *Node
	conn   net.Conn
	nAddr  Addr
	status int32
	ctx    context.Context
	cancel func()

	timeout           int
	lastSessCheckTime time.Time
	sessCb            sync.Map // [request_code int]*session;
	wBuf              chan []byte
	wBufferLock       sync.Mutex
	wBuffer           []*message
	wg                sync.WaitGroup
}

func newServerHandle(node *Node, nAddr Addr, conn net.Conn) *remoteHandle {
	ss := newRemoteHandle(node, nAddr, conn)
	if err := node.shPreprocessor.Process(conn); err != nil {
		slog.Debugf("illegal remote(%s) connection: %v", conn.RemoteAddr().String(), err)
		_ = conn.Close()
	}

	slog.Infof("node remote(%v) conntected", nAddr)
	return ss
}

func newRemoteHandle(node *Node, nAddr Addr, conn net.Conn) *remoteHandle {
	h := &remoteHandle{
		node:   node,
		conn:   conn,
		nAddr:  nAddr,
		status: 0,
		wBuf:   make(chan []byte, 4*1024),
	}
	h.ctx, h.cancel = context.WithCancel(context.Background())
	return h
}

func (ss *remoteHandle) Paused() bool {
	return false
}

func (ss *remoteHandle) Closed() bool {
	return ss.closed()
}

func (ss *remoteHandle) send(m *message) bool {
	if ss.closed() {
		slog.Debugf("remote handle(%v) closed", ss.nAddr)

		// 关闭时，若有回调，则立即调用
		if m != nil {
			if m.cb != nil {
				em := &message{
					trace: m.trace,
					err:   ErrRemoteDisconnected,
				}
				m.cb(em)
			}
			m.clear()
		}

		return false
	}

	// 若是请求，则设置超时回调
	if m != nil && m.sess > 0 {
		s := &session{
			cb:    m.cb,
			trace: m.trace,
		}
		if m.timeout > 0 {
			s.timeout = time.Now().Add(m.timeout)
		}
		ss.sessCb.Store(m.sess, s)

		if ss.closed() {
			slog.Debugf("remote handle(%v) closed, close all sessions", ss.nAddr)

			ss.closeAllSession()
			return true
		}
	}

	ss.wBufferLock.Lock()
	defer ss.wBufferLock.Unlock()

	ss.wBuffer = append(ss.wBuffer, m)
	return true
}

func (ss *remoteHandle) startClient() {
	ss.start(false)
}

func (ss *remoteHandle) startServer() {
	ss.start(true)
}

func (ss *remoteHandle) start(isReceiver bool) {
	ss.node.remoteHandleTickerPool.Add(ss)

	ss.wg.Add(2)
	task.Execute(ss.doSend)
	task.Execute(func() { ss.doReceive(isReceiver) })

	ss.wg.Wait()

	_ = ss.conn.Close()

	ss.safeDelete()
}

func (ss *remoteHandle) safeDelete() {
	if atomic.CompareAndSwapInt32(&ss.status, 0, 1) {
		nodeDelRemoteHandle(ss.nAddr)
		ss.closeAllSession()
	}
}

func (ss *remoteHandle) closed() bool {
	return atomic.LoadInt32(&ss.status) == 1
}

func (ss *remoteHandle) closeAllSession() {
	ss.sessCb.Range(func(key, _ any) bool {
		value, ok := ss.sessCb.LoadAndDelete(key)
		if ok {
			v := value.(*session)
			m := &message{
				err:   ErrRemoteDisconnected,
				trace: v.trace,
			}
			v.cb(m)
			v.cb = nil
		}

		return true
	})
}

func (ss *remoteHandle) onTick() {
	zero := time.Time{}
	now := time.Now()

	if now.Sub(ss.lastSessCheckTime) > 10*time.Second {
		ss.lastSessCheckTime = now

		ss.sessCb.Range(func(key, value any) bool {
			v := value.(*session)
			if !v.timeout.Equal(zero) && v.timeout.Before(now) {
				ss.sessCb.Delete(key)

				m := &message{
					trace: v.trace,
					err:   ErrRequestTimeoutRemote,
				}
				v.cb(m)
			}
			return true
		})
	}

	ss.wBufferLock.Lock()
	msgList := ss.wBuffer
	ss.wBuffer = nil
	ss.wBufferLock.Unlock()

	if len(msgList) > 0 {
		buffer := make([]byte, 0, 4*1024)
		for _, m := range msgList {
			bs, err := m.marshal()
			if m != nil {
				m.clear()
			}
			if err != nil {
				slog.Errorf("message marshal %v", err.Error())
				return
			}
			buffer = append(buffer, bs...)
		}

		select {
		case ss.wBuf <- buffer:
		default:
			slog.Fatalf("write to remote(%v) while channel full", ss.nAddr)
			return
		}
	}
}

func (ss *remoteHandle) doSend() {
	defer func() {
		ss.cancel()
		ss.wg.Done()
	}()

	for {
		select {
		case <-ss.ctx.Done():
			return
		case bs := <-ss.wBuf:
			n, err := ss.conn.Write(bs)
			select {
			case <-ss.ctx.Done():
				return
			default:
			}

			if err != nil {
				slog.Errorf("write to remote(%v) error: %+v", ss.nAddr, err)
				return
			}
			if n != len(bs) {
				slog.Errorf("write to remote(%v) error: length not match", ss.nAddr)
				return
			}
		}
	}
}

func (ss *remoteHandle) doReceive(isReceiver bool) {
	var data []byte
	buf := make([]byte, 4*1024)
	c := ss.conn

	defer func() {
		ss.cancel()
		ss.wg.Done()
	}()

	for {
		_ = c.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := c.Read(buf)
		select {
		case <-ss.ctx.Done():
			return
		default:
		}

		if err != nil {
			var opErr *net.OpError
			if errors.As(err, &opErr) && opErr.Timeout() {
				if isReceiver {
					if ss.timeout > 9 {
						// 发送 ping 消息
						ss.send((*message)(nil))
						ss.timeout = 0
					} else {
						ss.timeout++
					}
				}
				continue
			} else if errors.Is(err, io.EOF) {
				slog.Debugf("read from remote(%v) while remote closed the connection", ss.nAddr)
			} else if errors.As(err, &opErr) && strings.Contains(opErr.Err.Error(), "forcibly closed by the remote host") {
				slog.Debugf("read from remote(%v) while remote forcibly closed the connection", ss.nAddr)
			} else {
				slog.Errorf("read from remote(%v) while connection lost, err: %v", ss.nAddr, err)
			}
			return
		}

		data = append(data, buf[:n]...)
		data = ss.doDivide(data)
		if data == nil {
			return
		}
	}
}

func (ss *remoteHandle) doDivide(data []byte) []byte {
	for {
		if len(data) < 4 {
			break
		}
		msgLen := int(binary.LittleEndian.Uint32(data[:4]))
		if msgLen < 4 {
			slog.Errorf("net message from %v format error", ss.nAddr)
			return nil
		}
		if len(data) < msgLen {
			break
		}
		msg := make([]byte, msgLen)
		copy(msg, data)

		m := &message{}
		if err := m.unmarshal(msg); err != nil {
			slog.Errorf("net message from %v decode error", ss.nAddr)
			return nil
		}

		if m.dst != 0 {
			// dst 不为 0，不是 ping 包，需要处理

			m.nAddr = ss.nAddr
			ss.doDispatch(m)
		}

		// dst == 0 代表是 ping 包，ping 包为全 0 的 4 个字节
		data = data[msgLen:]
	}
	return data
}

func (ss *remoteHandle) doDispatch(m *message) {
	if m.sess < 0 {
		// response

		scb, ok := ss.sessCb.Load(-m.sess)
		if ok {
			ss.sessCb.Delete(-m.sess)

			cbSess := scb.(*session)
			cbSess.cb(m)
			cbSess.cb = nil
		} else {
			slog.Errorf("no session(%v) callback found, message data: %+v", -m.sess, m)
			m.clear()
		}
		return
	}

	if m.src == 0 {
		// error occurs

		slog.Warnf("remote error, code: %+v", m.getError())
		m.clear()
		return
	}

	// request
	srv := nodeGetService(m.dst)
	if srv != nil {
		srv.send(m)
	} else {
		slog.Warnf("remote(%v) call service(%d) which not found, message data: %+v", ss.nAddr, m.dst, m)
		mm := &message{
			nAddr: m.nAddr,
			src:   0,
			dst:   m.src,
			sess:  -m.sess,
			trace: m.trace,
			err:   fmt.Errorf("invalid address"),
		}
		ss.send(mm)
		m.clear()
	}
}
