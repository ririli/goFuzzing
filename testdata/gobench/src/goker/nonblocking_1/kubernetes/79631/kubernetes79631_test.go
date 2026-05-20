package kubernetes79631

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type heapData struct {
	items map[string]struct{}
}

func (h *heapData) Pop() {
	defer callstack.Trace(163208757249)()
	delete(h.items, "1")
}

type Interface interface {
	Pop()
}

func Pop(h Interface) {
	defer callstack.Trace(163208757250)()
	h.Pop()
}

type Heap struct {
	data *heapData
}

func (h *Heap) Pop() {
	defer callstack.Trace(163208757251)()
	Pop(h.data)
}

func (h *Heap) Get() {
	defer callstack.Trace(163208757252)()
	h.GetByKey()
}

func (h *Heap) GetByKey() {
	defer callstack.Trace(163208757253)()
	_ = h.data.items["1"]
}

func NewWithRecorder() *Heap {
	defer callstack.Trace(163208757254)()
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
	defer callstack.Trace(163208757255)()
	p.lock.Lock()
	defer p.lock.Unlock()
	p.podBackoffQ.Pop()

}

func NewPriorityQueue() *PriorityQueue {
	defer callstack.Trace(163208757256)()
	return NewPriorityQueueWithClock()
}

func NewPriorityQueueWithClock() *PriorityQueue {
	defer callstack.Trace(163208757257)()
	pg := &PriorityQueue{
		stop:        make(chan struct{}),
		podBackoffQ: NewWithRecorder(),
	}
	pg.run()
	return pg
}

func (p *PriorityQueue) run() {
	defer callstack.Trace(163208757258)()
	go Until(p.flushBackoffQCompleted, p.stop)
}

func BackoffUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(163208757259)()
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		func() {
			defer callstack.Trace(163208757260)()
			f()
		}()

		select {
		case <-stopCh:
			return
		}
	}
}

func JitterUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(163208757261)()
	BackoffUntil(f, stopCh)
}

func Until(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(163208757262)()
	JitterUntil(f, stopCh)
}

func TestKubernetes79631(t *testing.T) {
	defer callstack.Trace(163208757263)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(163208757264)()
		wg.Done()
		q := NewPriorityQueue()
		q.podBackoffQ.Get()
	}()
	wg.Wait()
}
func TestKubernetes79631_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(163208757263)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(163208757264)()
		wg.Done()
		q := NewPriorityQueue()
		q.podBackoffQ.Get()
	}()
	wg.Wait()
}
