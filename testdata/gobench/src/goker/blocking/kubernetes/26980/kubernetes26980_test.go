package kubernetes26980

import (
	sched "sched"
	"sync"
	"testing"
)

type processorListener struct {
	lock sync.RWMutex
	cond sync.Cond

	pendingNotifications []interface{}
}

func (p *processorListener) add(notification interface{}) {
	sched.InstMutexBF(936302870536, &p.lock)
	p.lock.Lock()
	sched.InstMutexAF(936302870536, &p.lock)
	defer func() {
		sched.InstMutexBF(936302870537, &p.lock)
		p.lock.Unlock()
		sched.InstMutexAF(936302870537, &p.lock)
	}()

	p.pendingNotifications = append(p.pendingNotifications, notification)
	p.cond.Broadcast()
}

func (p *processorListener) pop(stopCh <-chan struct{}) {
	sched.InstMutexBF(936302870538, &p.lock)
	p.lock.Lock()
	sched.InstMutexAF(936302870538, &p.lock)
	defer func() {
		sched.InstMutexBF(936302870539, &p.lock)
		p.lock.Unlock()
		sched.InstMutexAF(936302870539, &p.lock)
	}()
	for {
		for len(p.pendingNotifications) == 0 {
			select {
			case <-stopCh:
				sched.InstChAF(936302870534, stopCh)
				return
			default:
			}
			p.cond.Wait()
		}
		select { // block here
		case <-stopCh:
			sched.InstChAF(936302870535, stopCh)
			return
		}
	}
}

func newProcessListener() *processorListener {
	ret := &processorListener{
		pendingNotifications: []interface{}{},
	}
	ret.cond.L = &ret.lock
	return ret
}
func TestKubernetes26980(t *testing.T) {
	pl := newProcessListener()
	stopCh := make(chan struct{})
	defer func() {
		sched.InstChBF(936302870531, stopCh)
		close(stopCh)
		sched.InstChAF(936302870531, stopCh)
	}()
	pl.add(1)
	go pl.pop(stopCh)

	resultCh := make(chan struct{})
	go func() {
		sched.InstMutexBF(936302870540,
			// block here
			&pl.lock)
		pl.lock.Lock()
		sched.InstMutexAF(936302870540, &pl.lock)
		sched.InstChBF(936302870532, resultCh)
		close(resultCh)
		sched.InstChAF(936302870532, resultCh)
	}()
	sched.InstChBF(936302870533, resultCh)
	<-resultCh
	sched.InstChAF(936302870533, resultCh)
	sched.InstMutexBF(936302870541, &pl.lock)
	pl.lock.Unlock()
	sched.InstMutexAF(936302870541, &pl.lock)
}
func TestKubernetes26980_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	pl := newProcessListener()
	stopCh := make(chan struct{})
	defer func() {
		sched.InstChBF(936302870531, stopCh)
		close(stopCh)
		sched.InstChAF(936302870531, stopCh)
	}()
	pl.add(1)
	go pl.pop(stopCh)

	resultCh := make(chan struct{})
	go func() {
		sched.InstMutexBF(936302870540, &pl.lock)
		pl.lock.Lock()
		sched.InstMutexAF(936302870540, &pl.lock)
		sched.InstChBF(936302870532, resultCh)
		close(resultCh)
		sched.InstChAF(936302870532, resultCh)
	}()
	sched.InstChBF(936302870533, resultCh)
	<-resultCh
	sched.InstChAF(936302870533, resultCh)
	sched.InstMutexBF(936302870541, &pl.lock)
	pl.lock.Unlock()
	sched.InstMutexAF(936302870541, &pl.lock)
}
