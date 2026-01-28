package cockroach584

import (
	sched "sched"
	"sync"
	"testing"
)

type Gossip struct {
	mu     sync.Mutex
	closed bool
}

func (g *Gossip) bootstrap() {
	for {
		sched.InstMutexBF(700079669249, &g.mu)
		g.mu.Lock()
		sched.InstMutexAF(700079669249,

			/// Missing g.mu.Unlock
			&g.mu)
		if g.closed {

			break
		}
		sched.InstMutexBF(700079669250, &g.mu)
		g.mu.Unlock()
		sched.InstMutexAF(700079669250, &g.mu)
		break
	}
}

func (g *Gossip) manage() {
	for {
		sched.InstMutexBF(700079669251, &g.mu)
		g.mu.Lock()
		sched.InstMutexAF(700079669251,

			/// Missing g.mu.Unlock
			&g.mu)
		if g.closed {

			break
		}
		sched.InstMutexBF(700079669252, &g.mu)
		g.mu.Unlock()
		sched.InstMutexAF(700079669252, &g.mu)
		break
	}
}
func TestCockroach584(t *testing.T) {
	g := &Gossip{
		closed: true,
	}
	go func() {
		g.bootstrap()
		g.manage()
	}()
}
func TestCockroach584_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	g := &Gossip{
		closed: true,
	}
	go func() {
		g.bootstrap()
		g.manage()
	}()
}
