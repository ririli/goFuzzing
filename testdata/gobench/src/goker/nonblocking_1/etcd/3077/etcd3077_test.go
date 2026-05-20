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
	defer callstack.Trace(1005022347265)()
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
	defer callstack.Trace(1005022347266)()
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
		1005022347267)()
	go s.r.run()

	time.Sleep(10 * time.Millisecond)
	defer func() {
		defer callstack.Trace(1005022347268)()
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
	defer callstack.Trace(1005022347269)()
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	go s.run()
}

func (s *EtcdServer) Stop() {
	defer callstack.Trace(1005022347270)()
	select {
	case s.stop <- struct{}{}:
	case <-s.done:
		return
	}
	<-s.done
}

func TestEtcd3077_1(t *testing.T) {
	defer callstack.Trace(1005022347271)()
	srv := &EtcdServer{
		r: raftNode{},
	}
	srv.start()
	defer srv.Stop()
}
