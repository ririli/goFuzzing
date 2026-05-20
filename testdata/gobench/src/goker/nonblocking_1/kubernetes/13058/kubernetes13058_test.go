package kubernetes13058

import (
	"fmt"
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
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
	defer callstack.Trace(884763262977)()
	if r.DeleteFunc != nil {
		r.DeleteFunc(obj)
	}
}

type Controller struct {
	config Config
}

func (c *Controller) processLoop() {
	defer callstack.Trace(884763262978)()
	for {
		c.config.Process(nil)
		break
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer callstack.Trace(884763262979)()
	Until(c.processLoop, 10*time.Millisecond, stopCh)
}

func New(c *Config) *Controller {
	defer callstack.Trace(884763262980)()
	ctlr := &Controller{config: *c}
	return ctlr
}

func NewInformer(h ResourceEventHandler) *Controller {
	defer callstack.Trace(884763262981)()
	cfg := &Config{
		Process: func(obj interface{}) {
			defer callstack.Trace(884763262982)()
			h.OnDelete(obj)
		},
	}
	return New(cfg)
}

func Until(f func(), period time.Duration, stopCh <-chan struct{}) {
	defer callstack.Trace(884763262983)()
	for {
		select {
		case <-stopCh:
			return
		default:
		}
		func() {
			defer callstack.Trace(884763262984)()
			f()
		}()
		time.Sleep(period)
	}
}

func TestKubernetes13058(t *testing.T) {
	defer callstack.Trace(884763262985)()
	var testDoneWG sync.WaitGroup

	controller := NewInformer(ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			defer callstack.Trace(884763262986)()
			testDoneWG.Done()
		},
	})

	stop := make(chan struct{})
	go controller.Run(stop)

	tests := []func(string){
		func(name string) {},
	}

	const threads = 3
	var wg sync.WaitGroup
	wg.Add(threads * len(tests))
	testDoneWG.Add(threads * len(tests))
	for i := 0; i < threads; i++ {
		for j, f := range tests {
			go func(name string, f func(string)) {
				defer callstack.Trace(884763262987)()
				defer wg.Done()
				f(name)
			}(fmt.Sprintf("%v-%v", i, j), f)
		}
	}
	wg.Wait()
	testDoneWG.Wait()
	close(stop)
}
func TestKubernetes13058_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(884763262985)()
	var testDoneWG sync.WaitGroup

	controller := NewInformer(ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			defer callstack.Trace(884763262986)()
			testDoneWG.Done()
		},
	})

	stop := make(chan struct{})
	go controller.Run(stop)

	tests := []func(string){
		func(name string) {},
	}

	const threads = 3
	var wg sync.WaitGroup
	wg.Add(threads * len(tests))
	testDoneWG.Add(threads * len(tests))
	for i := 0; i < threads; i++ {
		for j, f := range tests {
			go func(name string, f func(string)) {
				defer callstack.Trace(884763262987)()
				defer wg.Done()
				f(name)
			}(fmt.Sprintf("%v-%v", i, j), f)
		}
	}
	wg.Wait()
	testDoneWG.Wait()
	close(stop)
}
