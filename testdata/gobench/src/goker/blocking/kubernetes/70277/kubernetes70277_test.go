package kubernetes70277

import (
	sched "sched"
	"testing"
	"time"
)

var ForerverTestTimeout = time.Second * 20

type WaitFunc func(done <-chan struct{}) <-chan struct{}

type ConditionFunc func() (done bool, err error)

func WaitFor(wait WaitFunc, fn ConditionFunc, done <-chan struct{}) error {
	c := wait(done)
	for {
		_, open := <-c
		ok, err := fn()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if !open {
			break
		}
	}
	return nil
}

func poller(interval, timeout time.Duration) WaitFunc {
	return WaitFunc(func(done <-chan struct{}) <-chan struct{} {
		ch := make(chan struct{})
		go func() {
			defer func() {
				sched.InstChBF(158913789953, ch)
				close(ch)
				sched.InstChAF(158913789953, ch)
			}()

			tick := time.NewTicker(interval)
			defer tick.Stop()

			var after <-chan time.Time
			if timeout != 0 {
				timer := time.NewTimer(timeout)
				after = timer.C
				defer timer.Stop()
			}
			for {
				select {
				case <-tick.C:
					sched.InstChAF(158913789960, tick.C)
					select {
					case ch <- struct{}{}:
						sched.InstChAF(158913789963, ch)
					default:
					}
				case <-after:
					sched.InstChAF(158913789961, after)
					return
				case <-done:
					sched.InstChAF(158913789962, done)
					return
				}
			}
		}()

		return ch
	})
}

func TestKubernetes70277(t *testing.T) {
	stopCh := make(chan struct{})
	defer func() {
		sched.InstChBF(158913789958, stopCh)
		close(stopCh)
		sched.InstChAF(158913789958, stopCh)
	}()
	waitFunc := poller(time.Millisecond, ForerverTestTimeout)
	var doneCh <-chan struct{}

	WaitFor(func(done <-chan struct{}) <-chan struct{} {
		doneCh = done
		return waitFunc(done)
	}, func() (bool, error) {
		return true, nil
	}, stopCh)
	sched.InstChBF(

		// block here
		158913789959, doneCh)
	<-doneCh
	sched.InstChAF(158913789959, doneCh)
}
func TestKubernetes70277_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	stopCh := make(chan struct{})
	defer func() {
		sched.InstChBF(158913789958, stopCh)
		close(stopCh)
		sched.InstChAF(158913789958, stopCh)
	}()
	waitFunc := poller(time.Millisecond, ForerverTestTimeout)
	var doneCh <-chan struct{}

	WaitFor(func(done <-chan struct{}) <-chan struct{} {
		doneCh = done
		return waitFunc(done)
	}, func() (bool, error) {
		return true, nil
	}, stopCh)
	sched.InstChBF(158913789959, doneCh)
	<-doneCh
	sched.InstChAF(158913789959, doneCh)
}
