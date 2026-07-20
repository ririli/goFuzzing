package etcd8194

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

var leaseRevokeRate = 1000

func testLessorRenewExtendPileup() {
	oldRevokeRate := leaseRevokeRate
	defer func() { leaseRevokeRate = oldRevokeRate }()
	leaseRevokeRate = 10
}

type Lease struct{}

type lessor struct {
	mu    sync.Mutex
	stopC chan struct{}
	doneC chan struct{}
}

func (le *lessor) runLoop() {
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
	l := &lessor{}
	go func(_parentGid uint64) {
		goroutine.Enter(13138048183844208641, _parentGid)
		defer goroutine.Exit(13138048183844208641)
		l.runLoop()
	}(goroutine.CurrentGid())
	return l
}

func testLessorGrant() {
	newLessor()
}

func TestEtcd8194(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(13138048183844208645)
	wg.Add(2)
	sched.InstWgAF(13138048183844208645, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(13138048183844208642, _parentGid)
		defer goroutine.Exit(13138048183844208642)
		func() {
			defer func() {
				sched.InstWgBF(13138048183844208646)
				wg.Done()
				sched.InstWgAF(13138048183844208646, &wg, "done")
			}()
			testLessorGrant()
		}()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(13138048183844208643, _parentGid)
		defer goroutine.Exit(13138048183844208643)
		func() {
			defer func() {
				sched.InstWgBF(13138048183844208647)
				wg.Done()
				sched.InstWgAF(13138048183844208647, &wg, "done")
			}()
			testLessorRenewExtendPileup()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestEtcd8194_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(13138048183844208645)
	wg.Add(2)
	sched.InstWgAF(13138048183844208645, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(13138048183844208642, _parentGid)
		defer goroutine.Exit(13138048183844208642)
		func() {
			defer func() {
				sched.InstWgBF(13138048183844208646)
				wg.Done()
				sched.InstWgAF(13138048183844208646, &wg, "done")
			}()
			testLessorGrant()
		}()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(13138048183844208643, _parentGid)
		defer goroutine.Exit(13138048183844208643)
		func() {
			defer func() {
				sched.InstWgBF(13138048183844208647)
				wg.Done()
				sched.InstWgAF(13138048183844208647, &wg, "done")
			}()
			testLessorRenewExtendPileup()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
