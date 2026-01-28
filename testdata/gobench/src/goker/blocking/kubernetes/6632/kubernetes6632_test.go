/*
 * Project: kubernetes
 * Issue or PR  : https://github.com/kubernetes/kubernetes/pull/6632
 * Buggy version: e597b41d939573502c8dda1dde7bf3439325fb5d
 * fix commit-id: 82afb7ab1fe12cf2efceede2322d082eaf5d5adc
 * Flaky: 4/100
 * Description:
 *   This is a lock-channel bug. When resetChan is full, WriteFrame
 * holds the lock and blocks on the channel. Then monitor() fails
 * to close the resetChan because lock is already held by WriteFrame.
 *   Fix: create a goroutine to drain the channel
 */
package kubernetes6632

import (
	sched "sched"
	"sync"
	"testing"
)

type Connection struct {
	closeChan chan bool
}

type idleAwareFramer struct {
	resetChan chan bool
	writeLock sync.Mutex
	conn      *Connection
}

func (i *idleAwareFramer) monitor() {
	var resetChan = i.resetChan
Loop:
	for {
		select {
		case <-i.conn.closeChan:
			sched.InstChAF(979252543493, i.conn.closeChan)
			sched.InstMutexBF(979252543494, &i.writeLock)
			i.writeLock.Lock()
			sched.InstMutexAF(979252543494, &i.writeLock)
			sched.InstChBF(979252543490, resetChan)
			close(resetChan)
			sched.InstChAF(979252543490, resetChan)
			i.resetChan = nil
			sched.InstMutexBF(979252543495, &i.writeLock)
			i.writeLock.Unlock()
			sched.InstMutexAF(979252543495, &i.writeLock)
			break Loop
		}
	}
}

func (i *idleAwareFramer) WriteFrame() {
	sched.InstMutexBF(979252543496, &i.writeLock)
	i.writeLock.Lock()
	sched.InstMutexAF(979252543496, &i.writeLock)
	defer func() {
		sched.InstMutexBF(979252543497, &i.writeLock)
		i.writeLock.Unlock()
		sched.InstMutexAF(979252543497, &i.writeLock)
	}()
	if i.resetChan == nil {
		return
	}
	sched.InstChBF(979252543491, i.resetChan)
	i.resetChan <- true
	sched.InstChAF(979252543491, i.resetChan)
}

func NewIdleAwareFramer() *idleAwareFramer {
	return &idleAwareFramer{
		resetChan: make(chan bool),
		conn: &Connection{
			closeChan: make(chan bool),
		},
	}
}

// /
// / G1						G2					helper goroutine
// / i.monitor()
// / <-i.conn.closeChan
// /							i.WriteFrame()
// /							i.writeLock.Lock()
// /							i.resetChan <-
// /												i.conn.closeChan<-
// /	i.writeLock.Lock()
// /	----------------------G1,G2 deadlock------------------------
// /
func TestKubernetes6632(t *testing.T) {
	i := NewIdleAwareFramer()

	go func() {
		sched. // helper goroutine
			InstChBF(979252543492, i.conn.closeChan)
		i.conn.closeChan <- true
		sched.InstChAF(979252543492, i.conn.closeChan)
	}()
	go i.monitor()    // G1
	go i.WriteFrame() // G2
}
func TestKubernetes6632_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	i := NewIdleAwareFramer()

	go func() {
		sched.InstChBF(979252543492, i.conn.closeChan)
		i.conn.closeChan <- true
		sched.InstChAF(979252543492, i.conn.closeChan)
	}()
	go i.monitor()
	go i.WriteFrame()
}
