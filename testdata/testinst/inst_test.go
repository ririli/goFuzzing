package testinst

import (
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

func A() {
	defer callstack.Trace(158913789953)()
	defer callstack.Trace(158913789953)()

	B()
}
func B() {

}
func a() {
	defer callstack.Trace(158913789954)()
	defer callstack.Trace(158913789954)()

	b()
}
func b() {

}
func TestA(t *testing.T) {
	defer callstack.Trace(158913789955)()
	defer callstack.Trace(158913789955)()

	go A()
	go a()
	time.Sleep(1 * time.Second)
}
func TestA_1(t *testing.T) {
	defer callstack.Trace(158913789956)()
	callstack.ParseInput()
	defer callstack.Trace(158913789955)()
	go A()
	go a()
	time.Sleep(1 * time.Second)
}
func TestA_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(158913789955)()
	defer callstack.Trace(158913789955)()

	go A()
	go a()
	time.Sleep(1 * time.Second)
}
