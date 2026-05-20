package serving3148

import (
	"fmt"
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
	"toolkit/pkg/sched/goleak"
)

type PodAutoscalerInterface interface {
	Create()
}

type PodAutoscalersGetter interface {
	PodAutoscalers() PodAutoscalerInterface
}

type AutoscalingV1alpha1Interface interface {
	PodAutoscalersGetter
}

type clientset_Interface interface {
	AutoscalingV1alpha1() AutoscalingV1alpha1Interface
}

type FakeAutoscalingV1alpha1 struct {
	*Fake
}

func (c *FakeAutoscalingV1alpha1) PodAutoscalers() PodAutoscalerInterface {
	defer callstack.Trace(184683593729)()
	return &FakePodAutoscalers{c}
}

type Clientset struct {
	Fake
}

func (c *Clientset) AutoscalingV1alpha1() AutoscalingV1alpha1Interface {
	defer callstack.Trace(184683593730)()
	return &FakeAutoscalingV1alpha1{Fake: &c.Fake}
}

type FakePodAutoscalers struct {
	Fake *FakeAutoscalingV1alpha1
}

func (c *FakePodAutoscalers) Create() {
	defer callstack.Trace(184683593731)()
	c.Fake.Invokes()
}

type Reconciler struct {
	ServingClientSet clientset_Interface
}

func (c *Reconciler) Reconcile() {
	defer callstack.Trace(184683593732)()
	c.reconcile()
}

func (c *Reconciler) reconcile() {
	defer callstack.Trace(184683593733)()
	phases := []struct {
		name string
		f    func()
	}{{
		name: "KPA",
		f:    c.reconcileKPA,
	}}
	for _, phase := range phases {
		phase.f()
	}
}

func (c *Reconciler) reconcileKPA() {
	defer callstack.Trace(184683593734)()
	c.createKPA()
}

func (c *Reconciler) createKPA() {
	defer callstack.Trace(184683593735)()
	c.ServingClientSet.AutoscalingV1alpha1().PodAutoscalers().Create()
}

type controller_Reconciler interface {
	Reconcile()
}

type Impl struct {
	controller_Reconciler controller_Reconciler
}

func (c *Impl) Run(threadiness int) {
	defer callstack.Trace(184683593736)()
	sg := sync.WaitGroup{}
	defer sg.Wait()

	for i := 0; i < threadiness; i++ {
		sg.Add(1)
		go func() {
			defer callstack.Trace(184683593737)()
			defer sg.Done()
			c.processNextWorkItem()
		}()
	}
}

func (c *Impl) processNextWorkItem() {
	defer callstack.Trace(184683593738)()
	c.controller_Reconciler.Reconcile()
}

func NewImpl(r controller_Reconciler) *Impl {
	defer callstack.Trace(184683593739)()
	return &Impl{
		controller_Reconciler: r,
	}
}

func NewController() *Impl {
	defer callstack.Trace(184683593740)()
	c := &Reconciler{}
	return NewImpl(c)
}

type Group struct {
	wg      sync.WaitGroup
	errOnce sync.Once
}

func (g *Group) Wait() {
	defer callstack.Trace(184683593741)()
	g.wg.Wait()
}

func (g *Group) Go(f func()) {
	defer callstack.Trace(184683593742)()
	g.wg.Add(1)
	go func() {
		defer callstack.Trace(184683593743)()
		defer g.wg.Done()
		f()
	}()
}

type Hooks struct{}

func NewHooks() *Hooks {
	defer callstack.Trace(184683593744)()
	return &Hooks{}
}
func (h *Hooks) OnUpdate(fake *Fake) {
	defer callstack.Trace(184683593745)()
	fake.PrependReactor()
}

type Reactor interface{}

type SimpleReactor struct{}

type Fake struct {
	ReactionChain []Reactor
}

func (c *Fake) Invokes() {
	defer callstack.Trace(184683593746)()
	for _ = range c.ReactionChain {
	}
}

func (c *Fake) PrependReactor() {
	defer callstack.Trace(184683593747)()
	c.ReactionChain = append([]Reactor{&SimpleReactor{}})
}

func TestServing3148(t *testing.T) {
	defer callstack.Trace(184683593748)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(184683593749)()
		defer wg.Done()
		cs := &Clientset{}
		controller := NewController()
		controller.controller_Reconciler.(*Reconciler).ServingClientSet = cs
		eg := &Group{}
		defer func() {
			defer callstack.Trace(184683593750)()
			eg.Wait()
		}()
		eg.Go(func() { defer callstack.Trace(184683593751)(); controller.Run(1) })
		h := NewHooks()
		h.OnUpdate(&cs.Fake)
	}()
	wg.Wait()
	stacks := goleak.All()
	for _, stack := range stacks {
		fmt.Println("id:", stack.ID(), "state:", stack.State(), "firstFunc:", stack.FirstFunction())
		fmt.Println(stack.Full())
	}
}
func TestServing3148_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(184683593748)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(184683593749)()
		defer wg.Done()
		cs := &Clientset{}
		controller := NewController()
		controller.controller_Reconciler.(*Reconciler).ServingClientSet = cs
		eg := &Group{}
		defer func() {
			defer callstack.Trace(184683593750)()
			eg.Wait()
		}()
		eg.Go(func() { defer callstack.Trace(184683593751)(); controller.Run(1) })
		h := NewHooks()
		h.OnUpdate(&cs.Fake)
	}()
	wg.Wait()
	stacks := goleak.All()
	for _, stack := range stacks {
		fmt.Println("id:", stack.ID(), "state:", stack.State(), "firstFunc:", stack.FirstFunction())
		fmt.Println(stack.Full())
	}
}
