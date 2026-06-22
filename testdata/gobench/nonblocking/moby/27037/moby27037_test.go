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
		wg.Add(1)
		go func() {
			goroutine.Enter(463856467969)
			defer goroutine.Exit(463856467969)
			func() {
				defer wg.Done()
				_ = fmt.Sprintf("v1.%d", i)
			}()
		}()
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
		wg.Add(1)
		go func() {
			goroutine.Enter(463856467969)
			defer goroutine.Exit(463856467969)
			func() {
				defer wg.Done()
				_ = fmt.Sprintf("v1.%d", i)
			}()
		}()
	}
	wg.Wait()
}
