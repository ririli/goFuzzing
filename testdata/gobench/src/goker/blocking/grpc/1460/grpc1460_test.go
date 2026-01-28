/*
 * Project: grpc
 * Issue or PR  : https://github.com/grpc/grpc-go/pull/1460
 * Buggy version: 7db1564ba1229bc42919bb1f6d9c4186f3aa8678
 * fix commit-id: e605a1ecf24b634f94f4eefdab10a9ada98b70dd
 * Flaky: 100/100
 * Description:
 *   When gRPC keepalives are enabled (which isn't the case
 * by default at this time) and PermitWithoutStream is false
 * (the default), the client can deadlock when transitioning
 * between having no active stream and having one active
 * stream.The keepalive() goroutine is stuck at “<-t.awakenKeepalive”,
 * while the main goroutine is stuck in NewStream() on t.mu.Lock().
 */
package grpc1460

import (
	sched "sched"
	"sync"
	"testing"
)

type Stream struct{}

type http2Client struct {
	mu              sync.Mutex
	awakenKeepalive chan struct{}
	activeStream    []*Stream
}

func (t *http2Client) keepalive() {
	sched.InstMutexBF(8589934596, &t.mu)
	t.mu.Lock()
	sched.InstMutexAF(8589934596, &t.mu)
	if len(t.activeStream) < 1 {
		sched.InstChBF(8589934593, t.awakenKeepalive)
		<-t.awakenKeepalive
		sched.InstChAF(8589934593, t.awakenKeepalive)
		sched.InstMutexBF(8589934597, &t.mu)
		t.mu.Unlock()
		sched.InstMutexAF(8589934597, &t.mu)
	} else {
		sched.InstMutexBF(8589934598, &t.mu)
		t.mu.Unlock()
		sched.InstMutexAF(8589934598, &t.mu)
	}
}

func (t *http2Client) NewStream() {
	sched.InstMutexBF(8589934599, &t.mu)
	t.mu.Lock()
	sched.InstMutexAF(8589934599, &t.mu)
	t.activeStream = append(t.activeStream, &Stream{})
	if len(t.activeStream) == 1 {
		select {
		case t.awakenKeepalive <- struct{}{}:
			sched.
				// FIX: t.awakenKeepalive <- struct{}{}
				InstChAF(8589934595, t.awakenKeepalive)

		default:
		}
	}
	sched.InstMutexBF(8589934600,

		///
		/// G1 						G2
		/// client.keepalive()
		/// 						client.NewStream()
		/// t.mu.Lock()
		/// <-t.awakenKeepalive
		/// 						t.mu.Lock()
		/// ---------------G1, G2 deadlock--------------
		///
		&t.mu)
	t.mu.Unlock()
	sched.InstMutexAF(8589934600, &t.mu)
}

func TestGrpc1460(t *testing.T) {
	client := &http2Client{
		awakenKeepalive: make(chan struct{}),
	}
	go client.keepalive() //G1
	go client.NewStream() //G2
}
func TestGrpc1460_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	client := &http2Client{
		awakenKeepalive: make(chan struct{}),
	}
	go client.keepalive()
	go client.NewStream()
}
