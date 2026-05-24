package istio8967

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type Source interface {
	Start()
	Stop()
}

type fsSource struct {
	donec chan struct{}
}

func (s *fsSource) Start() {
	defer callstack.Trace(360777252865)()
	go func() {
		defer callstack.Trace(360777252866)()
		for {
			select {
			case <-s.donec:
				return
			}
		}
	}()
}

func (s *fsSource) Stop() {
	defer callstack.Trace(360777252867)()
	close(s.donec)
	s.donec = nil
}

func newFsSource() *fsSource {
	defer callstack.Trace(360777252868)()
	return &fsSource{
		donec: make(chan struct{}),
	}
}

func New() Source {
	defer callstack.Trace(360777252869)()
	return newFsSource()
}

func TestIstio8967(t *testing.T) {
	defer callstack.Trace(360777252870)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(360777252871)()
		defer wg.Done()
		s := New()
		s.Start()
		s.Stop()
		time.Sleep(5 * time.Millisecond)
	}()
	wg.Wait()
}
func TestIstio8967_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(360777252870)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(360777252871)()
		defer wg.Done()
		s := New()
		s.Start()
		s.Stop()
		time.Sleep(5 * time.Millisecond)
	}()
	wg.Wait()
}
