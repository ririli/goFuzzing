package mytest3

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

// TestGoroutineParentChild 测试父子goroutine关系追踪
// 结构:
//
//	main (gid=0)
//	├── worker1
//	│   └── helper1
//	└── worker2
//	    └── helper2

func child() {
	a := 0
	for i := 0; i < 100; i++ {
		a++
	}
	a--
}

func TestGoroutineParentChild(t *testing.T) {
	var wg sync.WaitGroup
	ch := make(chan int, 2)

	// worker1: 由main创建
	wg.Add(1)
	go func(_parentGid uint64) {
		goroutine.Enter(1052266987521, _parentGid)
		defer

		// helper1: 由worker1创建
		goroutine.Exit(1052266987521)
		func() {
			defer wg.Done()
			time.Sleep(30 * time.Millisecond)

			wg.Add(1)
			go func(_parentGid uint64) {
				goroutine.Enter(1052266987522, _parentGid)
				defer goroutine.Exit(1052266987522)
				func() {
					defer wg.Done()
					time.Sleep(20 * time.Millisecond)
					ch <- 1
				}()
			}(goroutine.

				// worker2: 由main创建
				CurrentGid())
		}()
	}(goroutine.CurrentGid())

	wg.Add(1)
	go func(_parentGid uint64) {
		goroutine.Enter(1052266987523, _parentGid)
		defer

		// helper2: 由worker2创建
		goroutine.Exit(1052266987523)
		func() {
			defer wg.Done()
			time.Sleep(40 * time.Millisecond)

			wg.Add(1)
			go func(_parentGid uint64) {
				goroutine.Enter(1052266987524, _parentGid)
				defer goroutine.Exit(1052266987524)
				func() {
					defer wg.Done()
					time.Sleep(20 * time.Millisecond)
					ch <- 2
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(1052266987525, _parentGid)
				defer goroutine.Exit(1052266987525)
				child()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())

	wg.Wait()
	close(ch)

	for v := range ch {
		t.Logf("received: %d", v)
	}
}
func TestGoroutineParentChild_1(t *testing.T) {
	defer goroutine.PrintRecords()
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	ch := make(chan int, 2)

	wg.Add(1)
	go func(_parentGid uint64) {
		goroutine.Enter(1052266987521, _parentGid)
		defer goroutine.Exit(1052266987521)
		func() {
			defer wg.Done()
			time.Sleep(30 * time.Millisecond)

			wg.Add(1)
			go func(_parentGid uint64) {
				goroutine.Enter(1052266987522, _parentGid)
				defer goroutine.Exit(1052266987522)
				func() {
					defer wg.Done()
					time.Sleep(20 * time.Millisecond)
					ch <- 1
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())

	wg.Add(1)
	go func(_parentGid uint64) {
		goroutine.Enter(1052266987523, _parentGid)
		defer goroutine.Exit(1052266987523)
		func() {
			defer wg.Done()
			time.Sleep(40 * time.Millisecond)

			wg.Add(1)
			go func(_parentGid uint64) {
				goroutine.Enter(1052266987524, _parentGid)
				defer goroutine.Exit(1052266987524)
				func() {
					defer wg.Done()
					time.Sleep(20 * time.Millisecond)
					ch <- 2
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(1052266987525, _parentGid)
				defer goroutine.Exit(1052266987525)
				child()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())

	wg.Wait()
	close(ch)

	for v := range ch {
		t.Logf("received: %d", v)
	}
}
