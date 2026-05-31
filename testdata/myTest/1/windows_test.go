package a_test

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
	sched "toolkit/pkg/sched"
)

func TestChanAndWg(t *testing.T) {
	defer callstack.Trace(313532612609)()
	ch := make(chan int, 1)
	var wg sync.WaitGroup
	sched.InstWgBF(313532612614, &wg, 313532612609, "add")
	wg.Add(1)
	sched.InstWgAF(313532612614, &wg, 313532612609, "add")
	go func() {
		defer callstack.Trace(313532612610)()
		defer func() {
			sched.InstWgBF(313532612615, &wg, 313532612610, "done")
			wg.Done()
			sched.InstWgAF(313532612615, &wg, 313532612610, "done")
		}()
		sched.InstChBF(313532612611, ch, 313532612610, "send")
		ch <- 10
		sched.InstChAF(313532612611, ch, 313532612610, "send")
		sched.InstChBF(313532612612, ch, 313532612610, "close")
		close(ch)
		sched.InstChAF(313532612612, ch, 313532612610, "close")
	}()
	sched.InstChBF(313532612613, ch, 313532612609, "recv")
	<-ch
	sched.InstChAF(313532612613, ch, 313532612609, "recv")
	sched.InstWgBF(313532612616, &wg, 313532612609, "wait")
	wg.Wait()
	sched.InstWgAF(313532612616, &wg, 313532612609, "wait")
}
func TestChanAndWg_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(313532612609)()
	ch := make(chan int, 1)
	var wg sync.WaitGroup
	sched.InstWgBF(313532612614, &wg, 313532612609, "add")
	wg.Add(1)
	sched.InstWgAF(313532612614, &wg, 313532612609, "add")
	go func() {
		defer callstack.Trace(313532612610)()
		defer func() {
			sched.InstWgBF(313532612615, &wg, 313532612610, "done")
			wg.Done()
			sched.InstWgAF(313532612615, &wg, 313532612610, "done")
		}()
		sched.InstChBF(313532612611, ch, 313532612610, "send")
		ch <- 10
		sched.InstChAF(313532612611, ch, 313532612610, "send")
		sched.InstChBF(313532612612, ch, 313532612610, "close")
		close(ch)
		sched.InstChAF(313532612612, ch, 313532612610, "close")
	}()
	sched.InstChBF(313532612613, ch, 313532612609, "recv")
	<-ch
	sched.InstChAF(313532612613, ch, 313532612609, "recv")
	sched.InstWgBF(313532612616, &wg, 313532612609, "wait")
	wg.Wait()
	sched.InstWgAF(313532612616, &wg, 313532612609, "wait")
}
