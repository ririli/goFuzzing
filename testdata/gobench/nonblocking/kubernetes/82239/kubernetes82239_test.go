package kubernetes82239

import (
	"strconv"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type ObjectMeta struct {
	Annotations map[string]struct{}
}

func (in *ObjectMeta) DeepCopyInto(out *ObjectMeta) {
	defer callstack.Trace(472446402561)()
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]struct{}, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

type PersistentVolume struct {
	ObjectMeta
}

func (in *PersistentVolume) DeepCopy() *PersistentVolume {
	defer callstack.Trace(472446402562)()
	out := new(PersistentVolume)
	in.DeepCopyInto(out)
	return out
}

func (in *PersistentVolume) DeepCopyInto(out *PersistentVolume) {
	defer callstack.Trace(472446402563)()
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}

func newVolume() *PersistentVolume {
	defer callstack.Trace(472446402564)()
	volume := PersistentVolume{ObjectMeta{}}
	volume.Annotations = make(map[string]struct{})
	for i := 0; i < 2; i++ {
		volume.Annotations[strconv.Itoa(i)] = struct{}{}
	}
	return &volume
}

func newVolumeArray() []*PersistentVolume {
	defer callstack.Trace(472446402565)()
	return []*PersistentVolume{
		newVolume(),
	}
}

func volumesWithAnnotation(volumes []*PersistentVolume) []*PersistentVolume {
	defer callstack.Trace(472446402566)()
	return volumes
}

type testCall func(test controllerTest)

type controllerTest struct {
	initialVolumes []*PersistentVolume
	test           testCall
}

func Until(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(472446402567)()
	JitterUntil(f, stopCh)
}

func JitterUntil(f func(), stopCh <-chan struct{}) {
	defer callstack.Trace(472446402568)()
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		func() {
			defer callstack.Trace(472446402569)()
			f()
		}()
		select {
		case <-stopCh:
			return
		default:
		}
	}
}

type SimplifiedLister struct {
	volume *PersistentVolume
}

func (s *SimplifiedLister) Get(key string) *PersistentVolume {
	defer callstack.Trace(472446402570)()
	return s.volume
}

type PersistentVolumeController struct {
	volumeLister *SimplifiedLister
}

func (ctrl *PersistentVolumeController) Run(stopCh <-chan struct{}) {
	defer callstack.Trace(472446402571)()
	go Until(ctrl.volumeWorker, stopCh)
}

func (ctrl *PersistentVolumeController) volumeWorker() {
	defer callstack.Trace(472446402572)()
	workFunc := func() {
		defer callstack.Trace(472446402573)()
		volume := ctrl.volumeLister.Get("0")
		ctrl.updateVolume(volume)
	}
	workFunc()
}

func (ctrl *PersistentVolumeController) updateVolume(volume *PersistentVolume) {
	defer callstack.Trace(472446402574)()
	ctrl.syncVolume(volume)
}

func (ctrl *PersistentVolumeController) syncVolume(volume *PersistentVolume) {
	defer callstack.Trace(472446402575)()
	ctrl.updateVolumePhase(volume)
}

func (ctrl *PersistentVolumeController) updateVolumePhase(volume *PersistentVolume) {
	defer callstack.Trace(472446402576)()
	volume.DeepCopy()
}

func newTestController() *PersistentVolumeController {
	defer callstack.Trace(472446402577)()
	return &PersistentVolumeController{}
}

func TestKubernetes82239(t *testing.T) {
	defer callstack.Trace(472446402578)()
	tests := []controllerTest{
		{
			initialVolumes: volumesWithAnnotation(newVolumeArray()),
			test: func(test controllerTest) {
				defer callstack.Trace(472446402579)()
				test.initialVolumes[0].Annotations["0"] = struct{}{}
			},
		},
	}

	for _, test := range tests {
		ctrl := newTestController()

		lister := &SimplifiedLister{
			volume: test.initialVolumes[0],
		}
		ctrl.volumeLister = lister

		stopCh := make(chan struct{})
		go ctrl.Run(stopCh)
		time.Sleep(1 * time.Millisecond)
		test.test(test)
		close(stopCh)
	}
}
func TestKubernetes82239_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(472446402578)()
	tests := []controllerTest{
		{
			initialVolumes: volumesWithAnnotation(newVolumeArray()),
			test: func(test controllerTest) {
				defer callstack.Trace(472446402579)()
				test.initialVolumes[0].Annotations["0"] = struct{}{}
			},
		},
	}

	for _, test := range tests {
		ctrl := newTestController()

		lister := &SimplifiedLister{
			volume: test.initialVolumes[0],
		}
		ctrl.volumeLister = lister

		stopCh := make(chan struct{})
		go ctrl.Run(stopCh)
		time.Sleep(1 * time.Millisecond)
		test.test(test)
		close(stopCh)
	}
}
