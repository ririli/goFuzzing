package cockroach1055

import (
	sched "sched"
	"sync"
	"sync/atomic"
	"testing"
)

type Stopper struct {
	stopper  chan struct{}
	stop     sync.WaitGroup
	mu       sync.Mutex
	draining int32
	drain    sync.WaitGroup
}

func (s *Stopper) AddWorker() {
	s.stop.Add(1)
}

func (s *Stopper) ShouldStop() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.stopper
}

func (s *Stopper) SetStopped() {
	if s != nil {
		s.stop.Done()
	}
}

func (s *Stopper) Quiesce() {
	sched.InstMutexBF(1005022347269, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(1005022347269, &s.mu)
	defer func() {
		sched.InstMutexBF(1005022347270, &s.mu)
		s.mu.Unlock()
		sched.InstMutexAF(1005022347270, &s.mu)
	}()
	s.draining = 1
	s.drain.Wait()
	s.draining = 0
}

func (s *Stopper) Stop() {
	sched.InstMutexBF(1005022347271, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(1005022347271, &s.mu)
	defer func() {
		sched.InstMutexBF(1005022347272, &s.mu)
		s.mu.Unlock()
		sched.InstMutexAF(1005022347272, &s.mu)
	}()
	atomic.StoreInt32(&s.draining, 1)
	s.drain.Wait()
	close(s.stopper)
	s.stop.Wait()
}

func (s *Stopper) StartTask() bool {
	if atomic.LoadInt32(&s.draining) == 0 {
		sched.InstMutexBF(1005022347273, &s.mu)
		s.mu.Lock()
		sched.InstMutexAF(1005022347273, &s.mu)
		defer func() {
			sched.InstMutexBF(1005022347274, &s.mu)
			s.mu.Unlock()
			sched.InstMutexAF(1005022347274, &s.mu)
		}()
		s.drain.Add(1)
		return true
	}
	return false
}

func NewStopper() *Stopper {
	return &Stopper{
		stopper: make(chan struct{}),
	}
}

func TestCockroach1055(t *testing.T) {
	var stoppers []*Stopper
	for i := 0; i < 3; i++ {
		stoppers = append(stoppers, NewStopper())
	}

	for i := range stoppers {
		s := stoppers[i]
		s.AddWorker()
		go func() {
			s.StartTask()
			sched.InstChBF(1005022347266, s.ShouldStop())
			<-s.ShouldStop()
			sched.InstChAF(1005022347266, s.ShouldStop())
			s.SetStopped()
		}()
	}

	done := make(chan struct{})
	go func() {
		for _, s := range stoppers {
			s.Quiesce()
		}
		for _, s := range stoppers {
			s.Stop()
		}
		sched.InstChBF(1005022347267, done)
		close(done)
		sched.InstChAF(1005022347267, done)
	}()
	sched.InstChBF(1005022347268, done)
	<-done
	sched.InstChAF(1005022347268, done)
}
func TestCockroach1055_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	var stoppers []*Stopper
	for i := 0; i < 3; i++ {
		stoppers = append(stoppers, NewStopper())
	}

	for i := range stoppers {
		s := stoppers[i]
		s.AddWorker()
		go func() {
			s.StartTask()
			sched.InstChBF(1005022347266, s.ShouldStop())
			<-s.ShouldStop()
			sched.InstChAF(1005022347266, s.ShouldStop())
			s.SetStopped()
		}()
	}

	done := make(chan struct{})
	go func() {
		for _, s := range stoppers {
			s.Quiesce()
		}
		for _, s := range stoppers {
			s.Stop()
		}
		sched.InstChBF(1005022347267, done)
		close(done)
		sched.InstChAF(1005022347267, done)
	}()
	sched.InstChBF(1005022347268, done)
	<-done
	sched.InstChAF(1005022347268, done)
}
