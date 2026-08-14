package gort

import (
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
)

func TestGort(t *testing.T) {
	root()
}

func root() {
	go func(_parentGid uint64) {
		goroutine.Enter(13826375697288921089, _parentGid)
		defer goroutine.Exit(13826375697288921089)
		parent()
	}(goroutine.CurrentGid())
	dosomething()
}
func grandchild() {
	dosomething()
}
func child() {
	go func(_parentGid uint64) {
		goroutine.Enter(13826375697288921090, _parentGid)
		defer goroutine.Exit(13826375697288921090)
		grandchild()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(13826375697288921091, _parentGid)
		defer goroutine.Exit(13826375697288921091)
		grandchild()
	}(goroutine.CurrentGid())
	dosomething()
}
func parent() {
	dosomething()
	go func(_parentGid uint64) {
		goroutine.Enter(13826375697288921092, _parentGid)
		defer goroutine.Exit(13826375697288921092)
		child()
	}(goroutine.CurrentGid())
	go func(_parentGid uint64) {
		goroutine.Enter(13826375697288921093, _parentGid)
		defer goroutine.Exit(13826375697288921093)
		child()
	}(goroutine.CurrentGid())
	dosomething()
}

func dosomething() {
	time.Sleep(time.Second)
}
func TestGort_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	root()
}
