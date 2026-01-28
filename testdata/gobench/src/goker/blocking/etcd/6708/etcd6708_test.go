package etcd6708

import (
	"context"
	sched "sched"
	"sync"
	"testing"
)

type EndpointSelectionMode int

const (
	EndpointSelectionRandom EndpointSelectionMode = iota
	EndpointSelectionPrioritizeLeader
)

type MembersAPI interface {
	Leader(ctx context.Context)
}

type Client interface {
	Sync(ctx context.Context)
	SetEndpoints()
	httpClient
}

type httpClient interface {
	Do(context.Context)
}

type httpClusterClient struct {
	sync.RWMutex
	selectionMode EndpointSelectionMode
}

func (c *httpClusterClient) getLeaderEndpoint() {
	mAPI := NewMembersAPI(c)
	mAPI.Leader(context.Background())
}

func (c *httpClusterClient) SetEndpoints() {
	switch c.selectionMode {
	case EndpointSelectionRandom:
	case EndpointSelectionPrioritizeLeader:
		c.getLeaderEndpoint()
	}
}

func (c *httpClusterClient) Do(ctx context.Context) {
	sched.InstMutexBF(
		// block here
		416611827713, &c)
	c.RLock()
	sched.InstMutexAF(416611827713, &c)
	sched.InstMutexBF(416611827714, &c)
	c.RUnlock()
	sched.InstMutexAF(416611827714, &c)
}

func (c *httpClusterClient) Sync(ctx context.Context) {
	sched.InstMutexBF(416611827715, &c)
	c.Lock()
	sched.InstMutexAF(416611827715, &c)
	defer func() {
		sched.InstMutexBF(416611827716, &c)
		c.Unlock()
		sched.InstMutexAF(416611827716, &c)
	}()

	c.SetEndpoints()
}

type httpMembersAPI struct {
	client httpClient
}

func (m *httpMembersAPI) Leader(ctx context.Context) {
	m.client.Do(ctx)
}

func NewMembersAPI(c Client) MembersAPI {
	return &httpMembersAPI{
		client: c,
	}
}
func TestEtcd6708(t *testing.T) {
	hc := &httpClusterClient{
		selectionMode: EndpointSelectionPrioritizeLeader,
	}
	hc.Sync(context.Background())
}
func TestEtcd6708_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	hc := &httpClusterClient{
		selectionMode: EndpointSelectionPrioritizeLeader,
	}
	hc.Sync(context.Background())
}
