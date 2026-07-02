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
	sched.InstWgBF(545460846596)
	wg.Add(3)
	sched.InstWgAF(545460846596, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(545460846593, _parentGid)
		defer goroutine.Exit(545460846593)
		func() {
			defer func() {
				sched.InstWgBF(545460846597)
				wg.Done()
				sched.InstWgAF(545460846597, &wg, "done")
			}()
			txn := &readTx{
				buf: txReadBuffer{
					txBuffer{
						buckets: make(map[string]struct{}),
					},
				},
			}
			txn.buf.buckets["1"] = struct{}{}
			go func(_parentGid uint64) {
				goroutine.Enter(545460846594, _parentGid)
				defer goroutine.Exit(545460846594)
				func() {
					defer func() {
						sched.InstWgBF(545460846598)
						wg.Done()
						sched.InstWgAF(545460846598, &wg, "done")
					}()
					txn.reset()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(545460846595, _parentGid)
				defer goroutine.Exit(545460846595)
				func() {
					defer func() {
						sched.InstWgBF(545460846599)
						wg.Done()
						sched.InstWgAF(545460846599, &wg, "done")
					}()
					txn.UnsafeRange()
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestEtcd9446_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(545460846596)
	wg.Add(3)
	sched.InstWgAF(545460846596, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(545460846593, _parentGid)
		defer goroutine.Exit(545460846593)
		func() {
			defer func() {
				sched.InstWgBF(545460846597)
				wg.Done()
				sched.InstWgAF(545460846597, &wg, "done")
			}()
			txn := &readTx{
				buf: txReadBuffer{
					txBuffer{
						buckets: make(map[string]struct{}),
					},
				},
			}
			txn.buf.buckets["1"] = struct{}{}
			go func(_parentGid uint64) {
				goroutine.Enter(545460846594, _parentGid)
				defer goroutine.Exit(545460846594)
				func() {
					defer func() {
						sched.InstWgBF(545460846598)
						wg.Done()
						sched.InstWgAF(545460846598, &wg, "done")
					}()
					txn.reset()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(545460846595, _parentGid)
				defer goroutine.Exit(545460846595)
				func() {
					defer func() {
						sched.InstWgBF(545460846599)
						wg.Done()
						sched.InstWgAF(545460846599, &wg, "done")
					}()
					txn.UnsafeRange()
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
