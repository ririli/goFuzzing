package moby27037

import (
	"fmt"
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

func TestMoby27037(t *testing.T) {
	defer callstack.Trace(747324309505)()
	wg := sync.WaitGroup{}
	for i := 17; i <= 21; i++ {
		wg.Add(1)
		go func() {
			defer callstack.Trace(747324309506)()
			defer wg.Done()
			_ = fmt.Sprintf("v1.%d", i)
		}()
	}
	wg.Wait()
}
func TestMoby27037_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(747324309505)()
	wg := sync.WaitGroup{}
	for i := 17; i <= 21; i++ {
		wg.Add(1)
		go func() {
			defer callstack.Trace(747324309506)()
			defer wg.Done()
			_ = fmt.Sprintf("v1.%d", i)
		}()
	}
	wg.Wait()
}
