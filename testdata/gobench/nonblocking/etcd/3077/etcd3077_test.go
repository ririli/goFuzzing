package etcd3077

import (
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type raftNode struct {
	s       *EtcdServer
	stopped chan struct{}
	done    chan struct{}
}

func (r *raftNode) run() {
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
	close(r.done)
}

type EtcdServer struct {
	r    raftNode
	done chan struct{}
	stop chan struct{}
}

func (s *EtcdServer) run() {
	go func(_parentGid uint64) {
		// Wait s.r.run
		goroutine.Enter(141733920769, _parentGid)
		defer goroutine.Exit(141733920769)
		s.r.run()
	}(goroutine.CurrentGid())

	time.Sleep(10 * time.Millisecond)
	defer func() {
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
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	go func(_parentGid uint64) {
		goroutine.Enter(141733920770, _parentGid)
		defer goroutine.Exit(141733920770)
		s.run()
	}(goroutine.CurrentGid())
}

func (s *EtcdServer) Stop() {
	select {
	case s.stop <- struct{}{}:
		sched.InstChSelectAF(141733920773, s.stop, "send")
	case <-s.done:
		return
	}
	<-s.done
}

func TestEtcd3077(t *testing.T) {
	srv := &EtcdServer{
		r: raftNode{},
	}
	srv.start()
	defer srv.Stop()
}
func TestEtcd3077_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	srv := &EtcdServer{
		r: raftNode{},
	}
	srv.start()
	defer srv.Stop()
}
