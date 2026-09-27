package breakpoint

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestBarrierDuplicateSideCannotCover(t *testing.T) {
	b := NewBarrierConfig()
	b.Timeout = time.Second
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)
	gate := b.barriers[1][0]
	done := make(chan struct{})
	go func() { b.PointControl(1); close(done) }()
	deadline := time.Now().Add(time.Second)
	for atomic.LoadInt32(&gate.arrived) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("first side never arrived")
		}
		time.Sleep(time.Millisecond)
	}
	b.PointControl(1)
	select {
	case <-gate.release:
		t.Fatal("same side falsely covered barrier")
	default:
	}
	b.PointControl(2)
	<-done
	if atomic.LoadInt32(&gate.arrived) != 3 {
		t.Fatal("distinct sides failed to cover")
	}
}

func TestBarrierTimeoutIsTerminal(t *testing.T) {
	b := NewBarrierConfig()
	b.Timeout = time.Millisecond
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)
	b.PointControl(1)
	b.PointControl(2)
	if atomic.LoadInt32(&b.barriers[1][0].arrived) != 4 {
		t.Fatal("late arrival revived timed-out gate")
	}
}
