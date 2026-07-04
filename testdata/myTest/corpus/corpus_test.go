package corpus

import (
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

func closeCh(ch chan int) {
	time.Sleep(time.Millisecond * 10)
	sched.InstChBF(55834574851)
	close(ch)
	sched.InstChAF(55834574851, ch, "close")
}

func sendCh(ch chan int) {
	sched.InstChBF(55834574852)
	ch <- 1
	sched.InstChAF(55834574852, ch, "send")
}

func TestCorpus(t *testing.T) {

	ch := make(chan int, 1)
	go func(_parentGid uint64) {
		goroutine.Enter(55834574849, _parentGid)
		defer goroutine.Exit(55834574849)
		closeCh(ch)
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(55834574850, _parentGid)
		defer goroutine.Exit(55834574850)
		sendCh(ch)
	}(goroutine.CurrentGid())
}
func TestCorpus_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	ch := make(chan int, 1)
	go func(_parentGid uint64) {
		goroutine.Enter(55834574849, _parentGid)
		defer goroutine.Exit(55834574849)
		closeCh(ch)
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(55834574850, _parentGid)
		defer goroutine.Exit(55834574850)
		sendCh(ch)
	}(goroutine.CurrentGid())
}
