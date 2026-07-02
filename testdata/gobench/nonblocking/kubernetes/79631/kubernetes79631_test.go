package kubernetes79631

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type heapData struct {
	items map[string]struct{}
}

func (h *heapData) Pop() {
	delete(h.items, "1")
}

type Interface interface {
	Pop()
}

func Pop(h Interface) {
	h.Pop()
}

type Heap struct {
	data *heapData
}

func (h *Heap) Pop() {
	Pop(h.data)
}

func (h *Heap) Get() {
	h.GetByKey()
}

func (h *Heap) GetByKey() {
	_ = h.data.items["1"]
}

func NewWithRecorder() *Heap {
	return &Heap{
		data: &heapData{
			items: make(map[string]struct{}),
		},
	}
}

type PriorityQueue struct {
	stop        chan struct{}
	lock        sync.RWMutex
	podBackoffQ *Heap
}

func (p *PriorityQueue) flushBackoffQCompleted() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.podBackoffQ.Pop()

}

func NewPriorityQueue() *PriorityQueue {
	return NewPriorityQueueWithClock()
}

func NewPriorityQueueWithClock() *PriorityQueue {
	pg := &PriorityQueue{
		stop:        make(chan struct{}),
		podBackoffQ: NewWithRecorder(),
	}
	pg.run()
	return pg
}

func (p *PriorityQueue) run() {
	go func(_parentGid uint64) {
		goroutine.Enter(455266533377, _parentGid)
		defer goroutine.Exit(455266533377)
		Until(p.flushBackoffQCompleted, p.stop)
	}(goroutine.CurrentGid())
}

func BackoffUntil(f func(), stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		func() {
			f()
		}()

		select {
		case <-stopCh:
			return
		}
	}
}

func JitterUntil(f func(), stopCh <-chan struct{}) {
	BackoffUntil(f, stopCh)
}

func Until(f func(), stopCh <-chan struct{}) {
	JitterUntil(f, stopCh)
}

func TestKubernetes79631(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(455266533379)
	wg.Add(1)
	sched.InstWgAF(455266533379, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(455266533378, _parentGid)
		defer goroutine.Exit(455266533378)
		func() {
			sched.InstWgBF(455266533380)
			wg.Done()
			sched.InstWgAF(455266533380, &wg, "done")
			q := NewPriorityQueue()
			q.podBackoffQ.Get()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestKubernetes79631_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(455266533379)
	wg.Add(1)
	sched.InstWgAF(455266533379, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(455266533378, _parentGid)
		defer goroutine.Exit(455266533378)
		func() {
			sched.InstWgBF(455266533380)
			wg.Done()
			sched.InstWgAF(455266533380, &wg, "done")
			q := NewPriorityQueue()
			q.podBackoffQ.Get()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
