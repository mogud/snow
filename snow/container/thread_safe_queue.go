// Copyright 2022 all contributors.
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"sync/atomic"
	"unsafe"
)

func load[T any](p *unsafe.Pointer) (n *node[T]) {
	return (*node[T])(atomic.LoadPointer(p))
}

func cas[T any](p *unsafe.Pointer, old, new *node[T]) (ok bool) {
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}

type ThreadSafeQueue[T any] struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}

type node[T any] struct {
	value T
	next  unsafe.Pointer
}

// NewThreadSafeQueue creates a lock-free queue
func NewThreadSafeQueue[T any]() (q *ThreadSafeQueue[T]) {
	n := unsafe.Pointer(&node[T]{})
	q = &ThreadSafeQueue[T]{head: n, tail: n}
	return
}

func (q *ThreadSafeQueue[T]) Empty() bool {
	return load[T](&q.head) == load[T](&q.tail)
}

func (q *ThreadSafeQueue[T]) Enq(v T) {
	n := &node[T]{value: v}
	for {
		last := load[T](&q.tail)
		next := load[T](&last.next)
		if last == load[T](&q.tail) {
			if next == nil {
				if cas(&last.next, next, n) {
					cas(&q.tail, last, n)
					return
				}
			} else {
				cas(&q.tail, last, next)
			}
		}
	}
}

func (q *ThreadSafeQueue[T]) Deq() (ret T) {
	for {
		first := load[T](&q.head)
		last := load[T](&q.tail)
		next := load[T](&first.next)
		if first == load[T](&q.head) {
			if first == last {
				if next == nil {
					return
				}
				cas(&q.tail, last, next)
			} else {
				v := next.value
				if cas(&q.head, first, next) {
					return v
				}
			}
		}
	}
}
