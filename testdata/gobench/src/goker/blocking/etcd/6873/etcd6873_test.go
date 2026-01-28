/*
 * Project: etcd
 * Issue or PR  : https://github.com/etcd-io/etcd/commit/7618fdd1d642e47cac70c03f637b0fd798a53a6e
 * Buggy version: 377f19b0031f9c0aafe2aec28b6f9019311f52f9
 * fix commit-id: 7618fdd1d642e47cac70c03f637b0fd798a53a6e
 * Flaky: 9/100
 */
package etcd6873

import (
	sched "sched"
	"sync"
	"testing"
)

type watchBroadcast struct{}

type watchBroadcasts struct {
	mu      sync.Mutex
	updatec chan *watchBroadcast
	donec   chan struct{}
}

func newWatchBroadcasts() *watchBroadcasts {
	wbs := &watchBroadcasts{
		updatec: make(chan *watchBroadcast, 1),
		donec:   make(chan struct{}),
	}
	go func() { // G2
		defer close(wbs.donec)
		for wb := range wbs.updatec {
			wbs.coalesce(wb)
		}
	}()
	return wbs
}

func (wbs *watchBroadcasts) coalesce(wb *watchBroadcast) {
	sched.InstMutexBF(953482739718, &wbs.mu)
	wbs.mu.Lock()
	sched.InstMutexAF(953482739718, &wbs.mu)
	sched.InstMutexBF(953482739719, &wbs.mu)
	wbs.mu.Unlock()
	sched.InstMutexAF(953482739719, &wbs.mu)
}

func (wbs *watchBroadcasts) stop() {
	sched.InstMutexBF(953482739720, &wbs.mu)
	wbs.mu.Lock()
	sched.InstMutexAF(953482739720, &wbs.mu)
	defer func() {
		sched.InstMutexBF(953482739721, &wbs.mu)
		wbs.mu.Unlock()
		sched.InstMutexAF(953482739721, &wbs.mu)
	}()
	close(wbs.updatec)
	sched.InstChBF(953482739715, wbs.donec)
	<-wbs.donec
	sched.InstChAF(953482739715, wbs.donec)
}

func (wbs *watchBroadcasts) update(wb *watchBroadcast) {
	select {
	case wbs.updatec <- wb:
		sched.InstChAF(953482739717, wbs.updatec)
	default:
	}
}

// /
// / G1						G2					G3
// / newWatchBroadcasts()
// /	wbs.update()
// / wbs.updatec <-
// / return
// /							<-wbs.updatec
// /							wbs.coalesce()
// /												wbs.stop()
// /												wbs.mu.Lock()
// /												close(wbs.updatec)
// /												<-wbs.donec
// /							wbs.mu.Lock()
// /---------------------G2,G3 deadlock-------------------------
// /
func TestEtcd(t *testing.T) {
	wbs := newWatchBroadcasts() // G1
	wbs.update(&watchBroadcast{})
	go wbs.stop() // G3
}
func TestEtcd_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	wbs := newWatchBroadcasts()
	wbs.update(&watchBroadcast{})
	go wbs.stop()
}
