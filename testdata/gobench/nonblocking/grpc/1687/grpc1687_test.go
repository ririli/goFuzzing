package grpc1687

import (
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type ResponseWriter interface {
}

type testHandlerResponseWriter struct {
}

func newTestHandlerResponseWriter() ResponseWriter {
	defer callstack.Trace(450971566081)()
	return testHandlerResponseWriter{}
}

type serverHandlerTransport struct {
	closedCh chan struct{}
	writes   chan func()
}

func (ht *serverHandlerTransport) do(fn func()) {
	defer callstack.Trace(450971566082)()
	select {
	case <-ht.closedCh:
		return
	default:
		select {
		case ht.writes <- fn:
			return
		case <-ht.closedCh:
			return
		}
	}
}

func (ht *serverHandlerTransport) WriteStatus() {
	defer callstack.Trace(450971566083)()
	ht.do(func() {})
	close(ht.writes)
}

func (ht *serverHandlerTransport) Write() {
	defer callstack.Trace(450971566084)()
	ht.do(func() {})
}

func (ht *serverHandlerTransport) runStream() {
	defer callstack.Trace(450971566085)()
	for {
		select {
		case fn, ok := <-ht.writes:
			if !ok {
				return
			}
			fn()
		case <-ht.closedCh:
			return
		}
	}
}

func (ht *serverHandlerTransport) HandleStreams(startStream func()) {
	defer callstack.Trace(450971566086)()
	startStream()

	ht.runStream()
}

type ServerTransport interface {
	HandleStreams(func())
	Write()
	WriteStatus()
}

func NewServerHandlerTransport(writer ResponseWriter) ServerTransport {
	defer callstack.Trace(450971566087)()
	st := &serverHandlerTransport{
		closedCh: make(chan struct{}),
		writes:   make(chan func()),
	}
	return st
}

type handleStreamTest struct {
	t  *testing.T
	rw testHandlerResponseWriter
	ht *serverHandlerTransport
}

func newHandleStreamTest(t *testing.T) *handleStreamTest {
	defer callstack.Trace(450971566088)()
	rw := newTestHandlerResponseWriter().(testHandlerResponseWriter)
	ht := NewServerHandlerTransport(rw)
	return &handleStreamTest{
		t:  t,
		rw: rw,
		ht: ht.(*serverHandlerTransport),
	}
}

func testHandlerTransportHandleStreams(t *testing.T, handleStream func(st *handleStreamTest)) {
	defer callstack.Trace(450971566089)()
	st := newHandleStreamTest(t)
	st.ht.HandleStreams(func() { defer callstack.Trace(450971566090)(); go handleStream(st) })
}

func TestGrpc1687(t *testing.T) {
	defer callstack.Trace(450971566091)()
	testHandlerTransportHandleStreams(t, func(st *handleStreamTest) {
		defer callstack.Trace(450971566092)()
		st.ht.WriteStatus()
		st.ht.Write()
	})
	time.Sleep(10 * time.Millisecond)
}
func TestGrpc1687_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(450971566091)()
	testHandlerTransportHandleStreams(t, func(st *handleStreamTest) {
		defer callstack.Trace(450971566092)()
		st.ht.WriteStatus()
		st.ht.Write()
	})
	time.Sleep(10 * time.Millisecond)
}
