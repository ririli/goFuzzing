package istio8967

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type Source interface {
	Start()
	Stop()
}

type fsSource struct {
	donec chan struct{}
}

func (s *fsSource) Start() {
	go func(_parentGid uint64) {
		goroutine.Enter(360777252865, _parentGid)
		defer goroutine.Exit(360777252865)
		func() {
			for {
				select {
				case <-s.donec:
					return
				}
			}
		}()
	}(goroutine.CurrentGid())
}

func (s *fsSource) Stop() {
	close(s.donec)
	s.donec = nil
}

func newFsSource() *fsSource {
	return &fsSource{
		donec: make(chan struct{}),
	}
}

func New() Source {
	return newFsSource()
}

func TestIstio8967(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(360777252868)
	wg.Add(1)
	sched.InstWgAF(360777252868, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(360777252866, _parentGid)
		defer goroutine.Exit(360777252866)
		func() {
			defer func() {
				sched.InstWgBF(360777252869)
				wg.Done()
				sched.InstWgAF(360777252869, &wg, "done")
			}()
			s := New()
			s.Start()
			s.Stop()
			time.Sleep(5 * time.Millisecond)
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestIstio8967_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(360777252868)
	wg.Add(1)
	sched.InstWgAF(360777252868, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(360777252866, _parentGid)
		defer goroutine.Exit(360777252866)
		func() {
			defer func() {
				sched.InstWgBF(360777252869)
				wg.Done()
				sched.InstWgAF(360777252869, &wg, "done")
			}()
			s := New()
			s.Start()
			s.Stop()
			time.Sleep(5 * time.Millisecond)
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
