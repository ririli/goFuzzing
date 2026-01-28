/*
 * Project: moby
 * Issue or PR  : https://github.com/moby/moby/pull/33781
 * Buggy version: 33fd3817b0f5ca4b87f0a75c2bd583b4425d392b
 * fix commit-id: 67297ba0051d39be544009ba76abea14bc0be8a4
 * Flaky: 25/100
 * Description:
 *   The goroutine created using anonymous function is blocked at
 * sending message to a unbuffered channel. However there exists a
 * path in the parent goroutine where the parent function will
 * return without draining the channel.
 */

package moby33781

import (
	"context"
	sched "sched"
	"testing"
	"time"
)

func monitor(stop chan bool) {
	probeInterval := 50 * time.Nanosecond
	probeTimeout := 50 * time.Nanosecond
	for {
		select {
		case <-stop:
			sched.InstChAF(536870912010, stop)
			return
		case <-time.After(probeInterval):
			sched.InstChAF(536870912011, time.After(probeInterval))
			results := make(chan bool)
			ctx, cancelProbe := context.WithTimeout(context.Background(), probeTimeout)
			go func() {
				sched. // G3
					InstChBF(536870912003, results)
				results <- true
				sched.InstChAF(536870912003, results)
				sched.InstChBF(536870912004, results)
				close(results)
				sched.InstChAF(536870912004, results)
			}()
			select {
			case <-stop:
				sched.
					// results should be drained here
					InstChAF(536870912012, stop)

				cancelProbe()
				return
			case <-results:
				sched.InstChAF(536870912013, results)
				cancelProbe()
			case <-ctx.Done():
				sched.InstChAF(536870912014, ctx.Done())
				cancelProbe()
				sched.InstChBF(536870912008, results)
				<-results
				sched.InstChAF(536870912008, results)
			}
		}
	}
}

///
/// G1 				G2				G3
/// monitor()
/// <-time.After()
/// 				stop <-
/// <-stop
/// 				return
/// cancelProbe()
/// return
/// 								result<-
///----------------G3 leak------------------
///

func TestMoby33781(t *testing.T) {
	stop := make(chan bool)
	go monitor(stop) // G1
	go func() {      // G2
		time.Sleep(50 * time.Nanosecond)
		sched.InstChBF(536870912009, stop)
		stop <- true
		sched.InstChAF(536870912009, stop)
	}()
}
func TestMoby33781_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	stop := make(chan bool)
	go monitor(stop)
	go func() {
		time.Sleep(50 * time.Nanosecond)
		sched.InstChBF(536870912009, stop)
		stop <- true
		sched.InstChAF(536870912009, stop)
	}()
}
