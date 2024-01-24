package node

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"gitee.com/mogud/snow/logging/slog"
	"gitee.com/mogud/snow/task"
	"gitee.com/mogud/snow/ticker"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var _ = iMessageSender((*remoteHandle)(nil))

var ErrRemoteDisconnected = fmt.Errorf("remote disconnected")

type session struct {
	timeout time.Time
	cb      func(m *message)
}

type remoteHandle struct {
	node   *Node
	conn   net.Conn
	w      io.Writer
	r      io.Reader
	naddr  NodeAddr
	status int32
	ctx    context.Context
	cancel func()

	timeout int
	sessCb  sync.Map // [request_code int]*session;
	wbuf    chan *message
	wg      sync.WaitGroup
}

func newServerHandle(node *Node, naddr NodeAddr, conn net.Conn) *remoteHandle {
	ss := newRemoteHandle(node, naddr, conn)
	var err error
	if ss.r, ss.w, err = node.shPreprocessor.Process(conn); err != nil {
		slog.Debugf("illegal remote(%s) connection: %v", conn.RemoteAddr().String(), err)
		_ = conn.Close()
	}

	slog.Infof("node remote(%v) conntected", naddr)
	return ss
}

func newRemoteHandle(node *Node, naddr NodeAddr, conn net.Conn) *remoteHandle {
	h := &remoteHandle{
		node:   node,
		conn:   conn,
		naddr:  naddr,
		status: 0,
		wbuf:   make(chan *message, 1024*1024),
	}
	h.ctx, h.cancel = context.WithCancel(context.Background())
	return h
}

func (ss *remoteHandle) send(m *message) bool {
	if m != nil && m.sess > 0 {
		s := &session{
			cb: m.cb,
		}
		if m.timeout > 0 {
			s.timeout = time.Now().Add(m.timeout)
		}
		ss.sessCb.Store(m.sess, s)
	}

	select {
	case ss.wbuf <- m:
		return true
	default:
		if ss.closed() {
			ss.sessCb.Delete(m.sess)

			em := &message{}
			em.writeError(ErrRemoteDisconnected)
			m.cb(em)
		} else {
			slog.Warnf("remote handle(%v) message chan full", ss.naddr)
			if m.sess > 0 {
				ss.sessCb.Delete(m.sess)

				em := &message{}
				em.writeError(ErrNodeMessageChanFull)
				m.cb(em)
			}
		}
		return true
	}
}

func (ss *remoteHandle) startClient() {
	ss.start(false)
}

func (ss *remoteHandle) startServer() {
	ss.start(true)
}

func (ss *remoteHandle) start(isReceiver bool) {
	ss.wg.Add(3)

	task.Execute(ss.clearSession)
	task.Execute(ss.doSend)
	task.Execute(func() { ss.doReceive(isReceiver) })

	ss.wg.Wait()

	ss.safeDelete()
}

func (ss *remoteHandle) safeDelete() {
	if atomic.CompareAndSwapInt32(&ss.status, 0, 1) {
		nodeDelRemoteHandle(ss.naddr)
		close(ss.wbuf)
		ss.closeAllSession()
	}
}

func (ss *remoteHandle) closed() bool {
	return atomic.LoadInt32(&ss.status) == 1
}

func (ss *remoteHandle) clearSession() {
	ch := make(chan int64, 1024)
	ticker.Subscribe(-int64(ss.naddr), ch)
	prev := time.Now()
loop:
	for {
		select {
		case <-ss.ctx.Done():
			break loop
		case unixNano := <-ch:
			now := time.Unix(0, unixNano)
			if now.Sub(prev) < 10*time.Second {
				continue
			}
			prev = now

			m := &message{}
			m.writeError(ErrRequestTimeoutRemote)

			zero := time.Time{}
			ss.sessCb.Range(func(key, value interface{}) bool {
				v := value.(*session)
				if !v.timeout.Equal(zero) && v.timeout.Before(now) {
					ss.sessCb.Delete(key)

					v.cb(m)
				}
				return true
			})
		}
	}
	ticker.Unsubscribe(-int64(ss.naddr))
	ss.wg.Done()
}

func (ss *remoteHandle) closeAllSession() {
	m := &message{}
	m.writeError(ErrRemoteDisconnected)

	ss.sessCb.Range(func(key, value interface{}) bool {
		ss.sessCb.Delete(key)

		v := value.(*session)
		v.cb(m)
		return true
	})
}

func (ss *remoteHandle) doSend() {
loop:
	for {
		select {
		case <-ss.ctx.Done():
			break loop
		case m, ok := <-ss.wbuf:
			bs, err := m.marshal()
			if err != nil {
				slog.Errorf("message marshal %v", err.Error())
				return
			}

			if !ok {
				break loop
			}
			n, err := ss.conn.Write(bs)
			if err != nil {
				slog.Errorf("write to %v error: %+v", ss.naddr, err)
				break loop
			}
			if n != len(bs) {
				slog.Errorf("write to %v error: length not match", ss.naddr)
				break loop
			}
		}
	}
	_ = ss.conn.Close()
	ss.wg.Done()
}

func (ss *remoteHandle) doReceive(isReceiver bool) {
	var data []byte
	buf := make([]byte, 256)
	c := ss.conn

loop:
	for {
		_ = c.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := c.Read(buf)

		var e net.Error
		switch {
		case errors.As(err, &e) && e.Timeout():
			if isReceiver {
				if ss.timeout > 9 {
					ss.send((*message)(nil))
					ss.timeout = 0
				} else {
					ss.timeout++
				}
			}
			continue
		case errors.Is(err, net.ErrClosed):
			slog.Debugf("remote(%v) connection closed", ss.naddr)
			break loop
		case err != nil:
			slog.Warnf("receive remote(%v) message failed: %v", ss.naddr, err)
			break loop
		}

		select {
		case <-ss.ctx.Done():
			break loop
		default:
		}

		data = append(data, buf[:n]...)
		data = ss.doDivide(data)
		if data == nil {
			break
		}
	}
	ss.cancel()
	ss.wg.Done()
}

func (ss *remoteHandle) doDivide(data []byte) []byte {
	for {
		if len(data) < 4 {
			break
		}
		msgLen := int(binary.LittleEndian.Uint32(data[:4]))
		if msgLen < 4 {
			slog.Errorf("net message from %v format error", ss.naddr)
			return nil
		}
		if len(data) < msgLen {
			break
		}
		msg := data[:msgLen]

		m := &message{}
		if err := m.unmarshal(msg); err != nil {
			slog.Errorf("net message from %v decode error", ss.naddr)
			return nil
		}

		if m.dst != 0 { // dst == 0 代表是 ping 包
			m.naddr = ss.naddr
			ss.doDispatch(m)
		}
		data = data[msgLen:]
	}
	return data
}

func (ss *remoteHandle) doDispatch(m *message) {
	if m.sess < 0 { // response
		scb, ok := ss.sessCb.Load(-m.sess)
		if ok {
			ss.sessCb.Delete(-m.sess)

			scb.(*session).cb(m)
		} else {
			slog.Errorf("no session(%v) callback found, message data: %+v", -m.sess, m)
		}
		return
	}

	if m.src == 0 { // error occurs
		slog.Warnf("remote error, code: %+v", m.getError())
		return
	}

	// request
	srv := nodeGetService(m.dst)
	if srv != nil {
		srv.send(m)
	} else {
		slog.Warnf("remote(%v) call service(%d) which not found, message data: %+v", ss.naddr, m.dst, m)
		mm := &message{
			naddr: m.naddr,
			src:   0,
			dst:   m.src,
			sess:  -m.sess,
		}
		mm.writeError(fmt.Errorf("invalid address"))
		ss.send(mm)
	}
}
