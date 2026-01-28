package istio18454

import (
	"context"
	sched "sched"

	"sync"
	"testing"
	"time"
)

type Worker struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (w *Worker) Start(setupFn func(), runFn func(c context.Context)) {
	if setupFn != nil {
		setupFn()
	}
	go func() {
		runFn(w.ctx)
	}()
}

func (w *Worker) Stop() {
	w.ctxCancel()
}

type Strategy struct {
	timer          *time.Timer
	timerFrequency time.Duration
	stateLock      sync.Mutex
	resetChan      chan struct{}
	worker         *Worker
	startTimerFn   func()
}

func (s *Strategy) OnChange() {
	sched.InstMutexBF(743029342222, &s.stateLock)
	s.stateLock.Lock()
	sched.InstMutexAF(743029342222, &s.stateLock)
	if s.timer != nil {
		sched.InstMutexBF(743029342223, &s.stateLock)
		s.stateLock.Unlock()
		sched.InstMutexAF(743029342223, &s.stateLock)
		sched.InstChBF(743029342209, s.resetChan)
		s.resetChan <- struct{}{}
		sched.InstChAF(743029342209, s.resetChan)
		return
	}
	s.startTimerFn()
	sched.InstMutexBF(743029342224, &s.stateLock)
	s.stateLock.Unlock()
	sched.InstMutexAF(743029342224, &s.stateLock)
}

func (s *Strategy) startTimer() {
	s.timer = time.NewTimer(s.timerFrequency)
	eventLoop := func(ctx context.Context) {
		for {
			select {
			case <-s.timer.C:
				sched.InstChAF(743029342217, s.timer.C)
			case <-s.resetChan:
				sched.InstChAF(743029342218, s.resetChan)
				if !s.timer.Stop() {
					sched.InstChBF(743029342212, s.timer.C)
					<-s.timer.C
					sched.InstChAF(743029342212, s.timer.C)
				}
				s.timer.Reset(s.timerFrequency)
			case <-ctx.Done():
				sched.InstChAF(743029342219, ctx.Done())
				s.timer.Stop()
				return
			}
		}
	}
	s.worker.Start(nil, eventLoop)
}

func (s *Strategy) Close() {
	s.worker.Stop()
}

type Event int

type Processor struct {
	stateStrategy *Strategy
	worker        *Worker
	eventCh       chan Event
}

func (p *Processor) processEvent() {
	p.stateStrategy.OnChange()
}

func (p *Processor) Start() {
	setupFn := func() {
		for i := 0; i < 1024; i++ {
			sched.InstChBF(743029342214, p.eventCh)
			p.eventCh <- Event(0)
			sched.InstChAF(743029342214, p.eventCh)
		}
	}
	runFn := func(ctx context.Context) {
		defer func() {
			p.stateStrategy.Close()
		}()
		for {
			select {
			case <-ctx.Done():
				sched.InstChAF(743029342220, ctx.Done())
				return
			case <-p.eventCh:
				sched.InstChAF(743029342221, p.eventCh)
				p.processEvent()
			}
		}
	}
	p.worker.Start(setupFn, runFn)
}

func (p *Processor) Stop() {
	p.worker.Stop()
}

func NewWorker() *Worker {
	worker := &Worker{}
	worker.ctx, worker.ctxCancel = context.WithCancel(context.Background())
	return worker
}

func TestIstio18454(t *testing.T) {
	stateStrategy := &Strategy{
		timerFrequency: time.Nanosecond,
		resetChan:      make(chan struct{}, 1),
		worker:         NewWorker(),
	}
	stateStrategy.startTimerFn = stateStrategy.startTimer

	p := &Processor{
		stateStrategy: stateStrategy,
		worker:        NewWorker(),
		eventCh:       make(chan Event, 1024),
	}

	p.Start()
	defer p.Stop()
}
func TestIstio18454_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	stateStrategy := &Strategy{
		timerFrequency: time.Nanosecond,
		resetChan:      make(chan struct{}, 1),
		worker:         NewWorker(),
	}
	stateStrategy.startTimerFn = stateStrategy.startTimer

	p := &Processor{
		stateStrategy: stateStrategy,
		worker:        NewWorker(),
		eventCh:       make(chan Event, 1024),
	}

	p.Start()
	defer p.Stop()
}
