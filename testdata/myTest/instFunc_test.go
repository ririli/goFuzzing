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
	defer callstack.Trace(330712481793)()

	sum++
}
func ab() {
	defer callstack.Trace(330712481794)()

	sum++
	bc()
	all()
}
func bc() {
	defer callstack.Trace(330712481795)()

	sum++
	cd()
}
func cd() {
	defer callstack.Trace(330712481796)()

	sum++
}
func AB() {
	defer callstack.Trace(330712481797)()

	sum++
	BC()
	all()
	time.Sleep(1 * time.Second)
}
func BC() {
	defer callstack.Trace(330712481798)()

	sum++

	time.Sleep(1 * time.Second)
}

func TestA(t *testing.T) {
	defer callstack.Trace(330712481799)()

	go AB()

	go ab()

	time.Sleep(1 * time.Second)
}
func TestA_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(330712481799)()
	go AB()

	go ab()

	time.Sleep(1 * time.Second)
}
