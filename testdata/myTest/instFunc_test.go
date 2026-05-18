package myTest

import (
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

var (
	sum = 1
)

func all() {
	defer callstack.Trace(420906795008)()
	sum++
}
func ab() {
	defer callstack.Trace(420906795009)()

	sum++
	bc()
	all()
}
func bc() {
	defer callstack.Trace(420906795010)()

	sum++
	cd()
}
func cd() {
	defer callstack.Trace(420906795011)()

	sum++
}
func AB() {
	defer callstack.Trace(420906795012)()

	sum++
	BC()
	all()
	time.Sleep(1 * time.Second)
}
func BC() {
	defer callstack.Trace(420906795013)()

	sum++

	time.Sleep(1 * time.Second)
}

func TestA(t *testing.T) {
	defer callstack.Trace(420906795014)()

	go AB()

	go ab()

	time.Sleep(1 * time.Second)
}
func TestA_1(t *testing.T) {
	// todo 加一个输出到控制台的函数
	defer func() {
		callstack.PrintTrees()
		callstack.PrintSusConPairs()
	}()
	defer callstack.Trace(420906795014)()
	go AB()

	go ab()

	time.Sleep(3 * time.Second)
}
