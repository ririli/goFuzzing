package serving5865

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type revisionWatcher struct {
	destsCh chan struct{}
}

func (rw *revisionWatcher) run() {
	defer close(rw.destsCh)
}

type revisionBackendsManager struct {
	revisionWatchersMux sync.RWMutex
}

func newRevisionWatcher(destsCh chan struct{}) *revisionWatcher {
	return &revisionWatcher{destsCh: destsCh}
}

func (rbm *revisionBackendsManager) endpointsUpdated() {
	rw := rbm.getOrCreateRevisionWatcher()
	sched.InstChBF(403726925827)
	rw.destsCh <- struct{}{}
	sched.InstChAF(403726925827, rw.destsCh, "send")
}

func (rbm *revisionBackendsManager) getOrCreateRevisionWatcher() *revisionWatcher {
	rbm.revisionWatchersMux.Lock()
	defer rbm.revisionWatchersMux.Unlock()

	destsCh := make(chan struct{})
	rw := newRevisionWatcher(destsCh)
	go func(_parentGid uint64) {
		goroutine.Enter(403726925825, _parentGid)
		defer goroutine.Exit(403726925825)
		rw.run()
	}(goroutine.CurrentGid())

	return rw
}

func newRevisionBackendsManagerWithProbeFrequency() *revisionBackendsManager {
	rbm := &revisionBackendsManager{}
	return rbm
}

func TestServing5865(t *testing.T) {
	rbm := newRevisionBackendsManagerWithProbeFrequency()

	// Simplified code in the RealTestSuite
	func() {
		rbm.endpointsUpdated()
	}()
}
func TestServing5865_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	rbm := newRevisionBackendsManagerWithProbeFrequency()

	func() {
		rbm.endpointsUpdated()
	}()
}
