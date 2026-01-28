/*
 * Project: kubernetes
 * Issue or PR  : https://github.com/kubernetes/kubernetes/pull/10182
 * Buggy version: 4b990d128a17eea9058d28a3b3688ab8abafbd94
 * fix commit-id: 64ad3e17ad15cd0f9a4fd86706eec1c572033254
 * Flaky: 15/100
 * Description:
 *   This is a lock-channel bug. goroutine 1 is blocked on a lock
 * held by goroutine 3, while goroutine 3 is blocked on sending
 * message to ch, which is read by goroutine 1.
 */
package kubernetes10182

import (
	sched "sched"
	"sync"
	"testing"
)

type statusManager struct {
	podStatusesLock  sync.RWMutex
	podStatusChannel chan bool
}

func (s *statusManager) Start() {
	go func() {
		for i := 0; i < 2; i++ {
			s.syncBatch()
		}
	}()
}

func (s *statusManager) syncBatch() {
	sched.InstChBF(893353197569, s.podStatusChannel)
	<-s.podStatusChannel
	sched.InstChAF(893353197569, s.podStatusChannel)
	s.DeletePodStatus()
}

func (s *statusManager) DeletePodStatus() {
	sched.InstMutexBF(893353197571, &s.podStatusesLock)
	s.podStatusesLock.Lock()
	sched.InstMutexAF(893353197571, &s.podStatusesLock)
	defer func() {
		sched.InstMutexBF(893353197572, &s.podStatusesLock)
		s.podStatusesLock.Unlock()
		sched.InstMutexAF(893353197572, &s.podStatusesLock)
	}()
}

func (s *statusManager) SetPodStatus() {
	sched.InstMutexBF(893353197573, &s.podStatusesLock)
	s.podStatusesLock.Lock()
	sched.InstMutexAF(893353197573, &s.podStatusesLock)
	defer func() {
		sched.InstMutexBF(893353197574, &s.podStatusesLock)
		s.podStatusesLock.Unlock()
		sched.InstMutexAF(893353197574, &s.podStatusesLock)
	}()
	sched.InstChBF(893353197570, s.podStatusChannel)
	s.podStatusChannel <- true
	sched.InstChAF(893353197570, s.podStatusChannel)
}

func NewStatusManager() *statusManager {
	return &statusManager{
		podStatusChannel: make(chan bool),
	}
}

// / G1 						G2							G3
// / s.Start()
// / s.syncBatch()
// / 						s.SetPodStatus()
// / <-s.podStatusChannel
// / 						s.podStatusesLock.Lock()
// / 						s.podStatusChannel <- true
// / 						s.podStatusesLock.Unlock()
// / 						return
// / s.DeletePodStatus()
// / 													s.podStatusesLock.Lock()
// / 													s.podStatusChannel <- true
// / s.podStatusesLock.Lock()
// / -----------------------------G1,G3 deadlock----------------------------
func TestKubernetes10182(t *testing.T) {
	s := NewStatusManager()
	go s.Start()
	go s.SetPodStatus() // G2
	go s.SetPodStatus() // G3
}
func TestKubernetes10182_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	s := NewStatusManager()
	go s.Start()
	go s.SetPodStatus()
	go s.SetPodStatus()
}
