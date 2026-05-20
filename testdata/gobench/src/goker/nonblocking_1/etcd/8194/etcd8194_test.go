package etcd8194

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

var leaseRevokeRate = 1000

func testLessorRenewExtendPileup() {
	defer callstack.Trace(734439407617)()
	oldRevokeRate := leaseRevokeRate
	defer func() { defer callstack.Trace(734439407618)(); leaseRevokeRate = oldRevokeRate }()
	leaseRevokeRate = 10
}

type Lease struct{}

type lessor struct {
	mu    sync.Mutex
	stopC chan struct{}
	doneC chan struct{}
}

func (le *lessor) runLoop() {
	defer callstack.Trace(734439407619)()
	defer close(le.doneC)

	for i := 0; i < 10; i++ {
		var ls []*Lease

		ls = append(ls, &Lease{})

		if len(ls) != 0 {
			// rate limit
			if len(ls) > leaseRevokeRate/2 {
				ls = ls[:leaseRevokeRate/2]
			}
			select {
			case <-le.stopC:
				return
			default:
			}
		}

		select {
		case <-time.After(5 * time.Millisecond):
		case <-le.stopC:
			return
		}
	}
}

func newLessor() *lessor {
	defer callstack.Trace(734439407620)()
	l := &lessor{}
	go l.runLoop()
	return l
}

func testLessorGrant() {
	defer callstack.Trace(734439407621)()
	newLessor()
}

func TestEtcd8194(t *testing.T) {
	defer callstack.Trace(734439407622)()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer callstack.Trace(734439407623)()
		defer wg.Done()
		testLessorGrant()
	}()
	go func() {
		defer callstack.Trace(734439407624)()
		defer wg.Done()
		testLessorRenewExtendPileup()
	}()
	wg.Wait()
}
func TestEtcd8194_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(734439407622)()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer callstack.Trace(734439407623)()
		defer wg.Done()
		testLessorGrant()
	}()
	go func() {
		defer callstack.Trace(734439407624)()
		defer wg.Done()
		testLessorRenewExtendPileup()
	}()
	wg.Wait()
}
