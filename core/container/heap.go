package container

import (
	"gitee.com/mogud/snow/core/stdext/constraints"
)

type Heap[T any] struct {
	data []T
	comp func(l, r T) bool
}

func NewHeap[T any](comp func(l, r T) bool) *Heap[T] {
	return &Heap[T]{comp: comp}
}

func NewHeapLess[T constraints.Ordered]() *Heap[T] {
	return &Heap[T]{comp: func(l, r T) bool {
		return l < r
	}}
}

func (heap *Heap[T]) Peek() (first T, ok bool) {
	if len(heap.data) > 0 {
		first = heap.data[0]
		ok = true
	}
	return
}

func (heap *Heap[T]) Push(v T) {
	heap.data = append(heap.data, v)
	heap.up(heap.Len() - 1)
}

func (heap *Heap[T]) Remove(i int) T {
	n := heap.Len() - 1
	if n != i {
		heap.swap(i, n)
		if !heap.down(i, n) {
			heap.up(i)
		}
	}
	result := heap.data[n]
	heap.data = heap.data[:n]
	return result
}

func (heap *Heap[T]) Data() []T {
	return heap.data
}

func (heap *Heap[T]) Pop() T {
	n := heap.Len() - 1
	if n > 0 {
		heap.swap(0, n)
		heap.down(0, n)
	}
	result := heap.data[n]
	heap.data = heap.data[:n]
	return result
}

func (heap *Heap[T]) Len() int {
	return len(heap.data)
}

func (heap *Heap[T]) Copy() *Heap[T] {
	return &Heap[T]{
		data: List[T](heap.data).Copy(),
		comp: heap.comp,
	}
}

func (heap *Heap[T]) swap(i, j int) {
	heap.data[i], heap.data[j] = heap.data[j], heap.data[i]
}

func (heap *Heap[T]) up(i int) {
	for {
		p := (i - 1) / 2 // parent
		if p == i || !heap.comp(heap.data[i], heap.data[p]) {
			break
		}
		heap.swap(p, i)
		i = p
	}
}

func (heap *Heap[T]) down(i0, n int) bool {
	i := i0
	for {
		left := (i * 2) + 1 // left
		if left >= n || left < 0 {
			break
		}
		j := left
		right := (i * 2) + 2 // right
		if right < n && heap.comp(heap.data[right], heap.data[left]) {
			j = right
		}
		if !heap.comp(heap.data[j], heap.data[i]) {
			break
		}
		heap.swap(i, j)
		i = j
	}
	return i > i0
}
