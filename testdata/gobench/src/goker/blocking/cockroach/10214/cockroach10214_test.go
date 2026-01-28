/*
 * Project: cockroach
 * Issue or PR  : https://github.com/cockroachdb/cockroach/pull/10214
 * Buggy version: 7207111aa3a43df0552509365fdec741a53f873f
 * fix commit-id: 27e863d90ab0660494778f1c35966cc5ddc38e32
 * Flaky: 3/100
 * Description: This deadlock is caused by different order when acquiring
 * coalescedMu.Lock() and raftMu.Lock(). The fix is to refactor sendQueuedHeartbeats()
 * so that cockroachdb can unlock coalescedMu before locking raftMu.
 */
package cockroach10214

import (
	sched "sched"
	"sync"
	"testing"
	"unsafe"
)

type Store struct {
	coalescedMu struct {
		sync.Mutex
		heartbeatResponses []int
	}
	mu struct {
		replicas map[int]*Replica
	}
}

func (s *Store) sendQueuedHeartbeats() {
	sched.InstMutexBF(545460846593, &s.coalescedMu)
	s.coalescedMu.Lock()
	sched.InstMutexAF( // LockA acquire
		545460846593, &s.coalescedMu)
	defer func() {
		sched.InstMutexBF(545460846594, &s.coalescedMu)
		s.coalescedMu.Unlock()
		sched. // LockA release
			InstMutexAF(545460846594, &s.coalescedMu)
	}()
	for i := 0; i < len(s.coalescedMu.heartbeatResponses); i++ {
		s.sendQueuedHeartbeatsToNode() // LockB
	}
}

func (s *Store) sendQueuedHeartbeatsToNode() {
	for i := 0; i < len(s.mu.replicas); i++ {
		r := s.mu.replicas[i]
		r.reportUnreachable() // LockB
	}
}

type Replica struct {
	raftMu sync.Mutex
	mu     sync.Mutex
	store  *Store
}

func (r *Replica) reportUnreachable() {
	sched.InstMutexBF(545460846595,
		// LockB acquire
		&r.raftMu)
	r.raftMu.Lock()
	sched.InstMutexAF(545460846595,
		//+time.Sleep(time.Nanosecond)
		&r.raftMu)
	defer func() {
		sched.InstMutexBF(545460846596, &r.raftMu)
		r.raftMu.Unlock()
		sched.
			// LockB release
			InstMutexAF(545460846596, &r.raftMu)
	}()

}

func (r *Replica) tick() {
	sched.InstMutexBF(545460846597,
		// LockB acquire
		&r.raftMu)
	r.raftMu.Lock()
	sched.InstMutexAF(545460846597, &r.raftMu)
	defer func() {
		sched.InstMutexBF(545460846598, &r.raftMu)
		r.raftMu.Unlock()
		sched.InstMutexAF(545460846598,

			// LockB release
			&r.raftMu)
	}()
	r.tickRaftMuLocked()

}

func (r *Replica) tickRaftMuLocked() {
	sched.InstMutexBF(545460846599, &r.mu)
	r.mu.Lock()
	sched.InstMutexAF(545460846599, &r.mu)
	defer func() {
		sched.InstMutexBF(545460846600, &r.mu)
		r.mu.Unlock()
		sched.InstMutexAF(545460846600, &r.mu)
	}()
	if r.maybeQuiesceLocked() {
		return
	}
}
func (r *Replica) maybeQuiesceLocked() bool {
	for i := 0; i < 2; i++ {
		if !r.maybeCoalesceHeartbeat() {
			return true
		}
	}
	return false
}
func (r *Replica) maybeCoalesceHeartbeat() bool {
	msgtype := uintptr(unsafe.Pointer(r)) % 3
	switch msgtype {
	case 0, 1, 2:
		sched.InstMutexBF(545460846601,
			// LockA acquire
			&r.store.coalescedMu)
		r.store.coalescedMu.Lock()
		sched.InstMutexAF(545460846601, &r.store.coalescedMu)
	default:
		return false
	}
	sched.InstMutexBF(545460846602, &r.store.coalescedMu)
	r.store.coalescedMu.Unlock()
	sched. // LockA release
		InstMutexAF(545460846602, &r.store.coalescedMu)
	return true
}

func TestCockroach10214(t *testing.T) {
	store := &Store{}
	responses := &store.coalescedMu.heartbeatResponses
	*responses = append(*responses, 1, 2)
	store.mu.replicas = make(map[int]*Replica)

	rp1 := &Replica{
		store: store,
	}
	rp2 := &Replica{
		store: store,
	}
	store.mu.replicas[0] = rp1
	store.mu.replicas[1] = rp2

	go func() {
		store.sendQueuedHeartbeats()
	}()

	go func() {
		rp1.tick()
	}()
}
func TestCockroach10214_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	store := &Store{}
	responses := &store.coalescedMu.heartbeatResponses
	*responses = append(*responses, 1, 2)
	store.mu.replicas = make(map[int]*Replica)

	rp1 := &Replica{
		store: store,
	}
	rp2 := &Replica{
		store: store,
	}
	store.mu.replicas[0] = rp1
	store.mu.replicas[1] = rp2

	go func() {
		store.sendQueuedHeartbeats()
	}()

	go func() {
		rp1.tick()
	}()
}
