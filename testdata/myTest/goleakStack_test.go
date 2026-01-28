package myTest

import (
	"fmt"
	"testing"
	"time"
	"toolkit/pkg/sched/goleak"
)

type Num struct {
	numInt int
}

func (receiver *Num) add() {
	receiver.numInt++
	receiver.print()
}
func (receiver *Num) sub() {
	receiver.numInt--
	receiver.print()
}
func (receiver *Num) print() {
	fmt.Println(receiver.numInt)
}
func TestGoroutineStack(t *testing.T) {
	stacks := goleak.All()
	for _, stack := range stacks {
		fmt.Println("id:", stack.ID(), "state:", stack.State(), "firstFunc:", stack.FirstFunction())
		fmt.Println(stack.Full())
	}
	num := new(Num)

	go func() {
		num.add()
		time.Sleep(3 * time.Second)
		go num.sub()
	}()
	num.add()
	num.sub()
	time.Sleep(1 * time.Second)

	stacks = goleak.All()
	for _, stack := range stacks {
		fmt.Println("id:", stack.ID(), "state:", stack.State(), "firstFunc:", stack.FirstFunction())
		fmt.Println(stack.Full())
	}
}
