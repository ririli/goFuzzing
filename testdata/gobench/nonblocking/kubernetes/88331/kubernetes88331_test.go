package kubernetes88331

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type data struct {
	queue []struct{}
}

func (h *data) Pop() {
	defer callstack.Trace(296352743425)()
	h.queue = h.queue[0 : len(h.queue)-1]
}

type Interface interface {
	Pop()
}

func Pop(h Interface) {
	defer callstack.Trace(296352743426)()
	h.Pop()
}

type Heap struct {
	data *data
}

func (h *Heap) Pop() {
	defer callstack.Trace(296352743427)()
	Pop(h.data)
}
func (h *Heap) Len() int {
	defer callstack.Trace(296352743428)()
	return len(h.data.queue)
}

func NewWithRecorder() *Heap {
	defer callstack.Trace(296352743429)()
	return &Heap{
		data: &data{
			queue: []struct{}{
				struct{}{},
				struct{}{},
			},
		},
	}
}

type PriorityQueue struct {
	stop        chan struct{}
	lock        sync.RWMutex
	podBackoffQ *Heap
	activeQ     *Heap
}

func (p *PriorityQueue) flushBackoffQCompleted() {
	defer callstack.Trace(296352743430)()
	p.lock.Lock()
	defer p.lock.Unlock()
	p.podBackoffQ.Pop()

}

func NewPriorityQueue() *PriorityQueue {
	defer callstack.Trace(296352743431)()
	return &PriorityQueue{
		stop:        make(chan struct{}),
		activeQ:     NewWithRecorder(),
		podBackoffQ: NewWithRecorder(),
	}
}

func createAndRunPriorityQueue() *PriorityQueue {
	defer callstack.Trace(296352743432)()
	q := NewPriorityQueue()
	q.Run()
	return q
}

func (p *PriorityQueue) Run() {
	defer callstack.Trace(296352743433)()
	go Until(p.flushBackoffQCompleted, p.stop)
}

func BackoffUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(296352743434)()
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		func() {
			defer callstack.Trace(296352743435)()
			f()
		}()

		select {
		case <-stopCh:
			return
		}
	}
}

func JitterUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(296352743436)()
	BackoffUntil(f, stopCh)
}

func Until(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(296352743437)()
	JitterUntil(f, stopCh)
}

func TestKubernetes88331(t *testing.T) {
	defer callstack.Trace(296352743438)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(296352743439)()
		wg.Done()
		p := createAndRunPriorityQueue()
		p.podBackoffQ.Len()
	}()
	wg.Wait()
}
func TestKubernetes88331_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(296352743438)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(296352743439)()
		wg.Done()
		p := createAndRunPriorityQueue()
		p.podBackoffQ.Len()
	}()
	wg.Wait()
}
