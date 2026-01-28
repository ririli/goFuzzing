package grpc795

import (
	sched "sched"
	"sync"
	"testing"
)

type Server struct {
	mu    sync.Mutex
	drain bool
}

func (s *Server) GracefulStop() {
	sched.InstMutexBF(1030792151041, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(1030792151041, &s.mu)
	if s.drain == true {
		sched.InstMutexBF(1030792151042, &s.mu)
		s.mu.Lock()
		sched.InstMutexAF(1030792151042,

			// Missing Unlock
			&s.mu)
		return
	}
	s.drain = true
}

func (s *Server) Serve() {
	sched.InstMutexBF(1030792151043, &s.mu)
	s.mu.Lock()
	sched.InstMutexAF(1030792151043, &s.mu)
	sched.InstMutexBF(1030792151044, &s.mu)
	s.mu.Unlock()
	sched.InstMutexAF(1030792151044, &s.mu)
}

func NewServer() *Server {
	return &Server{}
}

type test struct {
	srv *Server
}

func (te *test) startServer() {
	s := NewServer()
	te.srv = s
	go s.Serve()
}

func newTest() *test {
	return &test{}
}

func testServerGracefulStopIdempotent() {
	te := newTest()

	te.startServer()

	for i := 0; i < 3; i++ {
		te.srv.GracefulStop()
	}
}

func TestGrpc795(t *testing.T) {
	testServerGracefulStopIdempotent()
}
func TestGrpc795_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	testServerGracefulStopIdempotent()
}
