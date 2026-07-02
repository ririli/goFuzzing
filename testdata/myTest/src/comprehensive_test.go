// Package mytest 插桩测试源码（插桩前）。
//
// 本文件覆盖全部 5 个活跃 Pass 的插桩模式：
//   - TestPass:      TestComprehensive → 生成 TestComprehensive_1 + 生命周期钩子
//   - GoroutinePass: go func() / go namedFunc() → goroutine.Enter/Exit
//   - ChRecPass:     ch <- v / close(ch) / defer close(ch) → sched.InstChBF/AF
//   - SelectPass:    select { case ch <- v: } → sched.InstChSelectAF
//   - WgPass:        wg.Add / wg.Done / defer wg.Done → sched.InstWgBF/AF
package mytest

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

// sendToCh 用于测试 go namedFunc() 模式下的 GoroutinePass 插桩，
// 同时内部的 ch <- v 和 defer wg.Done() 分别覆盖 ChRecPass 和 WgPass。
func sendToCh(ch chan<- int, wg *sync.WaitGroup) {
	defer func() {
		sched.InstWgBF(
			// WgPass: deferred Done
			244813135880)
		wg.Done()
		sched.InstWgAF(244813135880, &wg, "done")
	}()
	sched.InstChBF(244813135875)
	ch <- 42
	sched.InstChAF( // ChRecPass: direct send
		244813135875, ch, "send")
}

// TestComprehensive 覆盖全部 5 个插桩 Pass。
// 该测试在插桩前后均可正常编译运行，不会死锁。
func TestComprehensive(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(
		// WgPass: direct Add
		244813135881)
	wg.Add(2)
	sched.InstWgAF(244813135881, &wg, "add")

	ch := make(chan int, 1) // buffered channel
	done := make(chan struct{})
	defer // signal channel
	func() {
		sched.InstChBF(

			// ChRecPass: deferred close（在 wg.Wait 之后执行，安全）
			244813135876)
		close(ch)
		sched.

			// GoroutinePass: go namedFunc()
			InstChAF(244813135876, ch, "close")
	}()

	go func(_parentGid uint64) {

		// GoroutinePass: go func literal
		// SelectPass:   send in select case
		// WgPass:       defer Done
		goroutine.Enter(244813135873, _parentGid)
		defer goroutine.Exit(244813135873)
		sendToCh(ch, &wg)
	}(goroutine.CurrentGid())

	go func(_parentGid uint64) {
		goroutine.
			// WgPass: deferred Done
			Enter(244813135874, _parentGid)
		defer goroutine.Exit(

			// SelectPass: send in select → InstChSelectAF
			244813135874)
		func() {
			defer func() {
				sched.InstWgBF(244813135882)
				wg.Done()
				sched.InstWgAF(244813135882, &wg, "done")
			}()
			select {
			case ch <- 1:
				sched.InstChSelectAF(244813135879, ch, "send")
				t.Log("sent via select")
			case <-done: // SelectPass: recv in select → NOT instrumented
				t.Log("done signal received")
			}
		}()
	}(goroutine.CurrentGid(

	// consume from buffer, let goroutines make progress
	))

	<-ch
	sched.InstChBF(244813135878)
	close(done)
	sched. // ChRecPass: direct close
		InstChAF(244813135878, done, "close")

	wg.Wait() // WgPass: wg.Wait (currently NOT instrumented by WgPass)
}
func TestComprehensive_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(244813135881)
	wg.Add(2)
	sched.InstWgAF(244813135881, &wg, "add")

	ch := make(chan int, 1)
	done := make(chan struct{})
	defer func() {
		sched.InstChBF(244813135876)
		close(ch)
		sched.InstChAF(244813135876, ch, "close")
	}()

	go func(_parentGid uint64) {
		goroutine.Enter(244813135873, _parentGid)
		defer goroutine.Exit(244813135873)
		sendToCh(ch, &wg)
	}(goroutine.CurrentGid())

	go func(_parentGid uint64) {
		goroutine.Enter(244813135874, _parentGid)
		defer goroutine.Exit(244813135874)
		func() {
			defer func() {
				sched.InstWgBF(244813135882)
				wg.Done()
				sched.InstWgAF(244813135882, &wg, "done")
			}()
			select {
			case ch <- 1:
				sched.InstChSelectAF(244813135879, ch, "send")
				t.Log("sent via select")
			case <-done:
				t.Log("done signal received")
			}
		}()
	}(goroutine.CurrentGid())

	<-ch
	sched.InstChBF(244813135878)
	close(done)
	sched.InstChAF(244813135878, done, "close")

	wg.Wait()
}
