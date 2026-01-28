package cockroach24808

import (
	"context"
	sched "sched"
	"sync"
	"testing"
)

type Compactor struct {
	ch chan struct{}
}

type Stopper struct {
	stop    sync.WaitGroup
	stopper chan struct{}
}

func (s *Stopper) RunWorker(ctx context.Context, f func(context.Context)) {
	s.stop.Add(1)
	go func() {
		defer s.stop.Done()
		f(ctx)
	}()
}

func (s *Stopper) ShouldStop() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.stopper
}

func (s *Stopper) Stop() {
	close(s.stopper)
}

func NewStopper() *Stopper {
	s := &Stopper{
		stopper: make(chan struct{}),
	}
	return s
}

func NewCompactor() *Compactor {
	return &Compactor{ch: make(chan struct{}, 1)}
}

func (c *Compactor) Start(ctx context.Context, stopper *Stopper) {
	sched.InstChBF(365072220162, c.ch)
	c.ch <- struct{}{}
	sched.InstChAF(365072220162, c.ch)
	stopper.RunWorker(ctx, func(ctx context.Context) {
		for {
			select {
			case <-stopper.ShouldStop():
				sched.InstChAF(365072220166, stopper.ShouldStop())
				return
			case <-c.ch:
				sched.InstChAF(365072220167, c.ch)
			}
		}
	})
}

func TestCockroach24808(t *testing.T) {
	stopper := NewStopper()
	defer stopper.Stop()

	compactor := NewCompactor()
	sched.InstChBF(365072220165, compactor.ch)
	compactor.ch <- struct{}{}
	sched.InstChAF(365072220165, compactor.ch)

	compactor.Start(context.Background(), stopper)
}
func TestCockroach24808_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	stopper := NewStopper()
	defer stopper.Stop()

	compactor := NewCompactor()
	sched.InstChBF(365072220165, compactor.ch)
	compactor.ch <- struct{}{}
	sched.InstChAF(365072220165, compactor.ch)

	compactor.Start(context.Background(), stopper)
}
