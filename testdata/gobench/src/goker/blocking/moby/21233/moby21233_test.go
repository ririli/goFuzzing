/*
 * Project: moby
 * Issue or PR  : https://github.com/moby/moby/pull/21233
 * Buggy version: cc12d2bfaae135e63b1f962ad80e6943dd995337
 * fix commit-id: 2f4aa9658408ac72a598363c6e22eadf93dbb8a7
 * Flaky:100/100
 * Description:
 *   This test was checking that it received every progress update that was
 *  produced. But delivery of these intermediate progress updates is not
 *  guaranteed. A new update can overwrite the previous one if the previous
 *  one hasn't been sent to the channel yet.
 *    The call to t.Fatalf exited the cur rent goroutine which was consuming
 *  the channel, which caused a deadlock and eventual test timeout rather
 *  than a proper failure message.
 */
package moby21233

import (
	"fmt"
	"math/rand"
	sched "sched"
	"sync"
	"testing"
)

type Progress struct{}

type Output interface {
	WriteProgress(Progress) error
}

type chanOutput chan<- Progress

type TransferManager struct {
	mu sync.Mutex
}

type Transfer struct {
	mu sync.Mutex
}

type Watcher struct {
	signalChan  chan struct{}
	releaseChan chan struct{}
	running     chan struct{}
}

func ChanOutput(progressChan chan<- Progress) Output {
	return chanOutput(progressChan)
}
func (out chanOutput) WriteProgress(p Progress) error {
	sched.InstChBF(51539607553, out)
	out <- p
	sched.InstChAF(51539607553, out)
	return nil
}
func NewTransferManager() *TransferManager {
	return &TransferManager{}
}
func NewTransfer() *Transfer {
	return &Transfer{}
}
func (t *Transfer) Release(watcher *Watcher) {
	sched.InstMutexBF(51539607563, &t.mu)
	t.mu.Lock()
	sched.InstMutexAF(51539607563, &t.mu)
	sched.InstMutexBF(51539607564, &t.mu)
	t.mu.Unlock()
	sched.InstMutexAF(51539607564, &t.mu)
	close(watcher.releaseChan)
	sched.InstChBF(51539607555, watcher.running)
	<-watcher.running
	sched.InstChAF(51539607555, watcher.running)
}
func (t *Transfer) Watch(progressOutput Output) *Watcher {
	sched.InstMutexBF(51539607565, &t.mu)
	t.mu.Lock()
	sched.InstMutexAF(51539607565, &t.mu)
	defer func() {
		sched.InstMutexBF(51539607566, &t.mu)
		t.mu.Unlock()
		sched.InstMutexAF(51539607566, &t.mu)
	}()
	lastProgress := Progress{}
	w := &Watcher{
		releaseChan: make(chan struct{}),
		signalChan:  make(chan struct{}),
		running:     make(chan struct{}),
	}
	go func() { // G2
		defer func() {
			close(w.running)
		}()
		done := false
		for {
			sched.InstMutexBF(51539607567, &t.mu)
			t.mu.Lock()
			sched.InstMutexAF(51539607567, &t.mu)
			sched.InstMutexBF(51539607568, &t.mu)
			t.mu.Unlock()
			sched.InstMutexAF(51539607568, &t.mu)
			if rand.Int31n(2) >= 1 {
				progressOutput.WriteProgress(lastProgress)
			}
			if done {
				return
			}
			select {
			case <-w.signalChan:
				sched.InstChAF(51539607561, w.signalChan)
			case <-w.releaseChan:
				sched.InstChAF(51539607562, w.releaseChan)
				done = true
				select {
				default:
				}
			}
		}
	}()
	return w
}
func (tm *TransferManager) Transfer(progressOutput Output) (*Transfer, *Watcher) {
	sched.InstMutexBF(51539607569, &tm.mu)
	tm.mu.Lock()
	sched.InstMutexAF(51539607569, &tm.mu)
	defer func() {
		sched.InstMutexBF(51539607570, &tm.mu)
		tm.mu.Unlock()
		sched.InstMutexAF(51539607570, &tm.mu)
	}()
	t := NewTransfer()
	return t, t.Watch(progressOutput)
}

func testTransfer() {
	tm := NewTransferManager()
	progressChan := make(chan Progress)
	progressDone := make(chan struct{})
	go func() { // G3
		for p := range progressChan { /// Chan consumer
			if rand.Int31n(2) >= 1 {
				return
			}
			fmt.Println(p)
		}
		sched.InstChBF(51539607558, progressDone)
		close(progressDone)
		sched.InstChAF(51539607558, progressDone)
	}()
	ids := []string{"id1", "id2", "id3"}
	xrefs := make([]*Transfer, len(ids))
	watchers := make([]*Watcher, len(ids))
	for i := range ids {
		xrefs[i], watchers[i] = tm.Transfer(ChanOutput(progressChan)) /// Chan producer
	}

	for i := range xrefs {
		xrefs[i].Release(watchers[i]) /// Block here
	}
	sched.InstChBF(51539607559, progressChan)
	close(progressChan)
	sched.InstChAF(51539607559, progressChan)
	sched.InstChBF(51539607560, progressDone)
	<-progressDone
	sched.

		///
		/// G1 						G2					G3
		/// testTransfer()
		/// tm.Transfer()
		/// t.Watch()
		/// 						WriteProgress()
		/// 						ProgressChan<-
		/// 											<-progressChan
		/// 						...					...
		/// 						return
		/// 											<-progressChan
		/// <-watcher.running
		/// ----------------------G1, G3 leak--------------------------
		///
		InstChAF(51539607560, progressDone)
}

func TestMoby21233(t *testing.T) {
	go testTransfer() // G1
}
func TestMoby21233_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	go testTransfer()
}
