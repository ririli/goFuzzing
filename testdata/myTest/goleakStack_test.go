package myTest

import (
	"fmt"
	"sync"
	"testing"
	"time"
	sched "toolkit/pkg/sched"
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

func TestGoroutineStack_1(t *testing.T) {

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
	ch := make(chan int)
	sched.InstChBF(605590388737, ch)
	ch <- 1
	sched.InstChAF(605590388737, ch)
	sched.InstChBF(605590388738, ch)
	<-ch
	sched.InstChAF(605590388738, ch)
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Done()
	wg.Wait()
}
