package serving3148

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
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
	defer callstack.Trace(858993459201)()
	return &FakePodAutoscalers{c}
}

type Clientset struct {
	Fake
}

func (c *Clientset) AutoscalingV1alpha1() AutoscalingV1alpha1Interface {
	defer callstack.Trace(858993459202)()
	return &FakeAutoscalingV1alpha1{Fake: &c.Fake}
}

type FakePodAutoscalers struct {
	Fake *FakeAutoscalingV1alpha1
}

func (c *FakePodAutoscalers) Create() {
	defer callstack.Trace(858993459203)()
	c.Fake.Invokes()
}

type Reconciler struct {
	ServingClientSet clientset_Interface
}

func (c *Reconciler) Reconcile() {
	defer callstack.Trace(858993459204)()
	c.reconcile()
}

func (c *Reconciler) reconcile() {
	defer callstack.Trace(858993459205)()
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
	defer callstack.Trace(858993459206)()
	c.createKPA()
}

func (c *Reconciler) createKPA() {
	defer callstack.Trace(858993459207)()
	c.ServingClientSet.AutoscalingV1alpha1().PodAutoscalers().Create()
}

type controller_Reconciler interface {
	Reconcile()
}

type Impl struct {
	controller_Reconciler controller_Reconciler
}

func (c *Impl) Run(threadiness int) {
	defer callstack.Trace(858993459208)()
	sg := sync.WaitGroup{}
	defer sg.Wait()

	for i := 0; i < threadiness; i++ {
		sg.Add(1)
		go func() {
			defer callstack.Trace(858993459209)()
			defer sg.Done()
			c.processNextWorkItem()
		}()
	}
}

func (c *Impl) processNextWorkItem() {
	defer callstack.Trace(858993459210)()
	c.controller_Reconciler.Reconcile()
}

func NewImpl(r controller_Reconciler) *Impl {
	defer callstack.Trace(858993459211)()
	return &Impl{
		controller_Reconciler: r,
	}
}

func NewController() *Impl {
	defer callstack.Trace(858993459212)()
	c := &Reconciler{}
	return NewImpl(c)
}

type Group struct {
	wg      sync.WaitGroup
	errOnce sync.Once
}

func (g *Group) Wait() {
	defer callstack.Trace(858993459213)()
	g.wg.Wait()
}

func (g *Group) Go(f func()) {
	defer callstack.Trace(858993459214)()
	g.wg.Add(1)
	go func() {
		defer callstack.Trace(858993459215)()
		defer g.wg.Done()
		f()
	}()
}

type Hooks struct{}

func NewHooks() *Hooks {
	defer callstack.Trace(858993459216)()
	return &Hooks{}
}
func (h *Hooks) OnUpdate(fake *Fake) {
	defer callstack.Trace(858993459217)()
	fake.PrependReactor()
}

type Reactor interface{}

type SimpleReactor struct{}

type Fake struct {
	ReactionChain []Reactor
}

func (c *Fake) Invokes() {
	defer callstack.Trace(858993459218)()
	for _ = range c.ReactionChain {
	}
}

func (c *Fake) PrependReactor() {
	defer callstack.Trace(858993459219)()
	c.ReactionChain = append([]Reactor{&SimpleReactor{}})
}

func TestServing3148(t *testing.T) {
	defer callstack.Trace(858993459220)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(858993459221)()
		defer wg.Done()
		cs := &Clientset{}
		controller := NewController()
		controller.controller_Reconciler.(*Reconciler).ServingClientSet = cs
		eg := &Group{}
		defer func() {
			defer callstack.Trace(858993459222)()
			eg.Wait()
		}()
		eg.Go(func() { defer callstack.Trace(858993459223)(); controller.Run(1) })
		h := NewHooks()
		h.OnUpdate(&cs.Fake)
	}()
	wg.Wait()
}
func TestServing3148_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(858993459220)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(858993459221)()
		defer wg.Done()
		cs := &Clientset{}
		controller := NewController()
		controller.controller_Reconciler.(*Reconciler).ServingClientSet = cs
		eg := &Group{}
		defer func() {
			defer callstack.Trace(858993459222)()
			eg.Wait()
		}()
		eg.Go(func() { defer callstack.Trace(858993459223)(); controller.Run(1) })
		h := NewHooks()
		h.OnUpdate(&cs.Fake)
	}()
	wg.Wait()
}
