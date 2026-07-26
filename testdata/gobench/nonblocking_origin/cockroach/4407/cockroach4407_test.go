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
	sched.InstWgBF(6301501851195932676)
	s.stop.Add(1)
	sched.InstWgAF(6301501851195932676, &s.stop, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(6301501851195932673, _parentGid)
		defer goroutine.Exit(6301501851195932673)
		func() {
			defer func() {
				sched.InstWgBF(6301501851195932677)
				s.stop.Done()
				sched.InstWgAF(6301501851195932677, &s.stop, "done")
			}()
			f()
		}()
	}(goroutine.CurrentGid())
}

func (s *Stopper) SetStopped() {
	if s != nil {
		sched.InstWgBF(6301501851195932678)
		s.stop.Done()
		sched.InstWgAF(6301501851195932678, &s.stop, "done")
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
			goroutine.Enter(6301501851195932674, _parentGid)
			defer goroutine.Exit(6301501851195932674)
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
			goroutine.Enter(6301501851195932674, _parentGid)
			defer goroutine.Exit(6301501851195932674)
			s.Gossip()
		}(goroutine.CurrentGid())
	}
	time.Sleep(time.Millisecond)
}
