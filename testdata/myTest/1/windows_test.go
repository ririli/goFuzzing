package a_test

import (
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

var a int

func A() {
	defer callstack.Trace(313532612609)()
	a++
}

func TestA(t *testing.T) {
	defer callstack.Trace(313532612610)()
	go A()
}
func TestA_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(313532612610)()
	go A()
	time.Sleep(1 * time.Second)
}
