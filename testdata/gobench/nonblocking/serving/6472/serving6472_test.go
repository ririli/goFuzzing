package serving6472

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
)

type workItem struct {
	ingressState *ingressState
}

type ingressState struct {
	pendingCount int32
}

type t interface{}

type Interface interface {
	Get() interface{}
	Add(item interface{})
}

type Type struct {
	queue []t
	cond  *sync.Cond
}

func (q *Type) Get() (item interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if len(q.queue) == 0 {
		return nil
	}

	item, q.queue = q.queue[0], q.queue[1:]
	return item
}

func (q *Type) Add(item interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.queue = append(q.queue, item)
}

type DelayingInterface interface {
	Interface
	AddAfter(item interface{})
}

type delayingType struct {
	Interface
}

func (q *delayingType) AddAfter(item interface{}) {
	sched.InstWgBF(17458846813321691140)
	q.Add(item)
	sched.InstWgAF(17458846813321691140, &q, "add")
}

func newDelayingQueue() DelayingInterface {
	return &delayingType{&Type{queue: []t{}, cond: sync.NewCond(&sync.Mutex{})}}
}

func NewDelayingQueue() DelayingInterface {
	return newDelayingQueue()
}

type RateLimitingInterface interface {
	DelayingInterface
	AddRateLimited(item interface{})
}

type rateLimitingType struct {
	DelayingInterface
}

func (q *rateLimitingType) AddRateLimited(item interface{}) {
	q.DelayingInterface.AddAfter(item)
}

func NewRateLimitingQueue() RateLimitingInterface {
	return &rateLimitingType{
		DelayingInterface: NewDelayingQueue(),
	}
}

type Prober struct {
	workQueue RateLimitingInterface
}

func (m *Prober) IsReady() {
	workItems := make(map[string][]*workItem)
	ingressState := &ingressState{}
	workItems["0"] = append(workItems["0"], &workItem{
		ingressState: ingressState,
	})
	for _, ipWorkItems := range workItems {
		/*
			go func() {
				m.updateStates(ingressState)
			}()
		*/
		for _, wi := range ipWorkItems {
			sched.InstWgBF(17458846813321691141)
			m.workQueue.Add(wi)
			sched.InstWgAF(17458846813321691141, &m.workQueue, "add")
		}
	}
	ingressState.pendingCount += int32(len(workItems))
}

func (m *Prober) processWorkItem() {
	obj := m.workQueue.Get()
	item, ok := obj.(*workItem)
	if !ok {
		return
	}
	m.updateStates(item.ingressState)
}

func (m *Prober) updateStates(ingressState *ingressState) {
	if atomic.AddInt32(&ingressState.pendingCount, -1) == 0 {
	}
}

func (m *Prober) Start() chan struct{} {
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		sched.InstWgBF(17458846813321691142)
		wg.Add(1)
		sched.InstWgAF(17458846813321691142, &wg, "add")
		go func(_parentGid uint64) {
			goroutine.Enter(17458846813321691137, _parentGid)
			defer goroutine.Exit(17458846813321691137)
			func() {
				defer func() {
					sched.InstWgBF(17458846813321691143)
					wg.Done()
					sched.InstWgAF(17458846813321691143, &wg, "done")
				}()
				m.processWorkItem()
			}()
		}(goroutine.CurrentGid())
	}
	ch := make(chan struct{})
	go func(_parentGid uint64) {
		goroutine.Enter(17458846813321691138, _parentGid)
		defer goroutine.Exit(17458846813321691138)
		func() {
			wg.Wait()
			sched.InstChBF(17458846813321691139)
			close(ch)
			sched.InstChAF(17458846813321691139, ch, "close")
		}()
	}(goroutine.CurrentGid())
	return ch
}

func NewProber() *Prober {
	workQueue := NewRateLimitingQueue()
	workQueue.Add(&workItem{&ingressState{}})
	return &Prober{
		workQueue: workQueue,
	}
}

func TestServing6472(t *testing.T) {
	prober := NewProber()
	done := make(chan struct{})
	cancelled := prober.Start()
	defer func() {
		close(done)
		<-cancelled
	}()

	prober.IsReady()
	time.Sleep(1 * time.Millisecond)
}
func TestServing6472_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	prober := NewProber()
	done := make(chan struct{})
	cancelled := prober.Start()
	defer func() {
		close(done)
		<-cancelled
	}()

	prober.IsReady()
	time.Sleep(1 * time.Millisecond)
}
