package cockroach35931

import (
	sched "sched"
	"sync"
	"testing"
)

type RowReceiver interface {
	Push()
}

type inboundStreamInfo struct {
	receiver RowReceiver
}

type RowChannel struct {
	dataChan chan struct{}
}

func (rc *RowChannel) Push() {
	sched.InstChBF(1052266987521, rc.dataChan)
	rc.dataChan <- struct{}{}
	sched.InstChAF(1052266987521, rc.dataChan)
}

func (rc *RowChannel) initWithBufSizeAndNumSenders(chanBufSize int) {
	rc.dataChan = make(chan struct{}, chanBufSize)
}

type flowEntry struct {
	flow           *Flow
	inboundStreams map[int]*inboundStreamInfo
}

type flowRegistry struct {
	sync.Mutex
	flows map[int]*flowEntry
}

func (fr *flowRegistry) getEntryLocked(id int) *flowEntry {
	entry, ok := fr.flows[id]
	if !ok {
		entry = &flowEntry{}
		fr.flows[id] = entry
	}
	return entry
}

func (fr *flowRegistry) cancelPendingStreamsLocked(id int) []RowReceiver {
	entry := fr.flows[id]
	pendingReceivers := make([]RowReceiver, 0)
	for _, is := range entry.inboundStreams {
		pendingReceivers = append(pendingReceivers, is.receiver)
	}
	return pendingReceivers
}

type Flow struct {
	id             int
	flowRegistry   *flowRegistry
	inboundStreams map[int]*inboundStreamInfo
}

func (f *Flow) cancel() {
	sched.InstMutexBF(1052266987522, &f.flowRegistry)
	f.flowRegistry.Lock()
	sched.InstMutexAF(1052266987522, &f.flowRegistry)
	timedOutReceivers := f.flowRegistry.cancelPendingStreamsLocked(f.id)
	sched.InstMutexBF(1052266987523, &f.flowRegistry)
	f.flowRegistry.Unlock()
	sched.InstMutexAF(1052266987523, &f.flowRegistry)

	for _, receiver := range timedOutReceivers {
		receiver.Push()
	}
}

func (fr *flowRegistry) RegisterFlow(f *Flow, inboundStreams map[int]*inboundStreamInfo) {
	entry := fr.getEntryLocked(f.id)
	entry.flow = f
	entry.inboundStreams = inboundStreams
}

func makeFlowRegistry() *flowRegistry {
	return &flowRegistry{
		flows: make(map[int]*flowEntry),
	}
}

func TestCockroach35931(t *testing.T) {
	fr := makeFlowRegistry()

	left := &RowChannel{}
	left.initWithBufSizeAndNumSenders(1)
	right := &RowChannel{}
	right.initWithBufSizeAndNumSenders(1)

	inboundStreams := map[int]*inboundStreamInfo{
		0: {
			receiver: left,
		},
		1: {
			receiver: right,
		},
	}

	left.Push()

	flow := &Flow{
		id:             0,
		flowRegistry:   fr,
		inboundStreams: inboundStreams,
	}

	fr.RegisterFlow(flow, inboundStreams)

	flow.cancel()
}
func TestCockroach35931_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	fr := makeFlowRegistry()

	left := &RowChannel{}
	left.initWithBufSizeAndNumSenders(1)
	right := &RowChannel{}
	right.initWithBufSizeAndNumSenders(1)

	inboundStreams := map[int]*inboundStreamInfo{
		0: {
			receiver: left,
		},
		1: {
			receiver: right,
		},
	}

	left.Push()

	flow := &Flow{
		id:             0,
		flowRegistry:   fr,
		inboundStreams: inboundStreams,
	}

	fr.RegisterFlow(flow, inboundStreams)

	flow.cancel()
}
