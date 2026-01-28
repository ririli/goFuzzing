package cockroach2448

import (
	sched "sched"
	"testing"
	"time"
)

type Stopper struct {
	Done chan bool
}

func (s *Stopper) ShouldStop() <-chan bool {
	return s.Done
}

type EventMembershipChangeCommitted struct {
	Callback func()
}
type MultiRaft struct {
	stopper      *Stopper
	Events       chan interface{}
	callbackChan chan func()
}

// sendEvent can be invoked many times
func (m *MultiRaft) sendEvent(event interface{}) {
	/// FIX:
	/// Let event append a event queue instead of pending here
	select {
	case m.Events <- event:
		sched. // Waiting for events consumption
			InstChAF(51539607559, m.Events)
	case <-m.stopper.ShouldStop():
		sched.InstChAF(51539607560, m.stopper.ShouldStop())
	}
}

type state struct {
	*MultiRaft
}

func (s *state) start() {
	for {
		select {
		case <-s.stopper.ShouldStop():
			sched.InstChAF(51539607561, s.stopper.ShouldStop())
			return
		case cb := <-s.callbackChan:
			sched.InstChAF(51539607562, s.callbackChan)
			cb()
		default:
			s.handleWriteResponse()
		}
	}
}
func (s *state) handleWriteResponse() {
	s.processCommittedEntry()
}

func (s *state) processCommittedEntry() {
	s.sendEvent(&EventMembershipChangeCommitted{
		Callback: func() {
			select {
			case s.callbackChan <- func() { // Waiting for callbackChan consumption
				time.Sleep(time.Nanosecond)
			}:
				sched.InstChAF(51539607563, s.callbackChan)

			case <-s.stopper.ShouldStop():
				sched.InstChAF(51539607564, s.stopper.ShouldStop())
			}
		},
	})
}

type Store struct {
	multiraft *MultiRaft
}

func (s *Store) processRaft() {
	for {
		select {
		case e := <-s.multiraft.Events:
			sched.InstChAF(51539607565, s.multiraft.Events)
			var callback func()
			switch e := e.(type) {
			case *EventMembershipChangeCommitted:
				callback = e.Callback
				if callback != nil {
					callback() // Waiting for callbackChan consumption
				}
			}
		case <-s.multiraft.stopper.ShouldStop():
			sched.InstChAF(51539607566, s.multiraft.stopper.ShouldStop())
			return
		}
	}
}

func NewStoreAndState() (*Store, *state) {
	stopper := &Stopper{
		Done: make(chan bool),
	}
	mltrft := &MultiRaft{
		stopper:      stopper,
		Events:       make(chan interface{}),
		callbackChan: make(chan func()),
	}
	st := &state{mltrft}
	s := &Store{mltrft}
	return s, st
}

func TestCockroach2448(t *testing.T) {
	s, st := NewStoreAndState()
	go s.processRaft() // G1
	go st.start()      // G2
}
func TestCockroach2448_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	s, st := NewStoreAndState()
	go s.processRaft()
	go st.start()
}
