package istio16224

import (
	sched "sched"
	"sync"
	"testing"
)

type ConfigStoreCache interface {
	RegisterEventHandler(handler func())
	Run()
}

type Event int

type Handler func(Event)

type configstoreMonitor struct {
	handlers []Handler
	eventCh  chan Event
}

func (m *configstoreMonitor) Run(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			sched.InstChAF(103079215111, stop)
			if _, ok := <-m.eventCh; ok {
				close(m.eventCh)
			}
			return
		case ce, ok := <-m.eventCh:
			sched.InstChAF(103079215112, m.eventCh)
			if ok {
				m.processConfigEvent(ce)
			}
		}
	}
}

func (m *configstoreMonitor) processConfigEvent(ce Event) {
	m.applyHandlers(ce)
}

func (m *configstoreMonitor) AppendEventHandler(h Handler) {
	m.handlers = append(m.handlers, h)
}

func (m *configstoreMonitor) applyHandlers(e Event) {
	for _, f := range m.handlers {
		f(e)
	}
}
func (m *configstoreMonitor) ScheduleProcessEvent(configEvent Event) {
	sched.InstChBF(103079215107, m.eventCh)
	m.eventCh <- configEvent
	sched.InstChAF(103079215107, m.eventCh)
}

type Monitor interface {
	Run(<-chan struct{})
	AppendEventHandler(Handler)
	ScheduleProcessEvent(Event)
}

type controller struct {
	monitor Monitor
}

func (c *controller) RegisterEventHandler(f func(Event)) {
	c.monitor.AppendEventHandler(f)
}

func (c *controller) Run(stop <-chan struct{}) {
	c.monitor.Run(stop)
}

func (c *controller) Create() {
	c.monitor.ScheduleProcessEvent(Event(0))
}

func NewMonitor() Monitor {
	return NewBufferedMonitor()
}

func NewBufferedMonitor() Monitor {
	return &configstoreMonitor{
		eventCh: make(chan Event),
	}
}
func TestIstio16224(t *testing.T) {
	controller := &controller{monitor: NewMonitor()}
	done := make(chan bool)
	lock := sync.Mutex{}
	controller.RegisterEventHandler(func(event Event) {
		sched.InstMutexBF(103079215113, &lock)
		lock.Lock()
		sched.InstMutexAF(103079215113, &lock)
		defer func() {
			sched.InstMutexBF(103079215114, &lock)
			lock.Unlock()
			sched.InstMutexAF(103079215114, &lock)
		}()
		sched.InstChBF(103079215108, done)
		done <- true
		sched.InstChAF(103079215108, done)
	})

	stop := make(chan struct{})
	go controller.Run(stop)

	controller.Create()
	sched.InstMutexBF(103079215115, &lock)
	lock.Lock()
	sched.InstMutexAF(103079215115, &lock)
	sched.InstMutexBF(103079215116, &lock)
	lock.Unlock()
	sched.InstMutexAF(103079215116, &lock)
	sched.InstChBF(103079215109, done)
	<-done
	sched.InstChAF(103079215109, done)
	sched.InstChBF(103079215110, stop)
	close(stop)
	sched.InstChAF(103079215110, stop)
}
func TestIstio16224_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	controller := &controller{monitor: NewMonitor()}
	done := make(chan bool)
	lock := sync.Mutex{}
	controller.RegisterEventHandler(func(event Event) {
		sched.InstMutexBF(103079215113, &lock)
		lock.Lock()
		sched.InstMutexAF(103079215113, &lock)
		defer func() {
			sched.InstMutexBF(103079215114, &lock)
			lock.Unlock()
			sched.InstMutexAF(103079215114, &lock)
		}()
		sched.InstChBF(103079215108, done)
		done <- true
		sched.InstChAF(103079215108, done)
	})

	stop := make(chan struct{})
	go controller.Run(stop)

	controller.Create()
	sched.InstMutexBF(103079215115, &lock)
	lock.Lock()
	sched.InstMutexAF(103079215115, &lock)
	sched.InstMutexBF(103079215116, &lock)
	lock.Unlock()
	sched.InstMutexAF(103079215116, &lock)
	sched.InstChBF(103079215109, done)
	<-done
	sched.InstChAF(103079215109, done)
	sched.InstChBF(103079215110, stop)
	close(stop)
	sched.InstChAF(103079215110, stop)
}
