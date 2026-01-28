package moby30408

import (
	"errors"
	sched "sched"
	"sync"
	"testing"
)

type Manifest struct {
	Implements []string
}

type Plugin struct {
	activateWait *sync.Cond
	activateErr  error
	Manifest     *Manifest
}

func (p *Plugin) waitActive() error {
	sched.InstMutexBF(1065151889411, &p.activateWait.L)
	p.activateWait.L.Lock()
	sched.InstMutexAF(1065151889411, &p.activateWait.L)
	for !p.activated() {
		p.activateWait.Wait()
	}
	sched.InstMutexBF(1065151889412, &p.activateWait.L)
	p.activateWait.L.Unlock()
	sched.InstMutexAF(1065151889412, &p.activateWait.L)
	return p.activateErr
}

func (p *Plugin) activated() bool {
	return p.Manifest != nil
}

func testActive(p *Plugin) {
	done := make(chan struct{})
	go func() {
		p.waitActive()
		sched.InstChBF(1065151889409, done)
		close(done)
		sched.InstChAF(1065151889409, done)
	}()
	sched.InstChBF(1065151889410, done)
	<-done
	sched.InstChAF(1065151889410, done)
}
func TestMoby30408(t *testing.T) {
	p := &Plugin{activateWait: sync.NewCond(&sync.Mutex{})}
	p.activateErr = errors.New("some junk happened")

	testActive(p)
}
func TestMoby30408_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	p := &Plugin{activateWait: sync.NewCond(&sync.Mutex{})}
	p.activateErr = errors.New("some junk happened")

	testActive(p)
}
