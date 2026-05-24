package cockroach4407

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type Stopper struct {
	stopper chan struct{}
	stop    sync.WaitGroup
	mu      sync.Mutex
}

func (s *Stopper) RunWorker(f func()) {
	defer callstack.Trace(914828034049)()
	s.stop.Add(1)
	go func() {
		defer callstack.Trace(914828034050)()
		defer s.stop.Done()
		f()
	}()
}

func (s *Stopper) SetStopped() {
	defer callstack.Trace(914828034051)()
	if s != nil {
		s.stop.Done()
	}
}

func (s *Stopper) Stop() {
	defer callstack.Trace(914828034052)()
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
	defer callstack.Trace(914828034053)()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopper.RunWorker(func() {
		defer callstack.Trace(914828034054)()
		s.gossipSender()
	})
}

func (s *server) gossipSender() {
	defer callstack.Trace(914828034055)()
	s.mu.Lock()
	defer s.mu.Unlock()
}

func NewStopper() *Stopper {
	defer callstack.Trace(914828034056)()
	return &Stopper{
		stopper: make(chan struct{}),
	}
}

func TestCockroach4407(t *testing.T) {
	defer callstack.Trace(914828034057)()
	stopper := NewStopper()
	defer stopper.Stop()
	s := &server{
		stopper: stopper,
	}
	for i := 0; i < 2; i++ {
		go s.Gossip()
	}
	time.Sleep(time.Millisecond)
}
func TestCockroach4407_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(914828034057)()
	stopper := NewStopper()
	defer stopper.Stop()
	s := &server{
		stopper: stopper,
	}
	for i := 0; i < 2; i++ {
		go s.Gossip()
	}
	time.Sleep(time.Millisecond)
}
