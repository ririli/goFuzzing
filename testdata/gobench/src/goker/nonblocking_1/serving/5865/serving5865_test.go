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
	defer callstack.Trace(588410519553)()
	defer close(rw.destsCh)
}

type revisionBackendsManager struct {
	revisionWatchersMux sync.RWMutex
}

func newRevisionWatcher(destsCh chan struct{}) *revisionWatcher {
	defer callstack.Trace(588410519554)()
	return &revisionWatcher{destsCh: destsCh}
}

func (rbm *revisionBackendsManager) endpointsUpdated() {
	defer callstack.Trace(588410519555)()
	rw := rbm.getOrCreateRevisionWatcher()
	rw.destsCh <- struct{}{}
}

func (rbm *revisionBackendsManager) getOrCreateRevisionWatcher() *revisionWatcher {
	defer callstack.Trace(588410519556)()
	rbm.revisionWatchersMux.Lock()
	defer rbm.revisionWatchersMux.Unlock()

	destsCh := make(chan struct{})
	rw := newRevisionWatcher(destsCh)
	go rw.run()

	return rw
}

func newRevisionBackendsManagerWithProbeFrequency() *revisionBackendsManager {
	defer callstack.Trace(588410519557)()
	rbm := &revisionBackendsManager{}
	return rbm
}

func TestServing5865(t *testing.T) {
	defer callstack.Trace(588410519558)()
	rbm := newRevisionBackendsManagerWithProbeFrequency()

	// Simplified code in the RealTestSuite
	func() {
		defer callstack.Trace(588410519559)()
		rbm.endpointsUpdated()
	}()
}
func TestServing5865_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(588410519558)()
	rbm := newRevisionBackendsManagerWithProbeFrequency()

	func() {
		defer callstack.Trace(588410519559)()
		rbm.endpointsUpdated()
	}()
}
