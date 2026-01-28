package istio17860

import (
	"context"
	sched "sched"

	"sync"
	"testing"
	"time"
)

type Proxy interface {
	IsLive() bool
}

type TestProxy struct {
	live func() bool
}

func (tp TestProxy) IsLive() bool {
	if tp.live == nil {
		return true
	}
	return tp.live()
}

type Agent interface {
	Run(ctx context.Context)
	Restart()
}

type exitStatus int

type agent struct {
	proxy        Proxy
	mu           *sync.Mutex
	statusCh     chan exitStatus
	currentEpoch int
	activeEpochs map[int]struct{}
}

func (a *agent) Run(ctx context.Context) {
	for {
		select {
		case status := <-a.statusCh:
			sched.InstChAF(747324309509, a.statusCh)
			sched.InstMutexBF(747324309513, &a.mu)
			a.mu.Lock()
			sched.InstMutexAF(747324309513, &a.mu)
			delete(a.activeEpochs, int(status))
			active := len(a.activeEpochs)
			sched.InstMutexBF(747324309514, &a.mu)
			a.mu.Unlock()
			sched.InstMutexAF(747324309514, &a.mu)
			if active == 0 {
				return
			}
		case <-ctx.Done():
			sched.InstChAF(747324309510, ctx.Done())
			return
		}
	}
}

func (a *agent) Restart() {
	sched.InstMutexBF(747324309515, &a.mu)
	a.mu.Lock()
	sched.InstMutexAF(747324309515, &a.mu)
	defer func() {
		sched.InstMutexBF(747324309516, &a.mu)
		a.mu.Unlock()
		sched.InstMutexAF(747324309516, &a.mu)
	}()

	a.waitUntilLive()
	a.currentEpoch++
	a.activeEpochs[a.currentEpoch] = struct{}{}

	go a.runWait(a.currentEpoch)
}

func (a *agent) runWait(epoch int) {
	sched.InstChBF(747324309506, a.statusCh)
	a.statusCh <- exitStatus(epoch)
	sched.InstChAF(747324309506, a.statusCh)
}

func (a *agent) waitUntilLive() {
	if len(a.activeEpochs) == 0 {
		return
	}

	interval := time.NewTicker(30 * time.Nanosecond)
	timer := time.NewTimer(100 * time.Nanosecond)
	defer func() {
		interval.Stop()
		timer.Stop()
	}()

	if a.proxy.IsLive() {
		return
	}

	for {
		select {
		case <-timer.C:
			sched.InstChAF(747324309511, timer.C)
			return
		case <-interval.C:
			sched.InstChAF(747324309512, interval.C)
			if a.proxy.IsLive() {
				return
			}
		}
	}
}

func NewAgent(proxy Proxy) Agent {
	return &agent{
		proxy:        proxy,
		mu:           &sync.Mutex{},
		statusCh:     make(chan exitStatus),
		activeEpochs: make(map[int]struct{}),
	}
}
func TestIstio17860(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	neverLive := func() bool {
		return false
	}

	a := NewAgent(TestProxy{live: neverLive})
	go func() { a.Run(ctx) }()

	a.Restart()
	go a.Restart()

	time.Sleep(200 * time.Nanosecond)
}
func TestIstio17860_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	neverLive := func() bool {
		return false
	}

	a := NewAgent(TestProxy{live: neverLive})
	go func() { a.Run(ctx) }()

	a.Restart()
	go a.Restart()

	time.Sleep(200 * time.Nanosecond)
}
