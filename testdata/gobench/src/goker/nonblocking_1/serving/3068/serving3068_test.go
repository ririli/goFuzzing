package serving3068

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type Interface interface {
	Go(func())
	Wait()
}

type impl struct {
	wg     sync.WaitGroup
	workCh chan func()
	once   sync.Once
}

var _ Interface = (*impl)(nil)

func NewWithCapacity(workers, capacity int) Interface {
	defer callstack.Trace(133143986177)()
	i := &impl{
		workCh: make(chan func(), capacity),
	}

	for idx := 0; idx < workers; idx++ {
		go func() {
			defer callstack.Trace(133143986178)()
			for work := range i.workCh {
				func() {
					defer callstack.Trace(133143986179)()
					defer i.wg.Done()
					work()
				}()
			}
		}()
	}

	return i
}

func (i *impl) Go(w func()) {
	defer callstack.Trace(133143986180)()
	i.wg.Add(1)
	i.workCh <- w
}

func (i *impl) Wait() {
	defer callstack.Trace(133143986181)()
	i.once.Do(func() {
		defer callstack.Trace(133143986182)()
		close(i.workCh)

		go func() {
			defer callstack.Trace(133143986183)()
			i.wg.Wait()
		}()
	})
}

func TestServing3068(t *testing.T) {
	defer callstack.Trace(133143986184)()
	p := NewWithCapacity(1, 5)
	wg := &sync.WaitGroup{}
	var cntExecuted int32
	const n = 5
	wg.Add(n)
	go func() {
		defer callstack.Trace(133143986185)()
		for i := 0; i < n; i++ {
			p.Go(func() {
				defer callstack.Trace(133143986186)()
				atomic.AddInt32(&cntExecuted, 1)
			})
			time.Sleep(10 * time.Millisecond)
			wg.Done()
		}
	}()
	p.Wait()
	wg.Wait()
	if cntExecuted == n {
		t.Error("Not all items were expected to execute")
	}
}
func TestServing3068_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(133143986184)()
	p := NewWithCapacity(1, 5)
	wg := &sync.WaitGroup{}
	var cntExecuted int32
	const n = 5
	wg.Add(n)
	go func() {
		defer callstack.Trace(133143986185)()
		for i := 0; i < n; i++ {
			p.Go(func() {
				defer callstack.Trace(133143986186)()
				atomic.AddInt32(&cntExecuted, 1)
			})
			time.Sleep(10 * time.Millisecond)
			wg.Done()
		}
	}()
	p.Wait()
	wg.Wait()
	if cntExecuted == n {
		t.Error("Not all items were expected to execute")
	}
}
