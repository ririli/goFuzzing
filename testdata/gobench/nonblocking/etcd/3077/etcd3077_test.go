package etcd3077

import (
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type raftNode struct {
	s       *EtcdServer
	stopped chan struct{}
	done    chan struct{}
}

func (r *raftNode) run() {
	defer callstack.Trace(141733920769)()
	r.stopped = make(chan struct{})
	r.done = make(chan struct{})
	defer r.stop()
	for {
		select {
		case <-r.stopped:
			return
		}
	}
}

func (r *raftNode) stop() {
	defer callstack.Trace(141733920770)()
	close(r.done)
}

type EtcdServer struct {
	r    raftNode
	done chan struct{}
	stop chan struct{}
}

func (s *EtcdServer) run() {
	defer callstack.Trace(

		// Wait s.r.run
		141733920771)()
	go s.r.run()

	time.Sleep(10 * time.Millisecond)
	defer func() {
		defer callstack.Trace(141733920772)()
		s.r.stopped <- struct{}{}
		<-s.r.done
		close(s.done)
	}()

	for {
		select {
		case <-s.stop:
			return
		}
	}
}

func (s *EtcdServer) start() {
	defer callstack.Trace(141733920773)()
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	go s.run()
}

func (s *EtcdServer) Stop() {
	defer callstack.Trace(141733920774)()
	select {
	case s.stop <- struct{}{}:
	case <-s.done:
		return
	}
	<-s.done
}

func TestEtcd3077(t *testing.T) {
	defer callstack.Trace(141733920775)()
	srv := &EtcdServer{
		r: raftNode{},
	}
	srv.start()
	defer srv.Stop()
}
func TestEtcd3077_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(141733920775)()
	srv := &EtcdServer{
		r: raftNode{},
	}
	srv.start()
	defer srv.Stop()
}
