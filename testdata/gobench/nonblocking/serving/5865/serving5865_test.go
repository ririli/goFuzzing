package serving5865

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type revisionWatcher struct {
	destsCh chan struct{}
}

func (rw *revisionWatcher) run() {
	defer callstack.Trace(403726925825)()
	defer close(rw.destsCh)
}

type revisionBackendsManager struct {
	revisionWatchersMux sync.RWMutex
}

func newRevisionWatcher(destsCh chan struct{}) *revisionWatcher {
	defer callstack.Trace(403726925826)()
	return &revisionWatcher{destsCh: destsCh}
}

func (rbm *revisionBackendsManager) endpointsUpdated() {
	defer callstack.Trace(403726925827)()
	rw := rbm.getOrCreateRevisionWatcher()
	rw.destsCh <- struct{}{}
}

func (rbm *revisionBackendsManager) getOrCreateRevisionWatcher() *revisionWatcher {
	defer callstack.Trace(403726925828)()
	rbm.revisionWatchersMux.Lock()
	defer rbm.revisionWatchersMux.Unlock()

	destsCh := make(chan struct{})
	rw := newRevisionWatcher(destsCh)
	go rw.run()

	return rw
}

func newRevisionBackendsManagerWithProbeFrequency() *revisionBackendsManager {
	defer callstack.Trace(403726925829)()
	rbm := &revisionBackendsManager{}
	return rbm
}

func TestServing5865(t *testing.T) {
	defer callstack.Trace(403726925830)()
	rbm := newRevisionBackendsManagerWithProbeFrequency()

	// Simplified code in the RealTestSuite
	func() {
		defer callstack.Trace(403726925831)()
		rbm.endpointsUpdated()
	}()
}
func TestServing5865_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(403726925830)()
	rbm := newRevisionBackendsManagerWithProbeFrequency()

	func() {
		defer callstack.Trace(403726925831)()
		rbm.endpointsUpdated()
	}()
}
