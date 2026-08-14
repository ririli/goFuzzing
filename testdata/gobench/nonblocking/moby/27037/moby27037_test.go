package moby27037

import (
	"fmt"
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
)

func TestMoby27037(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 17; i <= 21; i++ {
		sched.InstWgBF(2110736234232938498)
		wg.Add(1)
		sched.InstWgAF(2110736234232938498, &wg, "add")
		go func(_parentGid uint64) {
			goroutine.Enter(2110736234232938497, _parentGid)
			defer goroutine.Exit(2110736234232938497)
			func() {
				defer func() {
					sched.InstWgBF(2110736234232938499)
					wg.Done()
					sched.InstWgAF(2110736234232938499, &wg, "done")
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
		sched.InstWgBF(2110736234232938498)
		wg.Add(1)
		sched.InstWgAF(2110736234232938498, &wg, "add")
		go func(_parentGid uint64) {
			goroutine.Enter(2110736234232938497, _parentGid)
			defer goroutine.Exit(2110736234232938497)
			func() {
				defer func() {
					sched.InstWgBF(2110736234232938499)
					wg.Done()
					sched.InstWgAF(2110736234232938499, &wg, "done")
				}()
				_ = fmt.Sprintf("v1.%d", i)
			}()
		}(goroutine.CurrentGid())
	}
	wg.Wait()
}
