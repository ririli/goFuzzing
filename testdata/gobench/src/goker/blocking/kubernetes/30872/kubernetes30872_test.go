package kubernetes30872

import (
	sched "sched"
	"sync"
	"testing"
)

type PopProcessFunc func()

type ProcessFunc func()

func Util(f func(), stopCh <-chan struct{}) {
	JitterUntil(f, stopCh)
}

func JitterUntil(f func(), stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			sched.InstChAF(261993005061, stopCh)
			return
		default:
		}
		func() {
			f()
		}()
	}
}

type Queue interface {
	HasSynced()
	Pop(PopProcessFunc)
}

type Config struct {
	Queue
	Process ProcessFunc
}

type Controller struct {
	config Config
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	Util(c.processLoop, stopCh)
}

func (c *Controller) HasSynced() {
	c.config.Queue.HasSynced()
}

func (c *Controller) processLoop() {
	c.config.Queue.Pop(PopProcessFunc(c.config.Process))
}

type ControllerInterface interface {
	Run(<-chan struct{})
	HasSynced()
}

type ResourceEventHandler interface {
	OnAdd()
}

type ResourceEventHandlerFuncs struct {
	AddFunc func()
}

func (r ResourceEventHandlerFuncs) OnAdd() {
	if r.AddFunc != nil {
		r.AddFunc()
	}
}

type informer struct {
	controller ControllerInterface

	stopChan chan struct{}
}

type federatedInformerImpl struct {
	sync.Mutex
	clusterInformer informer
}

func (f *federatedInformerImpl) ClustersSynced() {
	sched.InstMutexBF(261993005062, &f)
	f.Lock()
	sched.InstMutexAF(261993005062, &f)
	defer func() {
		sched.InstMutexBF(261993005063, &f)
		f.Unlock()
		sched.InstMutexAF(261993005063, &f)
	}()
	f.clusterInformer.controller.HasSynced()
}

func (f *federatedInformerImpl) addCluster() {
	sched.InstMutexBF(261993005064, &f)
	f.Lock()
	sched.InstMutexAF(261993005064, &f)
	defer func() {
		sched.InstMutexBF(261993005065, &f)
		f.Unlock()
		sched.InstMutexAF(261993005065, &f)
	}()
}

func (f *federatedInformerImpl) Start() {
	sched.InstMutexBF(261993005066, &f)
	f.Lock()
	sched.InstMutexAF(261993005066, &f)
	defer func() {
		sched.InstMutexBF(261993005067, &f)
		f.Unlock()
		sched.InstMutexAF(261993005067, &f)
	}()

	f.clusterInformer.stopChan = make(chan struct{})
	go f.clusterInformer.controller.Run(f.clusterInformer.stopChan)
}

func (f *federatedInformerImpl) Stop() {
	sched.InstMutexBF(261993005068, &f)
	f.Lock()
	sched.InstMutexAF(261993005068, &f)
	defer func() {
		sched.InstMutexBF(261993005069, &f)
		f.Unlock()
		sched.InstMutexAF(261993005069, &f)
	}()
	close(f.clusterInformer.stopChan)
}

type DelayingDeliverer struct{}

func (d *DelayingDeliverer) StartWithHandler(handler func()) {
	go func() {
		handler()
	}()
}

type FederationView interface {
	ClustersSynced()
}

type FederatedInformer interface {
	FederationView
	Start()
	Stop()
}

type NamespaceController struct {
	namespaceDeliverer         *DelayingDeliverer
	namespaceFederatedInformer FederatedInformer
}

func (nc *NamespaceController) isSynced() {
	nc.namespaceFederatedInformer.ClustersSynced()
}

func (nc *NamespaceController) reconcileNamespace() {
	nc.isSynced()
}

func (nc *NamespaceController) Run(stopChan <-chan struct{}) {
	nc.namespaceFederatedInformer.Start()
	go func() {
		sched.InstChBF(261993005059, stopChan)
		<-stopChan
		sched.InstChAF(261993005059, stopChan)
		nc.namespaceFederatedInformer.Stop()
	}()
	nc.namespaceDeliverer.StartWithHandler(func() {
		nc.reconcileNamespace()
	})
}

type DeltaFIFO struct {
	lock sync.RWMutex
}

func (f *DeltaFIFO) HasSynced() {
	sched.InstMutexBF(261993005070, &f.lock)
	f.lock.Lock()
	sched.InstMutexAF(261993005070, &f.lock)
	defer func() {
		sched.InstMutexBF(261993005071, &f.lock)
		f.lock.Unlock()
		sched.InstMutexAF(261993005071, &f.lock)
	}()
}

func (f *DeltaFIFO) Pop(process PopProcessFunc) {
	sched.InstMutexBF(261993005072, &f.lock)
	f.lock.Lock()
	sched.InstMutexAF(261993005072, &f.lock)
	defer func() {
		sched.InstMutexBF(261993005073, &f.lock)
		f.lock.Unlock()
		sched.InstMutexAF(261993005073, &f.lock)
	}()
	process()
}

func NewFederatedInformer() FederatedInformer {
	federatedInformer := &federatedInformerImpl{}
	federatedInformer.clusterInformer.controller = NewInformer(
		ResourceEventHandlerFuncs{
			AddFunc: func() {
				federatedInformer.addCluster()
			},
		})
	return federatedInformer
}

func NewInformer(h ResourceEventHandler) *Controller {
	fifo := &DeltaFIFO{}
	cfg := &Config{
		Queue: fifo,
		Process: func() {
			h.OnAdd()
		},
	}
	return &Controller{config: *cfg}
}

func NewNamespaceController() *NamespaceController {
	nc := &NamespaceController{}
	nc.namespaceDeliverer = &DelayingDeliverer{}
	nc.namespaceFederatedInformer = NewFederatedInformer()
	return nc
}

func TestKubernetes30872_bad_test(t *testing.T) {
	namespaceController := NewNamespaceController()
	stop := make(chan struct{})
	namespaceController.Run(stop)
	sched.InstChBF(261993005060, stop)
	close(stop)
	sched.InstChAF(261993005060, stop)
}
func TestKubernetes30872_bad_test_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	namespaceController := NewNamespaceController()
	stop := make(chan struct{})
	namespaceController.Run(stop)
	sched.InstChBF(261993005060, stop)
	close(stop)
	sched.InstChAF(261993005060, stop)
}
