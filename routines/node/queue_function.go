package node

import (
	"sync/atomic"
	"unsafe"
)

type tagFunc struct {
	Tag string
	F   func()
}

func funcNodeLoad(p *unsafe.Pointer) (n *funcNode) {
	return (*funcNode)(atomic.LoadPointer(p))
}

func funcNodeCas(p *unsafe.Pointer, old, new *funcNode) (ok bool) {
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}

type funcQueue struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}

type funcNode struct {
	value *tagFunc
	next  unsafe.Pointer
}

func newFuncQueue() (q *funcQueue) {
	n := unsafe.Pointer(&funcNode{})
	q = &funcQueue{head: n, tail: n}
	return
}

func (ss *funcQueue) empty() bool {
	return funcNodeLoad(&ss.head) == funcNodeLoad(&ss.tail)
}

func (ss *funcQueue) enq(v *tagFunc) {
	n := &funcNode{value: v}
	for {
		last := funcNodeLoad(&ss.tail)
		next := funcNodeLoad(&last.next)
		if last == funcNodeLoad(&ss.tail) {
			if next == nil {
				if funcNodeCas(&last.next, next, n) {
					funcNodeCas(&ss.tail, last, n)
					return
				}
			} else {
				funcNodeCas(&ss.tail, last, next)
			}
		}
	}
}

func (ss *funcQueue) deq() *tagFunc {
	for {
		first := funcNodeLoad(&ss.head)
		last := funcNodeLoad(&ss.tail)
		next := funcNodeLoad(&first.next)
		if first == funcNodeLoad(&ss.head) {
			if first == last {
				if next == nil {
					return nil
				}
				funcNodeCas(&ss.tail, last, next)
			} else {
				v := next.value
				if funcNodeCas(&ss.head, first, next) {
					return v
				}
			}
		}
	}
}
