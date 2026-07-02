package cockroach4407

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type Stopper struct {
	stopper chan struct{}
	stop    sync.WaitGroup
	mu      sync.Mutex
}

func (s *Stopper) RunWorker(f func()) {
	sched.InstWgBF(914828034052)
	s.stop.Add(1)
	sched.InstWgAF(914828034052, &s.stop, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(914828034049, _parentGid)
		defer goroutine.Exit(914828034049)
		func() {
			defer func() {
				sched.InstWgBF(914828034053)
				s.stop.Done()
				sched.InstWgAF(914828034053, &s.stop, "done")
			}()
			f()
		}()
	}(goroutine.CurrentGid())
}

func (s *Stopper) SetStopped() {
	if s != nil {
		sched.InstWgBF(914828034054)
		s.stop.Done()
		sched.InstWgAF(914828034054, &s.stop, "done")
	}
}

func (s *Stopper) Stop() {
	close(s.stopper)
	s.stop.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
}

type server struct {
	mu      sync.Mutex
	stopper *Stopper
}

func (s *server) Gossip() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopper.RunWorker(func() {
		s.gossipSender()
	})
}

func (s *server) gossipSender() {
	s.mu.Lock()
	defer s.mu.Unlock()
}

func NewStopper() *Stopper {
	return &Stopper{
		stopper: make(chan struct{}),
	}
}

func TestCockroach4407(t *testing.T) {
	stopper := NewStopper()
	defer stopper.Stop()
	s := &server{
		stopper: stopper,
	}
	for i := 0; i < 2; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(914828034050, _parentGid)
			defer goroutine.Exit(914828034050)
			s.Gossip()
		}(goroutine.CurrentGid())
	}
	time.Sleep(time.Millisecond)
}
func TestCockroach4407_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	stopper := NewStopper()
	defer stopper.Stop()
	s := &server{
		stopper: stopper,
	}
	for i := 0; i < 2; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(914828034050, _parentGid)
			defer goroutine.Exit(914828034050)
			s.Gossip()
		}(goroutine.CurrentGid())
	}
	time.Sleep(time.Millisecond)
}
