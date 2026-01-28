package cockroach25456

import (
	sched "sched"
	"testing"
)

type Stopper struct {
	quiescer chan struct{}
}

func (s *Stopper) ShouldQuiesce() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.quiescer
}

func NewStopper() *Stopper {
	return &Stopper{quiescer: make(chan struct{})}
}

type Store struct {
	stopper          *Stopper
	consistencyQueue *consistencyQueue
}

func (s *Store) Stopper() *Stopper {
	return s.stopper
}
func (s *Store) Start(stopper *Stopper) {
	s.stopper = stopper
}

func NewStore() *Store {
	return &Store{
		consistencyQueue: newConsistencyQueue(),
	}
}

type Replica struct {
	store *Store
}

func NewReplica(store *Store) *Replica {
	return &Replica{store: store}
}

type consistencyQueue struct{}

func (q *consistencyQueue) process(repl *Replica) {
	sched.InstChBF(987842478081, repl.store.Stopper().ShouldQuiesce())
	<-repl.store.Stopper().ShouldQuiesce()
	sched.InstChAF(987842478081, repl.store.Stopper().ShouldQuiesce())
}

func newConsistencyQueue() *consistencyQueue {
	return &consistencyQueue{}
}

type testContext struct {
	store *Store
	repl  *Replica
}

func (tc *testContext) StartWithStoreConfig(stopper *Stopper) {
	if tc.store == nil {
		tc.store = NewStore()
	}
	tc.store.Start(stopper)
	tc.repl = NewReplica(tc.store)
}

func TestCockroach25456(t *testing.T) {
	stopper := NewStopper()
	tc := testContext{}
	tc.StartWithStoreConfig(stopper)

	for i := 0; i < 2; i++ {
		tc.store.consistencyQueue.process(tc.repl)
	}
}
func TestCockroach25456_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	stopper := NewStopper()
	tc := testContext{}
	tc.StartWithStoreConfig(stopper)

	for i := 0; i < 2; i++ {
		tc.store.consistencyQueue.process(tc.repl)
	}
}
