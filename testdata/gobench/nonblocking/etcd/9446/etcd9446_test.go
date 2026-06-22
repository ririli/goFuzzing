package etcd9446

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type txBuffer struct {
	buckets map[string]struct{}
}

func (txb *txBuffer) reset() {
	for k, _ := range txb.buckets {
		delete(txb.buckets, k)
	}
}

type txReadBuffer struct{ txBuffer }

func (txr *txReadBuffer) Range() {
	_ = txr.buckets["1"]
}

type readTx struct {
	buf txReadBuffer
}

func (rt *readTx) reset() {
	rt.buf.reset()
}

func (rt *readTx) UnsafeRange() {
	rt.buf.Range()
}

func TestEtcd9446(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		goroutine.Enter(545460846593)
		defer goroutine.Exit(545460846593)
		func() {
			defer wg.Done()
			txn := &readTx{
				buf: txReadBuffer{
					txBuffer{
						buckets: make(map[string]struct{}),
					},
				},
			}
			txn.buf.buckets["1"] = struct{}{}
			go func() {
				goroutine.Enter(545460846594)
				defer goroutine.Exit(545460846594)
				func() {
					defer wg.Done()
					txn.reset()
				}()
			}()
			go func() {
				goroutine.Enter(545460846595)
				defer goroutine.Exit(545460846595)
				func() {
					defer wg.Done()
					txn.UnsafeRange()
				}()
			}()
		}()
	}()
	wg.Wait()
}
func TestEtcd9446_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		goroutine.Enter(545460846593)
		defer goroutine.Exit(545460846593)
		func() {
			defer wg.Done()
			txn := &readTx{
				buf: txReadBuffer{
					txBuffer{
						buckets: make(map[string]struct{}),
					},
				},
			}
			txn.buf.buckets["1"] = struct{}{}
			go func() {
				goroutine.Enter(545460846594)
				defer goroutine.Exit(545460846594)
				func() {
					defer wg.Done()
					txn.reset()
				}()
			}()
			go func() {
				goroutine.Enter(545460846595)
				defer goroutine.Exit(545460846595)
				func() {
					defer wg.Done()
					txn.UnsafeRange()
				}()
			}()
		}()
	}()
	wg.Wait()
}
