package serving3068

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type Interface interface {
	Go(func())
	Wait()
}

type impl struct {
	wg     sync.WaitGroup
	workCh chan func()
	once   sync.Once
}

var _ Interface = (*impl)(nil)

func NewWithCapacity(workers, capacity int) Interface {
	i := &impl{
		workCh: make(chan func(), capacity),
	}

	for idx := 0; idx < workers; idx++ {
		go func(_parentGid uint64) {
			goroutine.Enter(11727672913447354369, _parentGid)
			defer goroutine.Exit(11727672913447354369)
			func() {
				for work := range i.workCh {
					func() {
						defer func() {
							sched.InstWgBF(11727672913447354374)
							i.wg.Done()
							sched.InstWgAF(11727672913447354374, &i.wg, "done")
						}()
						work()
					}()
				}
			}()
		}(goroutine.CurrentGid())
	}

	return i
}

func (i *impl) Go(w func()) {
	sched.InstWgBF(11727672913447354375)
	i.wg.Add(1)
	sched.InstWgAF(11727672913447354375, &i.wg, "add")
	sched.InstChBF(11727672913447354372)
	i.workCh <- w
	sched.InstChAF(11727672913447354372, i.workCh, "send")
}

func (i *impl) Wait() {
	i.once.Do(func() {
		close(i.workCh)

		go func(_parentGid uint64) {
			goroutine.Enter(11727672913447354370, _parentGid)
			defer goroutine.Exit(11727672913447354370)
			func() {
				i.wg.Wait()
			}()
		}(goroutine.CurrentGid())
	})
}

func TestServing3068(t *testing.T) {
	p := NewWithCapacity(1, 5)
	wg := &sync.WaitGroup{}
	var cntExecuted int32
	const n = 5
	sched.InstWgBF(11727672913447354376)
	wg.Add(n)
	sched.InstWgAF(11727672913447354376, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(11727672913447354371, _parentGid)
		defer goroutine.Exit(11727672913447354371)
		func() {
			for i := 0; i < n; i++ {
				p.Go(func() {
					atomic.AddInt32(&cntExecuted, 1)
				})
				time.Sleep(10 * time.Millisecond)
				sched.InstWgBF(11727672913447354377)
				wg.Done()
				sched.InstWgAF(11727672913447354377, &wg, "done")
			}
		}()
	}(goroutine.CurrentGid())
	p.Wait()
	wg.Wait()
	if cntExecuted == n {
		t.Error("Not all items were expected to execute")
	}
}
func TestServing3068_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	p := NewWithCapacity(1, 5)
	wg := &sync.WaitGroup{}
	var cntExecuted int32
	const n = 5
	sched.InstWgBF(11727672913447354376)
	wg.Add(n)
	sched.InstWgAF(11727672913447354376, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(11727672913447354371, _parentGid)
		defer goroutine.Exit(11727672913447354371)
		func() {
			for i := 0; i < n; i++ {
				p.Go(func() {
					atomic.AddInt32(&cntExecuted, 1)
				})
				time.Sleep(10 * time.Millisecond)
				sched.InstWgBF(11727672913447354377)
				wg.Done()
				sched.InstWgAF(11727672913447354377, &wg, "done")
			}
		}()
	}(goroutine.CurrentGid())
	p.Wait()
	wg.Wait()
	if cntExecuted == n {
		t.Error("Not all items were expected to execute")
	}
}
