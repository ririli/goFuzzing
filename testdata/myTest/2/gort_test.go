package two

import (
	"fmt"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	"toolkit/pkg/sched"
)

var a int

func A() {
	a++
}
func B() {
	go func() {
		goroutine.Enter(103079215105)
		defer goroutine.Exit(103079215105)
		func() {
			a--
		}()
	}()
}

func TestGort(t *testing.T) {

	go func() {
		goroutine.Enter(103079215106)
		defer goroutine.Exit(103079215106)
		func() {
			fmt.Println("hello world")
			go func() {
				goroutine.Enter(103079215107)
				defer goroutine.Exit(103079215107)
				A()
			}()
		}()
	}()

	go func() {
		goroutine.Enter(103079215108)
		defer goroutine.Exit(103079215108)
		B()
	}()
}
func TestGort_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	go func() {
		goroutine.Enter(103079215106)
		defer goroutine.Exit(103079215106)
		func() {
			fmt.Println("hello world")
			go func() {
				goroutine.Enter(103079215107)
				defer goroutine.Exit(103079215107)
				A()
			}()
		}()
	}()

	go func() {
		goroutine.Enter(103079215108)
		defer goroutine.Exit(103079215108)
		B()
	}()
}
