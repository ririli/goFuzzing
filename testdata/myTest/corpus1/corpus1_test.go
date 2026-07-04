package corpus1

import (
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

var ch1 = make(chan int, 1)

func father() {
	go func(_parentGid uint64) {
		goroutine.Enter(107374182401, _parentGid)
		defer goroutine.Exit(107374182401)
		son1()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(107374182402, _parentGid)
		defer goroutine.Exit(107374182402)
		son2()
	}(goroutine.CurrentGid())
	<-ch1
	go func(_parentGid uint64) {
		goroutine.Enter(107374182403, _parentGid)
		defer goroutine.Exit(107374182403)
		son3()
	}(goroutine.CurrentGid())
}

func son1() {
	time.Sleep(10 * time.Millisecond)
}

func son2() {
	time.Sleep(10 * time.Millisecond)
	sched.InstChBF(107374182409)
	ch1 <- 1
	sched.InstChAF(107374182409, ch1, "send")
}
func son3() {
	time.Sleep(10 * time.Millisecond)
}

func grandFather() {
	go func(_parentGid uint64) {
		goroutine.Enter(107374182404, _parentGid)
		defer goroutine.Exit(107374182404)
		Father()
	}(goroutine.CurrentGid())
}
func Father() {
	time.Sleep(10 * time.Millisecond)
	go func(_parentGid uint64) {
		goroutine.Enter(107374182405, _parentGid)
		defer goroutine.Exit(107374182405)
		func() {
			time.Sleep(10 * time.Millisecond)
		}()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(107374182406, _parentGid)
		defer goroutine.Exit(107374182406)
		func() {
			time.Sleep(10 * time.Millisecond)
		}()
	}(goroutine.CurrentGid())
}

func TestA(t *testing.T) {

	go func(_parentGid uint64) {
		goroutine.Enter(107374182407, _parentGid)
		defer goroutine.Exit(107374182407)
		father()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(107374182408, _parentGid)
		defer goroutine.Exit(107374182408)
		grandFather()
	}(goroutine.CurrentGid())
}
func TestA_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	go func(_parentGid uint64) {
		goroutine.Enter(107374182407, _parentGid)
		defer goroutine.Exit(107374182407)
		father()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(107374182408, _parentGid)
		defer goroutine.Exit(107374182408)
		grandFather()
	}(goroutine.CurrentGid())
}
