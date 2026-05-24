package kubernetes81148

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

const unschedulableQTimeInterval = 60 * time.Second

type Pod string

type PodInfo struct {
	Pod       Pod
	Timestamp time.Time
}

type UnschedulablePodsMap struct {
	podInfoMap map[string]*PodInfo
	keyFunc    func(Pod) string
}

func (u *UnschedulablePodsMap) addOrUpdate(pInfo *PodInfo) {
	defer callstack.Trace(459561500673)()
	podID := u.keyFunc(pInfo.Pod)
	u.podInfoMap[podID] = pInfo
}

func GetPodFullName(pod Pod) string {
	defer callstack.Trace(459561500674)()
	return string(pod)
}

func newUnschedulablePodsMap() *UnschedulablePodsMap {
	defer callstack.Trace(459561500675)()
	return &UnschedulablePodsMap{
		podInfoMap: make(map[string]*PodInfo),
		keyFunc:    GetPodFullName,
	}
}

type PriorityQueue struct {
	stop           <-chan struct{}
	lock           sync.RWMutex
	unschedulableQ *UnschedulablePodsMap
}

func (p *PriorityQueue) flushUnschedulableQLeftover() {
	defer callstack.Trace(459561500676)()
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, pInfo := range p.unschedulableQ.podInfoMap {
		_ = pInfo.Timestamp
	}
}

func (p *PriorityQueue) run() {
	defer callstack.Trace(459561500677)()
	go Until(p.flushUnschedulableQLeftover, p.stop)
}

func (p *PriorityQueue) newPodInfo(pod Pod) *PodInfo {
	defer callstack.Trace(459561500678)()
	return &PodInfo{
		Pod:       pod,
		Timestamp: time.Now(),
	}
}

func NewPriorityQueueWithClock(stop <-chan struct{}) *PriorityQueue {
	defer callstack.Trace(459561500679)()
	pq := &PriorityQueue{
		stop:           stop,
		unschedulableQ: newUnschedulablePodsMap(),
	}
	pq.run()
	return pq
}

func NewPriorityQueue(stop <-chan struct{}) *PriorityQueue {
	defer callstack.Trace(459561500680)()
	return NewPriorityQueueWithClock(stop)
}

func BackoffUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(459561500681)()
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		func() {
			defer callstack.Trace(459561500682)()
			f()
		}()

		select {
		case <-stopCh:
			return
		}
	}
}

func JitterUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(459561500683)()
	BackoffUntil(f, stopCh)
}

func Until(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(459561500684)()
	JitterUntil(f, stopCh)
}

func addOrUpdateUnschedulablePod(p *PriorityQueue, pod Pod) {
	defer callstack.Trace(459561500685)()
	p.lock.Lock()
	defer p.lock.Unlock()
	p.unschedulableQ.addOrUpdate(p.newPodInfo(pod))
}

func TestKubernetes81148(t *testing.T) {
	defer callstack.Trace(459561500686)()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(459561500687)()
		defer wg.Done()
		q := NewPriorityQueue(stop)
		highPod := Pod("1")
		addOrUpdateUnschedulablePod(q, highPod)
		q.unschedulableQ.podInfoMap[GetPodFullName(highPod)].Timestamp = time.Now().Add(-1 * unschedulableQTimeInterval)
	}()
	wg.Wait()
	close(stop)
}
func TestKubernetes81148_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(459561500686)()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(459561500687)()
		defer wg.Done()
		q := NewPriorityQueue(stop)
		highPod := Pod("1")
		addOrUpdateUnschedulablePod(q, highPod)
		q.unschedulableQ.podInfoMap[GetPodFullName(highPod)].Timestamp = time.Now().Add(-1 * unschedulableQTimeInterval)
	}()
	wg.Wait()
	close(stop)
}
