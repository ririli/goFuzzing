package etcd10492

import (
	"context"
	sched "sched"
	"sync"
	"testing"
	"time"
)

type Checkpointer func(ctx context.Context)

type lessor struct {
	mu                 sync.RWMutex
	cp                 Checkpointer
	checkpointInterval time.Duration
}

func (le *lessor) Checkpoint() {
	sched.InstMutexBF(
		// block here
		721554505729, &le.mu)
	le.mu.Lock()
	sched.InstMutexAF(721554505729, &le.mu)
	defer func() {
		sched.InstMutexBF(721554505730, &le.mu)
		le.mu.Unlock()
		sched.InstMutexAF(721554505730, &le.mu)
	}()
}

func (le *lessor) SetCheckpointer(cp Checkpointer) {
	sched.InstMutexBF(721554505731, &le.mu)
	le.mu.Lock()
	sched.InstMutexAF(721554505731, &le.mu)
	defer func() {
		sched.InstMutexBF(721554505732, &le.mu)
		le.mu.Unlock()
		sched.InstMutexAF(721554505732, &le.mu)
	}()

	le.cp = cp
}

func (le *lessor) Renew() {
	sched.InstMutexBF(721554505733, &le.mu)
	le.mu.Lock()
	sched.InstMutexAF(721554505733, &le.mu)
	unlock := func() {
		sched.InstMutexBF(721554505734, &le.mu)
		le.mu.Unlock()
		sched.InstMutexAF(721554505734, &le.mu)
	}
	defer func() { unlock() }()

	if le.cp != nil {
		le.cp(context.Background())
	}
}
func TestEtcd10492(t *testing.T) {
	le := &lessor{
		checkpointInterval: 0,
	}
	fakerCheckerpointer := func(ctx context.Context) {
		le.Checkpoint()
	}
	le.SetCheckpointer(fakerCheckerpointer)
	sched.InstMutexBF(721554505735, &le.mu)
	le.mu.Lock()
	sched.InstMutexAF(721554505735, &le.mu)
	sched.InstMutexBF(721554505736, &le.mu)
	le.mu.Unlock()
	sched.InstMutexAF(721554505736, &le.mu)
	le.Renew()
}
func TestEtcd10492_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	le := &lessor{
		checkpointInterval: 0,
	}
	fakerCheckerpointer := func(ctx context.Context) {
		le.Checkpoint()
	}
	le.SetCheckpointer(fakerCheckerpointer)
	sched.InstMutexBF(721554505735, &le.mu)
	le.mu.Lock()
	sched.InstMutexAF(721554505735, &le.mu)
	sched.InstMutexBF(721554505736, &le.mu)
	le.mu.Unlock()
	sched.InstMutexAF(721554505736, &le.mu)
	le.Renew()
}
