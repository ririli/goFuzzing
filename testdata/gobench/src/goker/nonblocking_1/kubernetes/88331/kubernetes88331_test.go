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
	defer callstack.Trace(889058230273)()
	h.queue = h.queue[0 : len(h.queue)-1]
}

type Interface interface {
	Pop()
}

func Pop(h Interface) {
	defer callstack.Trace(889058230274)()
	h.Pop()
}

type Heap struct {
	data *data
}

func (h *Heap) Pop() {
	defer callstack.Trace(889058230275)()
	Pop(h.data)
}
func (h *Heap) Len() int {
	defer callstack.Trace(889058230276)()
	return len(h.data.queue)
}

func NewWithRecorder() *Heap {
	defer callstack.Trace(889058230277)()
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
	defer callstack.Trace(889058230278)()
	p.lock.Lock()
	defer p.lock.Unlock()
	p.podBackoffQ.Pop()

}

func NewPriorityQueue() *PriorityQueue {
	defer callstack.Trace(889058230279)()
	return &PriorityQueue{
		stop:        make(chan struct{}),
		activeQ:     NewWithRecorder(),
		podBackoffQ: NewWithRecorder(),
	}
}

func createAndRunPriorityQueue() *PriorityQueue {
	defer callstack.Trace(889058230280)()
	q := NewPriorityQueue()
	q.Run()
	return q
}

func (p *PriorityQueue) Run() {
	defer callstack.Trace(889058230281)()
	go Until(p.flushBackoffQCompleted, p.stop)
}

func BackoffUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(889058230282)()
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		func() {
			defer callstack.Trace(889058230283)()
			f()
		}()

		select {
		case <-stopCh:
			return
		}
	}
}

func JitterUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(889058230284)()
	BackoffUntil(f, stopCh)
}

func Until(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(889058230285)()
	JitterUntil(f, stopCh)
}

func TestKubernetes88331(t *testing.T) {
	defer callstack.Trace(889058230286)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(889058230287)()
		wg.Done()
		p := createAndRunPriorityQueue()
		p.podBackoffQ.Len()
	}()
	wg.Wait()
}
func TestKubernetes88331_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(889058230286)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(889058230287)()
		wg.Done()
		p := createAndRunPriorityQueue()
		p.podBackoffQ.Len()
	}()
	wg.Wait()
}
