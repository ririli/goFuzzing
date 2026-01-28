/*
 * Project: kubernetes
 * Issue or PR  : https://github.com/kubernetes/kubernetes/pull/62464
 * Buggy version: a048ca888ad27367b1a7b7377c67658920adbf5d
 * fix commit-id: c1b19fce903675b82e9fdd1befcc5f5d658bfe78
 * Flaky: 8/100
 * Description:
 *   This is another example for recursive read lock bug. It has
 * been noticed by the go developers that RLock should not be
 * recursively used in the same thread.
 */

package kubernetes62464

import (
	"math/rand"
	sched "sched"
	"sync"
	"testing"
)

type State interface {
	GetCPUSetOrDefault()
	GetCPUSet() bool
	GetDefaultCPUSet()
	SetDefaultCPUSet()
}

type stateMemory struct {
	sync.RWMutex
}

func (s *stateMemory) GetCPUSetOrDefault() {
	sched.InstMutexBF(369367187457, &s)
	s.RLock()
	sched.InstMutexAF(369367187457, &s)
	defer func() {
		sched.InstMutexBF(369367187458, &s)
		s.RUnlock()
		sched.InstMutexAF(369367187458, &s)
	}()
	if ok := s.GetCPUSet(); ok {
		return
	}
	s.GetDefaultCPUSet()
}

func (s *stateMemory) GetCPUSet() bool {
	sched.InstMutexBF(369367187459, &s)
	s.RLock()
	sched.InstMutexAF(369367187459, &s)
	defer func() {
		sched.InstMutexBF(369367187460, &s)
		s.RUnlock()
		sched.InstMutexAF(369367187460, &s)
	}()

	if rand.Intn(10) > 5 {
		return true
	}
	return false
}

func (s *stateMemory) GetDefaultCPUSet() {
	sched.InstMutexBF(369367187461, &s)
	s.RLock()
	sched.InstMutexAF(369367187461, &s)
	defer func() {
		sched.InstMutexBF(369367187462, &s)
		s.RUnlock()
		sched.InstMutexAF(369367187462, &s)
	}()
}

func (s *stateMemory) SetDefaultCPUSet() {
	sched.InstMutexBF(369367187463, &s)
	s.Lock()
	sched.InstMutexAF(369367187463, &s)
	defer func() {
		sched.InstMutexBF(369367187464, &s)
		s.Unlock()
		sched.InstMutexAF(369367187464, &s)
	}()
}

type staticPolicy struct{}

func (p *staticPolicy) RemoveContainer(s State) {
	s.GetDefaultCPUSet()
	s.SetDefaultCPUSet()
}

type manager struct {
	state *stateMemory
}

func (m *manager) reconcileState() {
	m.state.GetCPUSetOrDefault()
}

func NewPolicyAndManager() (*staticPolicy, *manager) {
	s := &stateMemory{}
	m := &manager{s}
	p := &staticPolicy{}
	return p, m
}

// /
// / G1 									G2
// / m.reconcileState()
// / m.state.GetCPUSetOrDefault()
// / s.RLock()
// / s.GetCPUSet()
// / 									p.RemoveContainer()
// / 									s.GetDefaultCPUSet()
// / 									s.SetDefaultCPUSet()
// / 									s.Lock()
// / s.RLock()
// / ---------------------G1,G2 deadlock---------------------
// /
func TestKubernetes62464(t *testing.T) {
	p, m := NewPolicyAndManager()
	go m.reconcileState()
	go p.RemoveContainer(m.state)
}
func TestKubernetes62464_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	p, m := NewPolicyAndManager()
	go m.reconcileState()
	go p.RemoveContainer(m.state)
}
