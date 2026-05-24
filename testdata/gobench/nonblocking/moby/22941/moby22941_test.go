package moby22941

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type Conn interface {
	Write(b []byte)
}

type pipe struct {
	wrMu sync.Mutex
}

func (p *pipe) Write(b []byte) {
	defer callstack.Trace(249108103169)()
	p.wrMu.Lock()
	defer p.wrMu.Unlock()
	b = b[1:]
}

func Pipe() Conn {
	defer callstack.Trace(249108103170)()
	return &pipe{}
}

func TestMoby22941(t *testing.T) {
	defer callstack.Trace(249108103171)()
	srv := Pipe()
	tests := [][2][]byte{
		{
			[]byte("GET /foo\nHost: /var/run/docker.sock\nUser-Agent: Docker\r\n\r\n"),
			[]byte("GET /foo\nHost: \r\nConnection: close\r\nUser-Agent: Docker\r\n\r\n"),
		},
		{
			[]byte("GET /foo\nHost: /var/run/docker.sock\nUser-Agent: Docker\nFoo: Bar\r\n"),
			[]byte("GET /foo\nHost: \r\nConnection: close\r\nUser-Agent: Docker\nFoo: Bar\r\n"),
		},
	}
	for _, pair := range tests {
		go func() {
			defer callstack.Trace(249108103172)()
			srv.Write(pair[0])
		}()
	}
	time.Sleep(10 * time.Millisecond)
}
func TestMoby22941_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(249108103171)()
	srv := Pipe()
	tests := [][2][]byte{
		{
			[]byte("GET /foo\nHost: /var/run/docker.sock\nUser-Agent: Docker\r\n\r\n"),
			[]byte("GET /foo\nHost: \r\nConnection: close\r\nUser-Agent: Docker\r\n\r\n"),
		},
		{
			[]byte("GET /foo\nHost: /var/run/docker.sock\nUser-Agent: Docker\nFoo: Bar\r\n"),
			[]byte("GET /foo\nHost: \r\nConnection: close\r\nUser-Agent: Docker\nFoo: Bar\r\n"),
		},
	}
	for _, pair := range tests {
		go func() {
			defer callstack.Trace(249108103172)()
			srv.Write(pair[0])
		}()
	}
	time.Sleep(10 * time.Millisecond)
}
