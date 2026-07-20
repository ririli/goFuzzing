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
		goroutine.Enter(18210754054793986049, _parentGid)
		defer goroutine.Exit(18210754054793986049)
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
	sched.InstWgBF(18210754054793986052)
	wg.Add(1)
	sched.InstWgAF(18210754054793986052, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(18210754054793986050, _parentGid)
		defer goroutine.Exit(18210754054793986050)
		func() {
			defer func() {
				sched.InstWgBF(18210754054793986053)
				wg.Done()
				sched.InstWgAF(18210754054793986053, &wg, "done")
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
	sched.InstWgBF(18210754054793986052)
	wg.Add(1)
	sched.InstWgAF(18210754054793986052, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(18210754054793986050, _parentGid)
		defer goroutine.Exit(18210754054793986050)
		func() {
			defer func() {
				sched.InstWgBF(18210754054793986053)
				wg.Done()
				sched.InstWgAF(18210754054793986053, &wg, "done")
			}()
			s := New()
			s.Start()
			s.Stop()
			time.Sleep(5 * time.Millisecond)
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
