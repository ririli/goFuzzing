/*
 * Project: cockroach
 * Issue or PR  : https://github.com/cockroachdb/cockroach/pull/9935
 * Buggy version: 4df302cc3f03328395dc3fefbfba58b7718e4f2f
 * fix commit-id: ed6a100ba38dd51b0888b9a3d3ac6bdbb26c528c
 * Flaky: 100/100
 * Description: This bug is caused by acquiring l.mu.Lock() twice. The fix is
 * to release l.mu.Lock() before acquiring l.mu.Lock for the second time.
 */
package cockroach9935

import (
	"errors"
	"math/rand"
	sched "sched"
	"sync"
	"testing"
)

type loggingT struct {
	mu sync.Mutex
}

func (l *loggingT) outputLogEntry() {
	sched.InstMutexBF(579820584961, &l.mu)
	l.mu.Lock()
	sched.InstMutexAF(579820584961, &l.mu)
	if err := l.createFile(); err != nil {
		l.exit(err)
	}
	sched.InstMutexBF(579820584962, &l.mu)
	l.mu.Unlock()
	sched.InstMutexAF(579820584962, &l.mu)
}
func (l *loggingT) createFile() error {
	if rand.Intn(8)%4 > 0 {
		return errors.New("")
	}
	return nil
}
func (l *loggingT) exit(err error) {
	sched.InstMutexBF(579820584963, &l.mu)
	l.mu.Lock()
	sched.InstMutexAF(579820584963, &l.mu)
	defer func() {
		sched.InstMutexBF(579820584964, &l.mu)
		l.mu.Unlock()
		sched.InstMutexAF(579820584964, &l.mu)
	}()
}
func TestCockroach9935(t *testing.T) {
	l := &loggingT{}
	go l.outputLogEntry()
}
func TestCockroach9935_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	l := &loggingT{}
	go l.outputLogEntry()
}
