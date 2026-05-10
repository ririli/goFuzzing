package myTest

import (
	sched "sched"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

var (
	sum = 1
)

func all() {

}
func ab() {
	defer callstack.Trace(661424963585)()

	sum++
	bc()
	all()
}
func bc() {
	defer callstack.Trace(661424963586)()

	sum++
	cd()
}
func cd() {
	defer callstack.Trace(661424963587)()

	sum++
}
func AB() {
	defer callstack.Trace(661424963588)()

	sum++
	BC()
	all()
}
func BC() {
	defer callstack.Trace(661424963589)()

	sum++
}

func Test1(t *testing.T) {
	defer callstack.Trace(661424963590)()

	go AB()

	go ab()

	time.Sleep(1 * time.Second)
}
func Test1_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	defer callstack.Trace(661424963590)()
	go AB()

	go ab()

	time.Sleep(1 * time.Second)
}
