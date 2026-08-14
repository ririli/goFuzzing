package kubernetes70892

import (
	"context"
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
)

type HostPriorityList []int

type DoWorkPieceFunc func(piece int)

func ParallelizeUntil(ctx context.Context, workers, pieces int, doWorkPiece DoWorkPieceFunc) {
	var stop <-chan struct{}
	if ctx != nil {
		stop = ctx.Done()
	}

	toProcess := make(chan int, pieces)
	for i := 0; i < pieces; i++ {
		sched.InstChBF(413985407965855746)
		toProcess <- i
		sched.InstChAF(413985407965855746, toProcess, "send")
	}
	sched.InstChBF(413985407965855747)
	close(toProcess)
	sched.InstChAF(413985407965855747, toProcess, "close")

	if pieces < workers {
		workers = pieces
	}

	wg := sync.WaitGroup{}
	sched.InstWgBF(413985407965855748)
	wg.Add(workers)
	sched.InstWgAF(413985407965855748, &wg, "add")
	for i := 0; i < workers; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(413985407965855745, _parentGid)
			defer goroutine.Exit(413985407965855745)
			func() {
				defer func() {
					sched.InstWgBF(413985407965855749)
					wg.Done()
					sched.InstWgAF(413985407965855749, &wg, "done")
				}()
				for piece := range toProcess {
					select {
					case <-stop:
						return
					default:
						doWorkPiece(piece)
					}
				}
			}()
		}(goroutine.CurrentGid())
	}
	wg.Wait()
}

func TestKubernetes70892(t *testing.T) {
	priorityConfigs := append([]int{}, 1, 2, 3)
	results := make([]HostPriorityList, len(priorityConfigs), len(priorityConfigs))

	for i := range priorityConfigs {
		results[i] = make(HostPriorityList, 2)
	}
	processNode := func(index int) {
		for i := range priorityConfigs {
			if results[i][0] != 4 {
				results[i] = HostPriorityList{7, 8, 9}
			}
		}
	}
	ParallelizeUntil(context.Background(), 2, 2, processNode)
}
func TestKubernetes70892_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	priorityConfigs := append([]int{}, 1, 2, 3)
	results := make([]HostPriorityList, len(priorityConfigs), len(priorityConfigs))

	for i := range priorityConfigs {
		results[i] = make(HostPriorityList, 2)
	}
	processNode := func(index int) {
		for i := range priorityConfigs {
			if results[i][0] != 4 {
				results[i] = HostPriorityList{7, 8, 9}
			}
		}
	}
	ParallelizeUntil(context.Background(), 2, 2, processNode)
}
