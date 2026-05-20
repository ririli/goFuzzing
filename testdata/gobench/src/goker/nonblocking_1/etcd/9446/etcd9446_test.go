package etcd9446

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type txBuffer struct {
	buckets map[string]struct{}
}

func (txb *txBuffer) reset() {
	defer callstack.Trace(158913789953)()
	for k, _ := range txb.buckets {
		delete(txb.buckets, k)
	}
}

type txReadBuffer struct{ txBuffer }

func (txr *txReadBuffer) Range() {
	defer callstack.Trace(158913789954)()
	_ = txr.buckets["1"]
}

type readTx struct {
	buf txReadBuffer
}

func (rt *readTx) reset() {
	defer callstack.Trace(158913789955)()
	rt.buf.reset()
}

func (rt *readTx) UnsafeRange() {
	defer callstack.Trace(158913789956)()
	rt.buf.Range()
}

func TestEtcd9446(t *testing.T) {
	defer callstack.Trace(158913789957)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(158913789958)()
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
			defer callstack.Trace(158913789959)()
			defer wg.Done()
			txn.reset()
		}()
		go func() {
			defer callstack.Trace(158913789960)()
			defer wg.Done()
			txn.UnsafeRange()
		}()
	}()
	wg.Wait()
}
func TestEtcd9446_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(158913789957)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(158913789958)()
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
			defer callstack.Trace(158913789959)()
			defer wg.Done()
			txn.reset()
		}()
		go func() {
			defer callstack.Trace(158913789960)()
			defer wg.Done()
			txn.UnsafeRange()
		}()
	}()
	wg.Wait()
}
