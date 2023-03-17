package climit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestConcurrencyLimit(t *testing.T) {
	cl := New("test", "test", 2, nil)
	event := make(chan struct{})

	var count atomic.Int32

	var t1, t2, t4, t8 *Token
	go func() {
		t1 = cl.Acquire()
		count.Add(1)
		event <- struct{}{}
		t2 = cl.Acquire()
		count.Add(2)
		event <- struct{}{}
		t4 = cl.Acquire()
		count.Add(4)
		event <- struct{}{}
		t8 = cl.Acquire()
		count.Add(8)
		event <- struct{}{}
	}()

	<-event
	<-event
	assert.Equal(t, int32(3), count.Load())
	time.Sleep(10 * time.Millisecond)
	select {
	case <-event:
		t.Fatal("unexpected event")
	default:
		// OK
	}

	// Release a token
	t2.Release()
	<-event
	assert.Equal(t, int32(7), count.Load())

	// Release the same again, nothing happens
	t2.Release()
	time.Sleep(10 * time.Millisecond)
	select {
	case <-event:
		t.Fatal("unexpected event")
	default:
		// OK
	}
	assert.Equal(t, int32(7), count.Load())

	// Release another for the last increment
	t1.Release()
	<-event
	assert.Equal(t, int32(15), count.Load())

	t4.Release()
	t8.Release()
}
