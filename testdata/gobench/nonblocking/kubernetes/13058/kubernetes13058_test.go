package kubernetes13058

import (
	"fmt"
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type ProcessFunc func(obj interface{})

type Config struct {
	Process ProcessFunc
}

type ResourceEventHandler interface {
	OnDelete(obj interface{})
}

type ResourceEventHandlerFuncs struct {
	DeleteFunc func(obj interface{})
}

func (r ResourceEventHandlerFuncs) OnDelete(obj interface{}) {
	if r.DeleteFunc != nil {
		r.DeleteFunc(obj)
	}
}

type Controller struct {
	config Config
}

func (c *Controller) processLoop() {
	for {
		c.config.Process(nil)
		break
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	Until(c.processLoop, 10*time.Millisecond, stopCh)
}

func New(c *Config) *Controller {
	ctlr := &Controller{config: *c}
	return ctlr
}

func NewInformer(h ResourceEventHandler) *Controller {
	cfg := &Config{
		Process: func(obj interface{}) {
			h.OnDelete(obj)
		},
	}
	return New(cfg)
}

func Until(f func(), period time.Duration, stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
		}
		func() {
			f()
		}()
		time.Sleep(period)
	}
}

func TestKubernetes13058(t *testing.T) {
	var testDoneWG sync.WaitGroup

	controller := NewInformer(ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			sched.InstWgBF(8806856596798832644)
			testDoneWG.Done()
			sched.InstWgAF(8806856596798832644, &testDoneWG, "done")
		},
	})

	stop := make(chan struct{})
	go func(_parentGid uint64) {
		goroutine.Enter(8806856596798832641, _parentGid)
		defer goroutine.Exit(8806856596798832641)
		controller.Run(stop)
	}(goroutine.CurrentGid())

	tests := []func(string){
		func(name string) {},
	}

	const threads = 3
	var wg sync.WaitGroup
	time.Sleep(1 * time.Second)
	sched.InstWgBF(8806856596798832645)
	wg.Add(threads * len(tests))
	sched.InstWgAF(8806856596798832645, &wg, "add")
	sched.InstWgBF(8806856596798832646)
	testDoneWG.Add(threads * len(tests))
	sched.InstWgAF(8806856596798832646, &testDoneWG, "add")
	for i := 0; i < threads; i++ {
		for j, f := range tests {
			go func(_parentGid uint64) {
				goroutine.Enter(8806856596798832642, _parentGid)
				defer goroutine.Exit(8806856596798832642)
				func(name string, f func(string)) {
					defer func() {
						sched.InstWgBF(8806856596798832647)
						wg.Done()
						sched.InstWgAF(8806856596798832647, &wg, "done")
					}()
					f(name)
				}(fmt.Sprintf("%v-%v", i, j), f)
			}(goroutine.CurrentGid())
		}
	}
	wg.Wait()
	testDoneWG.Wait()
	sched.InstChBF(8806856596798832643)
	close(stop)
	sched.InstChAF(8806856596798832643, stop, "close")
}
func TestKubernetes13058_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var testDoneWG sync.WaitGroup

	controller := NewInformer(ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			sched.InstWgBF(8806856596798832644)
			testDoneWG.Done()
			sched.InstWgAF(8806856596798832644, &testDoneWG, "done")
		},
	})

	stop := make(chan struct{})
	go func(_parentGid uint64) {
		goroutine.Enter(8806856596798832641, _parentGid)
		defer goroutine.Exit(8806856596798832641)
		controller.Run(stop)
	}(goroutine.CurrentGid())

	tests := []func(string){
		func(name string) {},
	}

	const threads = 3
	var wg sync.WaitGroup
	time.Sleep(1 * time.Second)
	sched.InstWgBF(8806856596798832645)
	wg.Add(threads * len(tests))
	sched.InstWgAF(8806856596798832645, &wg, "add")
	sched.InstWgBF(8806856596798832646)
	testDoneWG.Add(threads * len(tests))
	sched.InstWgAF(8806856596798832646, &testDoneWG, "add")
	for i := 0; i < threads; i++ {
		for j, f := range tests {
			go func(_parentGid uint64) {
				goroutine.Enter(8806856596798832642, _parentGid)
				defer goroutine.Exit(8806856596798832642)
				func(name string, f func(string)) {
					defer func() {
						sched.InstWgBF(8806856596798832647)
						wg.Done()
						sched.InstWgAF(8806856596798832647, &wg, "done")
					}()
					f(name)
				}(fmt.Sprintf("%v-%v", i, j), f)
			}(goroutine.CurrentGid())
		}
	}
	wg.Wait()
	testDoneWG.Wait()
	sched.InstChBF(8806856596798832643)
	close(stop)
	sched.InstChAF(8806856596798832643, stop, "close")
}
