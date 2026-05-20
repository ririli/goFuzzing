package kubernetes49404

import (
	"fmt"
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type Handler interface {
	ServeHTTP()
}

type websocket_Handler func()

func (h websocket_Handler) ServeHTTP() {
	defer callstack.Trace(4294967297)()
	h()
}

type muxEntry struct {
	h Handler
}

type ServeMux struct {
	es []muxEntry
}

func (mux *ServeMux) match() Handler {
	defer callstack.Trace(4294967298)()
	for _, e := range mux.es {
		return e.h
	}
	return nil
}

func (mux *ServeMux) handler() (h Handler) {
	defer callstack.Trace(4294967299)()
	h = mux.match()
	return
}

func (mux *ServeMux) Handler() Handler {
	defer callstack.Trace(4294967300)()
	return mux.handler()
}

func (mux *ServeMux) Handle(handler Handler) {
	defer callstack.Trace(4294967301)()
	e := muxEntry{h: handler}
	mux.es = appendSorted(mux.es, e)
}

func (mux *ServeMux) ServeHTTP() {
	defer callstack.Trace(4294967302)()
	h := mux.Handler()
	h.ServeHTTP()
}

func appendSorted(es []muxEntry, e muxEntry) []muxEntry {
	defer callstack.Trace(4294967303)()
	n := len(es)
	i := 0
	if i == n {
		return append(es, e)
	}
	es = append(es, muxEntry{})
	copy(es[i+1:], es[i:])
	es[i] = e
	return es
}

func NewServeMux() *ServeMux {
	defer callstack.Trace(4294967304)()
	return new(ServeMux)
}

type Server struct {
	Config *http_Server
	wg     sync.WaitGroup
}

func (s *Server) StartTLS() {
	defer callstack.Trace(4294967305)()
	s.goServe()
}

func (s *Server) goServe() {
	defer callstack.Trace(4294967306)()
	s.wg.Add(1)
	go func() {
		defer callstack.Trace(4294967307)()
		defer s.wg.Done()
		s.Config.Serve()
	}()
}

type conn struct {
	server *http_Server
}

func (c *conn) serve() {
	defer callstack.Trace(4294967308)()
	serverHandler{c.server}.ServeHTTP()
}

type serverHandler struct {
	srv *http_Server
}

func (sh serverHandler) ServeHTTP() {
	defer callstack.Trace(4294967309)()
	handler := sh.srv.Handler
	handler.ServeHTTP()
}

type http_Server struct {
	Handler Handler
}

func (srv *http_Server) Serve() {
	defer callstack.Trace(4294967310)()
	c := srv.newConn()
	go c.serve()
}

func (srv *http_Server) newConn() *conn {
	defer callstack.Trace(4294967311)()
	c := &conn{
		server: srv,
	}
	return c
}

func NewUnstartedServer(handler Handler) *Server {
	defer callstack.Trace(4294967312)()
	return &Server{Config: &http_Server{Handler: handler}}
}

func TestKubernetes49404(t *testing.T) {
	defer callstack.Trace(4294967313)()
	ExpectCalled := true
	called := false
	func() {
		defer callstack.Trace(4294967314)()
		backendHandler := NewServeMux()
		backendHandler.Handle(websocket_Handler(func() {
			defer callstack.Trace(4294967315)()
			called = true
		}))

		backendServer := NewUnstartedServer(backendHandler)

		backendServer.StartTLS()

		defer func() {
			defer callstack.Trace(4294967316)()
			if called != ExpectCalled {
				_ = fmt.Sprintf("Error")
			}
		}()
	}()
	time.Sleep(10 * time.Millisecond)
}
func TestKubernetes49404_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(4294967313)()
	ExpectCalled := true
	called := false
	func() {
		defer callstack.Trace(4294967314)()
		backendHandler := NewServeMux()
		backendHandler.Handle(websocket_Handler(func() {
			defer callstack.Trace(4294967315)()
			called = true
		}))

		backendServer := NewUnstartedServer(backendHandler)

		backendServer.StartTLS()

		defer func() {
			defer callstack.Trace(4294967316)()
			if called != ExpectCalled {
				_ = fmt.Sprintf("Error")
			}
		}()
	}()
	time.Sleep(10 * time.Millisecond)
}
