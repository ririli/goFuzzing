package a_test

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
	sched "toolkit/pkg/sched"
)

// TestSelectInst 验证 select 插桩效果：
//   - select 中的 send/recv 应被 SelectPass 插入 InstChSelectAF（仅 AF，无 BF）
//   - 普通 close 应被 ChRecPass 插入 InstChBF + InstChAF（完整 BF+AF）
//   - goroutine 间通过 ch1 和 done 同步，不会死锁
func TestSelectInst(t *testing.T) {
	defer callstack.Trace(313532612609)()
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	var wg sync.WaitGroup
	sched.InstWgBF(

		// goroutine: 在 select 中同时 send 和 recv
		313532612615, &wg, 313532612609, "add")
	wg.Add(1)
	sched.InstWgAF(313532612615, &wg, 313532612609, "add")

	go func() {
		defer callstack.Trace(313532612610)()
		defer func() {
			sched.

				// ★ select send → 插桩后应有 InstChSelectAF
				InstWgBF(313532612616, &wg, 313532612610, "done")
			wg.Done()
			sched.InstWgAF(313532612616, &wg, 313532612610, "done")
		}()
		select {
		case ch1 <- 42:
			sched.InstChSelectAF(313532612613, ch1, 313532612610, "send")
			t.Log("sent 42")
		case v := <-ch2:
			sched. // ★ select recv → 插桩后应有 InstChSelectAF
				InstChSelectAF(313532612614, ch2, 313532612610, "recv")
			t.Log("received", v)
		}
	}()

	// 等待 goroutine 进入 select 阻塞
	time.Sleep(10 * time.Millisecond)
	sched.

		// 普通 close（非 select）→ 插桩后应有 InstChBF + InstChAF
		InstChBF(313532612612, ch1, 313532612609, "close")
	close(ch1)
	sched.InstChAF(313532612612, ch1, 313532612609, "close")
	sched.InstWgBF(313532612617, &wg, 313532612609, "wait")
	wg.Wait()
	sched.InstWgAF(313532612617, &wg, 313532612609, "wait")
}
func TestSelectInst_1(t *testing.T) {
	callstack.ParseInput()
	sched.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(313532612609)()
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	var wg sync.WaitGroup
	sched.InstWgBF(313532612615, &wg, 313532612609, "add")
	wg.Add(1)
	sched.InstWgAF(313532612615, &wg, 313532612609, "add")

	go func() {
		defer callstack.Trace(313532612610)()
		defer func() {
			sched.InstWgBF(313532612616, &wg, 313532612610, "done")
			wg.Done()
			sched.InstWgAF(313532612616, &wg, 313532612610, "done")
		}()
		select {
		case ch1 <- 42:
			sched.InstChSelectAF(313532612613, ch1, 313532612610, "send")
			t.Log("sent 42")
		case v := <-ch2:
			sched.InstChSelectAF(313532612614, ch2, 313532612610, "recv")
			t.Log("received", v)
		}
	}()

	time.Sleep(10 * time.Millisecond)
	sched.InstChBF(313532612612, ch1, 313532612609, "close")
	close(ch1)
	sched.InstChAF(313532612612, ch1, 313532612609, "close")
	sched.InstWgBF(313532612617, &wg, 313532612609, "wait")
	wg.Wait()
	sched.InstWgAF(313532612617, &wg, 313532612609, "wait")
}
