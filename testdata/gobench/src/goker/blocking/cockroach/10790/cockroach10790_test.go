/*
 * Project: cockroach
 * Issue or PR  : https://github.com/cockroachdb/cockroach/pull/10790
 * Buggy version: 96b5452557ebe26bd9d85fe7905155009204d893
 * fix commit-id: f1a5c19125c65129b966fbdc0e6408e8df214aba
 * Flaky: 28/100
 * Description:
 *   It is possible that a message from ctxDone will make the function beginCmds
 * returns without draining the channel ch, so that goroutines created by anonymous
 * function will leak.
 */

package cockroach10790

import (
	"context"
	sched "sched"
	"sync"
	"testing"
)

type Stopper struct {
	quiescer chan struct{}
	mu       struct {
		sync.Mutex
		quiescing bool
	}
}

func (s *Stopper) ShouldQuiesce() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.quiescer
}

func (s *Stopper) Quiesce() {
	sched.InstMutexBF(1026497183755, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(1026497183755, &s.mu)
	defer func() {
		sched.InstMutexBF(1026497183756, &s.mu)
		s.mu.Unlock()
		sched.InstMutexAF(1026497183756, &s.mu)
	}()
	if !s.mu.quiescing {
		s.mu.quiescing = true
		close(s.quiescer)
	}
}

func (s *Stopper) Stop() {
	s.Quiesce()
}

type Replica struct {
	chans   []chan bool
	stopper *Stopper
}

func (r *Replica) beginCmds(ctx context.Context) {
	ctxDone := ctx.Done()
	for _, ch := range r.chans {
		select {
		case <-ch:
			sched.InstChAF(1026497183751, ch)
		case <-ctxDone:
			sched.InstChAF(1026497183752, ctxDone)
			go func() {
				for _, ch := range r.chans {
					sched.InstChBF(1026497183748, ch)
					<-ch
					sched.InstChAF(1026497183748, ch)
				}
			}()
		}
	}
}

// / helper goroutine, not present in the real bug.
func (r *Replica) sendChans(ctx context.Context) {
	for _, ch := range r.chans {
		select {
		case ch <- true:
			sched.InstChAF(1026497183753, ch)
		case <-ctx.Done():
			sched.InstChAF(1026497183754, ctx.Done())
			return
		}
	}
}

func NewReplica() *Replica {
	r := &Replica{
		stopper: &Stopper{
			quiescer: make(chan struct{}),
		},
	}
	r.chans = append(r.chans, make(chan bool))
	r.chans = append(r.chans, make(chan bool))
	return r
}

// /
// / G1					G2				helper goroutine
// / 									r.sendChans()
// / r.beginCmds()
// / 									ch1 <- true
// / <- ch1
// /										ch2 <- true
// /	...					...				...
// /						cancel()
// /	<- ch1
// /	------------------G1 leak--------------------------
// /
func TestCockroach10790(t *testing.T) {
	r := NewReplica()
	ctx, cancel := context.WithCancel(context.Background())
	go r.sendChans(ctx) // helper goroutine
	go r.beginCmds(ctx) // G1
	go cancel()         // G2
	r.stopper.Stop()
}
func TestCockroach10790_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	r := NewReplica()
	ctx, cancel := context.WithCancel(context.Background())
	go r.sendChans(ctx)
	go r.beginCmds(ctx)
	go cancel()
	r.stopper.Stop()
}
