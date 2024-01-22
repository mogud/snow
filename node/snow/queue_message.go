package snow

import (
	"sync/atomic"
	"unsafe"
)

func msgNodeLoad(p *unsafe.Pointer) (n *msgNode) {
	return (*msgNode)(atomic.LoadPointer(p))
}

func msgNodeCas(p *unsafe.Pointer, old, new *msgNode) (ok bool) {
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}

type msgQueue struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}

type msgNode struct {
	value *message
	next  unsafe.Pointer
}

func newMsgQueue() (q *msgQueue) {
	n := unsafe.Pointer(&msgNode{})
	q = &msgQueue{head: n, tail: n}
	return
}

func (ss *msgQueue) empty() bool {
	return msgNodeLoad(&ss.head) == msgNodeLoad(&ss.tail)
}

func (ss *msgQueue) enq(v *message) {
	n := &msgNode{value: v}
	for {
		last := msgNodeLoad(&ss.tail)
		next := msgNodeLoad(&last.next)
		if last == msgNodeLoad(&ss.tail) {
			if next == nil {
				if msgNodeCas(&last.next, next, n) {
					msgNodeCas(&ss.tail, last, n)
					return
				}
			} else {
				msgNodeCas(&ss.tail, last, next)
			}
		}
	}
}

func (ss *msgQueue) deq() *message {
	for {
		first := msgNodeLoad(&ss.head)
		last := msgNodeLoad(&ss.tail)
		next := msgNodeLoad(&first.next)
		if first == msgNodeLoad(&ss.head) {
			if first == last {
				if next == nil {
					return nil
				}
				msgNodeCas(&ss.tail, last, next)
			} else {
				v := next.value
				if msgNodeCas(&ss.head, first, next) {
					return v
				}
			}
		}
	}
}
