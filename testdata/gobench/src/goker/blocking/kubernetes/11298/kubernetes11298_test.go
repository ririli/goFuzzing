package kubernetes11298

import (
	sched "sched"
	"sync"
	"testing"
	"time"
)

type Signal <-chan struct{}

func After(f func()) Signal {
	ch := make(chan struct{})
	go func() {
		defer func() {
			sched.InstChBF(738734374913, ch)
			close(ch)
			sched.InstChAF(738734374913, ch)
		}()
		if f != nil {
			f()
		}
	}()
	return Signal(ch)
}

func Until(f func(), period time.Duration, stopCh <-chan struct{}) {
	if f == nil {
		return
	}
	for {
		select {
		case <-stopCh:
			sched.InstChAF(738734374924, stopCh)
			return
		default:
		}
		func() {
			f()
		}()
		select {
		case <-stopCh:
			sched.InstChAF(738734374925, stopCh)
		case <-time.After(period):
			sched.InstChAF(738734374926, time.After(period))
		}
	}

}

type notifier struct {
	lock sync.Mutex
	cond *sync.Cond
}

func (n *notifier) serviceLoop(abort <-chan struct{}) {
	sched.InstMutexBF(738734374931, &n.lock)
	n.lock.Lock()
	sched.InstMutexAF(738734374931, &n.lock)
	defer func() {
		sched.InstMutexBF(738734374932, &n.lock)
		n.lock.Unlock()
		sched.InstMutexAF(738734374932, &n.lock)
	}()
	for {
		select {
		case <-abort:
			sched.InstChAF(738734374927, abort)
			return
		default:
			ch := After(n.cond.Wait)
			select {
			case <-abort:
				sched.InstChAF(738734374928, abort)
				n.cond.Signal()
				sched.InstChBF(738734374919, ch)
				<-ch
				sched.InstChAF(738734374919, ch)
				return
			case <-ch:
				sched.InstChAF(738734374929, ch)
			}
		}
	}
}

func Notify(abort <-chan struct{}) {
	n := &notifier{}
	n.cond = sync.NewCond(&n.lock)
	finished := After(func() {
		Until(func() {
			for {
				select {
				case <-abort:
					sched.InstChAF(738734374930, abort)
					return
				default:
					func() {
						sched.InstMutexBF(738734374933, &n.lock)
						n.lock.Lock()
						sched.InstMutexAF(738734374933, &n.lock)
						defer func() {
							sched.InstMutexBF(738734374934, &n.lock)
							n.lock.Unlock()
							sched.InstMutexAF(738734374934, &n.lock)
						}()
						n.cond.Signal()
					}()
				}
			}
		}, 0, abort)
	})
	Until(func() { n.serviceLoop(finished) }, 0, abort)
}
func TestKubernetes11298(t *testing.T) {
	done := make(chan struct{})
	notifyDone := After(func() { Notify(done) })
	go func() {
		defer func() {
			sched.InstChBF(738734374922, done)
			close(done)
			sched.InstChAF(738734374922, done)
		}()
		time.Sleep(300 * time.Nanosecond)
	}()
	sched.InstChBF(738734374923, notifyDone)
	<-notifyDone
	sched.InstChAF(738734374923, notifyDone)
}
func TestKubernetes11298_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	done := make(chan struct{})
	notifyDone := After(func() { Notify(done) })
	go func() {
		defer func() {
			sched.InstChBF(738734374922, done)
			close(done)
			sched.InstChAF(738734374922, done)
		}()
		time.Sleep(300 * time.Nanosecond)
	}()
	sched.InstChBF(738734374923, notifyDone)
	<-notifyDone
	sched.InstChAF(738734374923, notifyDone)
}
