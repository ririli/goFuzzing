/*
 * Project: moby
 * Issue or PR  : https://github.com/moby/moby/pull/36114
 * Buggy version: 6d4d3c52ae7c3f910bfc7552a2a673a8338e5b9f
 * fix commit-id: a44fcd3d27c06aaa60d8d1cbce169f0d982e74b1
 * Flaky: 100/100
 * Description:
 *   This is a double lock bug. The the lock for the
 * struct svm has already been locked when calling
 * svm.hotRemoveVHDsAtStart()
 */
package moby36114

import (
	sched "sched"
	"sync"
	"testing"
)

type serviceVM struct {
	sync.Mutex
}

func (svm *serviceVM) hotAddVHDsAtStart() {
	sched.InstMutexBF(936302870529, &svm)
	svm.Lock()
	sched.InstMutexAF(936302870529, &svm)
	defer func() {
		sched.InstMutexBF(936302870530, &svm)
		svm.Unlock()
		sched.InstMutexAF(936302870530, &svm)
	}()
	svm.hotRemoveVHDsAtStart()
}

func (svm *serviceVM) hotRemoveVHDsAtStart() {
	sched.InstMutexBF(
		// Double lock here
		936302870531, &svm)
	svm.Lock()
	sched.InstMutexAF(936302870531, &svm)
	defer func() {
		sched.InstMutexBF(936302870532, &svm)
		svm.Unlock()
		sched.InstMutexAF(936302870532, &svm)
	}()
}

func TestMoby36114(t *testing.T) {
	s := &serviceVM{}
	go s.hotAddVHDsAtStart()
}
func TestMoby36114_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	s := &serviceVM{}
	go s.hotAddVHDsAtStart()
}
