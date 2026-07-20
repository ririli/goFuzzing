package kubernetes81091

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type FakeFilterPlugin struct {
	numFilterCalled int
}

func (fp *FakeFilterPlugin) Filter() {
	fp.numFilterCalled++
}

type FilterPlugin interface {
	Filter()
}

type Framework interface {
	RunFilterPlugins()
}

type framework struct {
	filterPlugins []FilterPlugin
}

func NewFramework() Framework {
	f := &framework{}
	f.filterPlugins = append(f.filterPlugins, &FakeFilterPlugin{})
	return f
}

func (f *framework) RunFilterPlugins() {
	for _, pl := range f.filterPlugins {
		pl.Filter()
	}
}

type genericScheduler struct {
	framework Framework
}

func NewGenericScheduler(framework Framework) *genericScheduler {
	return &genericScheduler{
		framework: framework,
	}
}

func (g *genericScheduler) findNodesThatFit() {
	checkNode := func(i int) {
		g.framework.RunFilterPlugins()
	}
	ParallelizeUntil(2, 2, checkNode)
}

func (g *genericScheduler) Schedule() {
	g.findNodesThatFit()
}

type DoWorkPieceFunc func(piece int)

func ParallelizeUntil(workers, pieces int, doWorkPiece DoWorkPieceFunc) {
	var stop <-chan struct{}

	toProcess := make(chan int, pieces)
	for i := 0; i < pieces; i++ {
		sched.InstChBF(15209918116554342403)
		toProcess <- i
		sched.InstChAF(15209918116554342403, toProcess, "send")
	}
	sched.InstChBF(15209918116554342404)
	close(toProcess)
	sched.InstChAF(15209918116554342404, toProcess, "close")

	if pieces < workers {
		workers = pieces
	}

	wg := sync.WaitGroup{}
	sched.InstWgBF(15209918116554342405)
	wg.Add(workers)
	sched.InstWgAF(15209918116554342405, &wg, "add")
	for i := 0; i < workers; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(15209918116554342401, _parentGid)
			defer goroutine.Exit(15209918116554342401)
			func() {
				defer func() {
					sched.InstWgBF(15209918116554342406)
					wg.Done()
					sched.InstWgAF(15209918116554342406, &wg, "done")
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

func TestKubernetes81091(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(15209918116554342407)
	wg.Add(1)
	sched.InstWgAF(15209918116554342407, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(15209918116554342402, _parentGid)
		defer goroutine.Exit(15209918116554342402)
		func() {
			defer func() {
				sched.InstWgBF(15209918116554342408)
				wg.Done()
				sched.InstWgAF(15209918116554342408, &wg, "done")
			}()
			filterFramework := NewFramework()
			scheduler := NewGenericScheduler(filterFramework)
			scheduler.Schedule()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestKubernetes81091_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(15209918116554342407)
	wg.Add(1)
	sched.InstWgAF(15209918116554342407, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(15209918116554342402, _parentGid)
		defer goroutine.Exit(15209918116554342402)
		func() {
			defer func() {
				sched.InstWgBF(15209918116554342408)
				wg.Done()
				sched.InstWgAF(15209918116554342408, &wg, "done")
			}()
			filterFramework := NewFramework()
			scheduler := NewGenericScheduler(filterFramework)
			scheduler.Schedule()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
