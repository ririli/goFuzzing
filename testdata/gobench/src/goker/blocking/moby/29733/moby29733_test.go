package moby29733

import (
	sched "sched"
	"sync"
	"testing"
)

type Plugin struct {
	activated    bool
	activateWait *sync.Cond
}

type plugins struct {
	sync.Mutex
	plugins map[int]*Plugin
}

func (p *Plugin) waitActive() {
	sched.InstMutexBF(807453851651, &p.activateWait.L)
	p.activateWait.L.Lock()
	sched.InstMutexAF(807453851651, &p.activateWait.L)
	for !p.activated {
		p.activateWait.Wait()
	}
	sched.InstMutexBF(807453851652, &p.activateWait.L)
	p.activateWait.L.Unlock()
	sched.InstMutexAF(807453851652, &p.activateWait.L)
}

type extpointHandlers struct {
	sync.RWMutex
	extpointHandlers map[int]struct{}
}

var (
	storage  = plugins{plugins: make(map[int]*Plugin)}
	handlers = extpointHandlers{extpointHandlers: make(map[int]struct{})}
)

func Handle() {
	sched.InstMutexBF(807453851653, &handlers)
	handlers.Lock()
	sched.InstMutexAF(807453851653, &handlers)
	for _, p := range storage.plugins {
		p.activated = false
	}
	sched.InstMutexBF(807453851654, &handlers)
	handlers.Unlock()
	sched.InstMutexAF(807453851654, &handlers)
}

func testActive(p *Plugin) {
	done := make(chan struct{})
	go func() {
		p.waitActive()
		sched.InstChBF(807453851649, done)
		close(done)
		sched.InstChAF(807453851649, done)
	}()
	sched.InstChBF(807453851650, done)
	<-done
	sched.InstChAF(807453851650, done)
}

func TestMoby29733(t *testing.T) {
	p := &Plugin{activateWait: sync.NewCond(&sync.Mutex{})}
	storage.plugins[0] = p

	testActive(p)
	Handle()
	testActive(p)
}
func TestMoby29733_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	p := &Plugin{activateWait: sync.NewCond(&sync.Mutex{})}
	storage.plugins[0] = p

	testActive(p)
	Handle()
	testActive(p)
}
