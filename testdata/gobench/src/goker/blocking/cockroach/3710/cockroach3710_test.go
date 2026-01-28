/*
 * Project: cockroach
 * Issue or PR  : https://github.com/cockroachdb/cockroach/pull/3710
 * Buggy version: 4afdd4860fd7c3bd9e92489f84a95e5cc7d11a0d
 * fix commit-id: cb65190f9caaf464723e7d072b1f1b69a044ef7b
 * Flaky: 2/100
 * Description: This deadlock is casued by acquiring a RLock twice in a call chain.
 * ForceRaftLogScanAndProcess(acquire s.mu.RLock()) ->MaybeAdd()->shouldQueue()->
 * getTruncatableIndexes()->RaftStatus(acquire s.mu.Rlock())
 */

package cockroach3710

import (
	sched "sched"
	"sync"
	"testing"
	"unsafe"
)

type Store struct {
	raftLogQueue *baseQueue
	replicas     map[int]*Replica

	mu struct {
		sync.RWMutex
	}
}

func (s *Store) ForceRaftLogScanAndProcess() {
	sched.InstMutexBF(21474836481, &s.mu)
	s.mu.RLock()
	sched.InstMutexAF(21474836481, &s.mu)
	for _, r := range s.replicas {
		s.raftLogQueue.MaybeAdd(r)
	}
	sched.InstMutexBF(21474836482, &s.mu)
	s.mu.RUnlock()
	sched.InstMutexAF(21474836482, &s.mu)
}

func (s *Store) RaftStatus() {
	sched.InstMutexBF(21474836483, &s.mu)
	s.mu.RLock()
	sched.InstMutexAF(21474836483, &s.mu)
	defer func() {
		sched.InstMutexBF(21474836484, &s.mu)
		s.mu.RUnlock()
		sched.InstMutexAF(21474836484, &s.mu)
	}()
}

func (s *Store) processRaft() {
	go func() {
		for {
			var replicas []*Replica
			sched.InstMutexBF(21474836485, &s.mu)
			s.mu.Lock()
			sched.InstMutexAF(21474836485, &s.mu)
			for _, r := range s.replicas {
				replicas = append(replicas, r)
			}
			sched.InstMutexBF(21474836486, &s.mu)
			s.mu.Unlock()
			sched.InstMutexAF(21474836486, &s.mu)
			break
		}
	}()
}

type Replica struct {
	store *Store
}

type baseQueue struct {
	sync.Mutex
	impl *raftLogQueue
}

func (bq *baseQueue) MaybeAdd(repl *Replica) {
	sched.InstMutexBF(21474836487, &bq)
	bq.Lock()
	sched.InstMutexAF(21474836487, &bq)
	defer func() {
		sched.InstMutexBF(21474836488, &bq)
		bq.Unlock()
		sched.InstMutexAF(21474836488, &bq)
	}()
	bq.impl.shouldQueue(repl)
}

type raftLogQueue struct{}

func (*raftLogQueue) shouldQueue(r *Replica) {
	getTruncatableIndexes(r)
}

func getTruncatableIndexes(r *Replica) {
	r.store.RaftStatus()
}

func NewStore() *Store {
	rlq := &raftLogQueue{}
	bq := &baseQueue{impl: rlq}
	store := &Store{
		raftLogQueue: bq,
		replicas:     make(map[int]*Replica),
	}
	r1 := &Replica{store}
	r2 := &Replica{store}

	makeKey := func(r *Replica) int {
		return int((uintptr(unsafe.Pointer(r)) >> 1) % 7)
	}
	store.replicas[makeKey(r1)] = r1
	store.replicas[makeKey(r2)] = r2

	return store
}

// / G1 										G2
// / store.ForceRaftLogScanAndProcess()
// / s.mu.RLock()
// / s.raftLogQueue.MaybeAdd()
// / bq.impl.shouldQueue()
// / getTruncatableIndexes()
// / r.store.RaftStatus()
// / 										store.processRaft()
// / 										s.mu.Lock()
// / s.mu.RLock()
// / ----------------------G1,G2 deadlock---------------------
func TestCockroach3710(t *testing.T) {
	store := NewStore()
	go store.ForceRaftLogScanAndProcess() // G1
	go store.processRaft()                // G2
}
func TestCockroach3710_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	store := NewStore()
	go store.ForceRaftLogScanAndProcess()
	go store.processRaft()
}
