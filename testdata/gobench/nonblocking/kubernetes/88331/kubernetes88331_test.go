package kubernetes88331

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type data struct {
	queue []struct{}
}

func (h *data) Pop() {
	h.queue = h.queue[0 : len(h.queue)-1]
}

type Interface interface {
	Pop()
}

func Pop(h Interface) {
	h.Pop()
}

type Heap struct {
	data *data
}

func (h *Heap) Pop() {
	Pop(h.data)
}
func (h *Heap) Len() int {
	return len(h.data.queue)
}

func NewWithRecorder() *Heap {
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
	p.lock.Lock()
	defer p.lock.Unlock()
	p.podBackoffQ.Pop()

}

func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		stop:        make(chan struct{}),
		activeQ:     NewWithRecorder(),
		podBackoffQ: NewWithRecorder(),
	}
}

func createAndRunPriorityQueue() *PriorityQueue {
	q := NewPriorityQueue()
	q.Run()
	return q
}

func (p *PriorityQueue) Run() {
	go func(_parentGid uint64) {
		goroutine.Enter(7055360803116941313, _parentGid)
		defer goroutine.Exit(7055360803116941313)
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

func TestKubernetes88331(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(7055360803116941315)
	wg.Add(1)
	sched.InstWgAF(7055360803116941315, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(7055360803116941314, _parentGid)
		defer goroutine.Exit(7055360803116941314)
		func() {
			sched.InstWgBF(7055360803116941316)
			wg.Done()
			sched.InstWgAF(7055360803116941316, &wg, "done")
			p := createAndRunPriorityQueue()
			p.podBackoffQ.Len()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestKubernetes88331_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(7055360803116941315)
	wg.Add(1)
	sched.InstWgAF(7055360803116941315, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(7055360803116941314, _parentGid)
		defer goroutine.Exit(7055360803116941314)
		func() {
			sched.InstWgBF(7055360803116941316)
			wg.Done()
			sched.InstWgAF(7055360803116941316, &wg, "done")
			p := createAndRunPriorityQueue()
			p.podBackoffQ.Len()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
