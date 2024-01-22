package syncext_test

import (
	"gitee.com/mogud/snow/core/syncext"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestWait(t *testing.T) {
	a := atomic.Int32{}
	wg := syncext.NewTimeoutWaitGroup()
	wg.Add(1)
	go func() {
		a.Store(5)
		wg.Done()
	}()
	wg.Wait()
	assert.Equal(t, int32(5), a.Load())
}

func TestWaitTimeout(t *testing.T) {
	a := atomic.Int32{}
	wg1 := syncext.NewTimeoutWaitGroup()
	wg1.Add(1)
	go func() {
		time.Sleep(5 * time.Millisecond)
		a.Store(5)
		wg1.Done()
	}()
	assert.False(t, wg1.WaitTimeout(time.Millisecond))
	assert.NotEqual(t, int32(5), a.Load())

	wg2 := syncext.NewTimeoutWaitGroup()
	wg2.Add(1)
	go func() {
		time.Sleep(5 * time.Millisecond)
		a.Store(10)
		wg2.Done()
	}()
	assert.True(t, wg2.WaitTimeout(10*time.Millisecond))
	assert.Equal(t, int32(10), a.Load())
}
