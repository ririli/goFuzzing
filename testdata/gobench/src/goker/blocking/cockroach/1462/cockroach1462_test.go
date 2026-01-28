package cockroach1462

import (
	sched "sched"
	"sync"
	"testing"
)

type Stopper struct {
	stopper  chan struct{}
	stopped  chan struct{}
	stop     sync.WaitGroup
	mu       sync.Mutex
	drain    *sync.Cond
	draining bool
	numTasks int
}

func NewStopper() *Stopper {
	s := &Stopper{
		stopper: make(chan struct{}),
		stopped: make(chan struct{}),
	}
	s.drain = sync.NewCond(&s.mu)
	return s
}

func (s *Stopper) RunWorker(f func()) {
	s.AddWorker()
	go func() {
		defer s.SetStopped()
		f()
	}()
}

func (s *Stopper) AddWorker() {
	s.stop.Add(1)
}
func (s *Stopper) StartTask() bool {
	sched.InstMutexBF(627065225225, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(627065225225, &s.mu)
	defer func() {
		sched.InstMutexBF(627065225226, &s.mu)
		s.mu.Unlock()
		sched.InstMutexAF(627065225226, &s.mu)
	}()
	if s.draining {
		return false
	}
	s.numTasks++
	return true
}

func (s *Stopper) FinishTask() {
	sched.InstMutexBF(627065225227, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(627065225227, &s.mu)
	defer func() {
		sched.InstMutexBF(627065225228, &s.mu)
		s.mu.Unlock()
		sched.InstMutexAF(627065225228, &s.mu)
	}()
	s.numTasks--
	s.drain.Broadcast()
}
func (s *Stopper) SetStopped() {
	if s != nil {
		s.stop.Done()
	}
}
func (s *Stopper) ShouldStop() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.stopper
}

func (s *Stopper) Quiesce() {
	sched.InstMutexBF(627065225229, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(627065225229, &s.mu)
	defer func() {
		sched.InstMutexBF(627065225230, &s.mu)
		s.mu.Unlock()
		sched.InstMutexAF(627065225230, &s.mu)
	}()
	s.draining = true
	for s.numTasks > 0 {
		// Unlock s.mu, wait for the signal, and lock s.mu.
		s.drain.Wait()
	}
}

func (s *Stopper) Stop() {
	s.Quiesce()
	close(s.stopper)
	s.stop.Wait()
	sched.InstMutexBF(627065225231, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(627065225231, &s.mu)
	defer func() {
		sched.InstMutexBF(627065225232, &s.mu)
		s.mu.Unlock()
		sched.InstMutexAF(627065225232, &s.mu)
	}()
	close(s.stopped)
}

type interceptMessage int

type localInterceptableTransport struct {
	mu      sync.Mutex
	Events  chan interceptMessage
	stopper *Stopper
}

func (lt *localInterceptableTransport) Close() {}

type Transport interface {
	Close()
}

func NewLocalInterceptableTransport(stopper *Stopper) Transport {
	lt := &localInterceptableTransport{
		Events:  make(chan interceptMessage),
		stopper: stopper,
	}
	lt.start()
	return lt
}

func (lt *localInterceptableTransport) start() {
	lt.stopper.RunWorker(func() {
		for {
			select {
			case <-lt.stopper.ShouldStop():
				sched.InstChAF(627065225222, lt.stopper.ShouldStop())
				return
			default:
				sched.InstChBF(627065225220, lt.Events)
				lt.Events <- interceptMessage(0)
				sched.InstChAF(627065225220, lt.Events)
			}
		}
	})
}

func processEventsUntil(ch <-chan interceptMessage, stopper *Stopper) {
	for {
		select {
		case _, ok := <-ch:
			sched.InstChAF(627065225223, ch)
			if !ok {
				return
			}
		case <-stopper.ShouldStop():
			sched.InstChAF(627065225224, stopper.ShouldStop())
			return
		}
	}
}
func TestCockroach1462(t *testing.T) {
	stopper := NewStopper()
	transport := NewLocalInterceptableTransport(stopper).(*localInterceptableTransport)
	stopper.RunWorker(func() {
		processEventsUntil(transport.Events, stopper)
	})
	stopper.Stop()
}
func TestCockroach1462_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	stopper := NewStopper()
	transport := NewLocalInterceptableTransport(stopper).(*localInterceptableTransport)
	stopper.RunWorker(func() {
		processEventsUntil(transport.Events, stopper)
	})
	stopper.Stop()
}
