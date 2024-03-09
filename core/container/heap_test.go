package container_test

import (
	"github.com/mogud/snow/core/container"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeap(t *testing.T) {
	// 小顶堆
	intHeap := container.NewHeapLess[int]()

	intArr := []int{1, 3, 4, 2}
	for _, it := range intArr {
		num := it
		intHeap.Push(num)
	}

	len := intHeap.Len()
	assert.True(t, len == 4, "expect heap len = 4 but get %d", len)
	num, _ := intHeap.Peek()
	assert.True(t, num == 1, "expect peek num 1 but get %d", num)
	num = intHeap.Pop()
	assert.True(t, num == 1, "expect pop num 1 but get %d", num)

	len = intHeap.Len()
	assert.True(t, len == 3, "expect heap len = 3 but get %d", len)
	num, _ = intHeap.Peek()
	assert.True(t, num == 2, "expect peek num 2 but get %d", num)
	num = intHeap.Pop()
	assert.True(t, num == 2, "expect pop num 2 but get %d", num)

	len = intHeap.Len()
	assert.True(t, len == 2, "expect heap len = 2 but get %d", len)
	num, _ = intHeap.Peek()
	assert.True(t, num == 3, "expect peek num 3 but get %d", num)
	num = intHeap.Pop()
	assert.True(t, num == 3, "expect pop num 3 but get %d", num)

	len = intHeap.Len()
	assert.True(t, len == 1, "expect heap len = 1 but get %d", len)
	num, _ = intHeap.Peek()
	assert.True(t, num == 4, "expect peek num 4 but get %d", num)
	num = intHeap.Pop()
	assert.True(t, num == 4, "expect pop num 4 but get %d", num)

	len = intHeap.Len()
	assert.True(t, len == 0, "expect heap len =0 but get %d", len)

	// 小顶堆
	intSmallTopHeap := container.NewHeap(func(l, r int) bool {
		return l < r
	})
	// 大顶堆
	intBigTopHeap := container.NewHeap(func(l, r int) bool {
		return l > r
	})

	for _, it := range intArr {
		num := it
		intSmallTopHeap.Push(num)
		intBigTopHeap.Push(num)
	}

	// assert.True(t, intSmallTopHeap.Peek() == 1, "int small top heap expect peek num 1 but get %d", num)
	// assert.True(t, intBigTopHeap.Peek() == 4, "int big top heap expect peek num 4 but get %d", num)
}
