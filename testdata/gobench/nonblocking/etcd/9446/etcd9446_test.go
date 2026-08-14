package etcd9446

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
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
	sched.InstWgBF(16105148790356312068)
	wg.Add(3)
	sched.InstWgAF(16105148790356312068, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(16105148790356312065, _parentGid)
		defer goroutine.Exit(16105148790356312065)
		func() {
			defer func() {
				sched.InstWgBF(16105148790356312069)
				wg.Done()
				sched.InstWgAF(16105148790356312069, &wg, "done")
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
				goroutine.Enter(16105148790356312066, _parentGid)
				defer goroutine.Exit(16105148790356312066)
				func() {
					defer func() {
						sched.InstWgBF(16105148790356312070)
						wg.Done()
						sched.InstWgAF(16105148790356312070, &wg, "done")
					}()
					txn.reset()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(16105148790356312067, _parentGid)
				defer goroutine.Exit(16105148790356312067)
				func() {
					defer func() {
						sched.InstWgBF(16105148790356312071)
						wg.Done()
						sched.InstWgAF(16105148790356312071, &wg, "done")
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
	sched.InstWgBF(16105148790356312068)
	wg.Add(3)
	sched.InstWgAF(16105148790356312068, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(16105148790356312065, _parentGid)
		defer goroutine.Exit(16105148790356312065)
		func() {
			defer func() {
				sched.InstWgBF(16105148790356312069)
				wg.Done()
				sched.InstWgAF(16105148790356312069, &wg, "done")
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
				goroutine.Enter(16105148790356312066, _parentGid)
				defer goroutine.Exit(16105148790356312066)
				func() {
					defer func() {
						sched.InstWgBF(16105148790356312070)
						wg.Done()
						sched.InstWgAF(16105148790356312070, &wg, "done")
					}()
					txn.reset()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(16105148790356312067, _parentGid)
				defer goroutine.Exit(16105148790356312067)
				func() {
					defer func() {
						sched.InstWgBF(16105148790356312071)
						wg.Done()
						sched.InstWgAF(16105148790356312071, &wg, "done")
					}()
					txn.UnsafeRange()
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
