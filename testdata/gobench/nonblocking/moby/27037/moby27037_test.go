package moby27037

import (
	"fmt"
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

func TestMoby27037(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 17; i <= 21; i++ {
		sched.InstWgBF(463856467970)
		wg.Add(1)
		sched.InstWgAF(463856467970, &wg, "add")
		go func(_parentGid uint64) {
			goroutine.Enter(463856467969, _parentGid)
			defer goroutine.Exit(463856467969)
			func() {
				defer func() {
					sched.InstWgBF(463856467971)
					wg.Done()
					sched.InstWgAF(463856467971, &wg, "done")
				}()
				_ = fmt.Sprintf("v1.%d", i)
			}()
		}(goroutine.CurrentGid())
	}
	wg.Wait()
}
func TestMoby27037_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	wg := sync.WaitGroup{}
	for i := 17; i <= 21; i++ {
		sched.InstWgBF(463856467970)
		wg.Add(1)
		sched.InstWgAF(463856467970, &wg, "add")
		go func(_parentGid uint64) {
			goroutine.Enter(463856467969, _parentGid)
			defer goroutine.Exit(463856467969)
			func() {
				defer func() {
					sched.InstWgBF(463856467971)
					wg.Done()
					sched.InstWgAF(463856467971, &wg, "done")
				}()
				_ = fmt.Sprintf("v1.%d", i)
			}()
		}(goroutine.CurrentGid())
	}
	wg.Wait()
}
