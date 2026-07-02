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
			goroutine.Enter(416611827713, _parentGid)
			defer goroutine.Exit(416611827713)
			func() {
				for work := range i.workCh {
					func() {
						defer func() {
							sched.InstWgBF(416611827718)
							i.wg.Done()
							sched.InstWgAF(416611827718, &i.wg, "done")
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
	sched.InstWgBF(416611827719)
	i.wg.Add(1)
	sched.InstWgAF(416611827719, &i.wg, "add")
	sched.InstChBF(416611827716)
	i.workCh <- w
	sched.InstChAF(416611827716, i.workCh, "send")
}

func (i *impl) Wait() {
	i.once.Do(func() {
		close(i.workCh)

		go func(_parentGid uint64) {
			goroutine.Enter(416611827714, _parentGid)
			defer goroutine.Exit(416611827714)
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
	sched.InstWgBF(416611827720)
	wg.Add(n)
	sched.InstWgAF(416611827720, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(416611827715, _parentGid)
		defer goroutine.Exit(416611827715)
		func() {
			for i := 0; i < n; i++ {
				p.Go(func() {
					atomic.AddInt32(&cntExecuted, 1)
				})
				time.Sleep(10 * time.Millisecond)
				sched.InstWgBF(416611827721)
				wg.Done()
				sched.InstWgAF(416611827721, &wg, "done")
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
	sched.InstWgBF(416611827720)
	wg.Add(n)
	sched.InstWgAF(416611827720, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(416611827715, _parentGid)
		defer goroutine.Exit(416611827715)
		func() {
			for i := 0; i < n; i++ {
				p.Go(func() {
					atomic.AddInt32(&cntExecuted, 1)
				})
				time.Sleep(10 * time.Millisecond)
				sched.InstWgBF(416611827721)
				wg.Done()
				sched.InstWgAF(416611827721, &wg, "done")
			}
		}()
	}(goroutine.CurrentGid())
	p.Wait()
	wg.Wait()
	if cntExecuted == n {
		t.Error("Not all items were expected to execute")
	}
}
