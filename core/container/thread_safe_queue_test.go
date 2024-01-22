// Copyright 2022 all contributors.
// SPDX-License-Identifier: Apache-2.0

package container_test

import (
	"github.com/stretchr/testify/assert"
	"snow/core/container"
	"sync"
	"testing"
)

func TestSafeQueue(t *testing.T) {
	intQueue := container.NewThreadSafeQueue[int]()
	intArr := []int{1, 2, 3, 4}
	for _, it := range intArr {
		num := it
		intQueue.Enq(num)
	}

	for !intQueue.Empty() {
		tt := intQueue.Deq()
		t.Log(tt)
	}

	for _, it := range intArr {
		num := it
		intQueue.Enq(num)
	}

	num := intQueue.Deq()
	assert.True(t, num == 1, "expect 1 but get %d ", num)

	num = intQueue.Deq()
	assert.True(t, num == 2, "expect 2 but get %d ", num)

	num = intQueue.Deq()
	assert.True(t, num == 3, "expect 3 but get %d ", num)

	num = intQueue.Deq()
	assert.True(t, num == 4, "expect 4 but get %d ", num)

	for _, it := range intArr {
		num := it
		intQueue.Enq(num)
	}

	sum := 0
	for !intQueue.Empty() {
		sum += intQueue.Deq()
	}
	assert.True(t, sum == 10, "sum expect 10 but get %d ", sum)

	ch := make(chan int, 1)
	finishCH := make(chan bool, 1)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		<-finishCH
		sum := 0

		for !intQueue.Empty() {
			sum += intQueue.Deq()
		}
		ch <- sum
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		sum = <-ch
		if sum != 10 {
			assert.True(t, sum == 10, "sum expect 10 but get %d ", sum)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for _, it := range intArr {
			num := it
			intQueue.Enq(num)
		}
		finishCH <- true
		wg.Done()
	}()
	wg.Wait()
}
