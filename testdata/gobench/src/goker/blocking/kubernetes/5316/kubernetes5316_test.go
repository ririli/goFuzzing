/*
 * Project: kubernetes
 * Issue or PR  : https://github.com/kubernetes/kubernetes/pull/5316
 * Buggy version: c868b0bbf09128960bc7c4ada1a77347a464d876
 * fix commit-id: cc3a433a7abc89d2f766d4c87eaae9448e3dc091
 * Flaky: 100/100
 * Description:
 *   If the main goroutine selects a case that doesn’t consumes
 * the channels, the anonymous goroutine will be blocked on sending
 * to channel.
 */

package kubernetes5316

import (
	"errors"
	"math/rand"
	sched "sched"
	"testing"
	"time"
)

func finishRequest(timeout time.Duration, fn func() error) {
	ch := make(chan bool)     // FIX: ch := make(chan bool, 1)
	errCh := make(chan error) // FIX: errCh := make(chan error, 1)
	go func() {               // G2
		if err := fn(); err != nil {
			sched.InstChBF(949187772417, errCh)
			errCh <- err
			sched.InstChAF(949187772417, errCh)
		} else {
			sched.InstChBF(949187772418, ch)
			ch <- true
			sched.InstChAF(949187772418, ch)
		}
	}()

	select {
	case <-ch:
		sched.InstChAF(949187772422, ch)
	case <-errCh:
		sched.InstChAF(949187772423, errCh)
	case <-time.After(timeout):
		sched.InstChAF(

			///
			/// G1 						G2
			/// finishRequest()
			/// 						fn()
			/// time.After()
			/// 						errCh<-/ch<-
			/// --------------G2 leak----------------
			///
			949187772424, time.After(timeout))
	}
}

func TestKubernetes5316(t *testing.T) {
	fn := func() error {
		time.Sleep(2 * time.Millisecond)
		if rand.Intn(10) > 5 {
			return errors.New("Error")
		}
		return nil
	}
	go finishRequest(time.Millisecond, fn) // G1
}
func TestKubernetes5316_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	fn := func() error {
		time.Sleep(2 * time.Millisecond)
		if rand.Intn(10) > 5 {
			return errors.New("Error")
		}
		return nil
	}
	go finishRequest(time.Millisecond, fn)
}
