package etcd5509

import (
	"context"
	"fmt"
	sched "sched"
	"sync"
	"testing"
)

var ErrConnClosed error

type Client struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Client) Close() {
	sched.InstMutexBF(927712935939, &c.mu)
	c.mu.Lock()
	sched.InstMutexAF(927712935939, &c.mu)
	defer func() {
		sched.InstMutexBF(927712935940, &c.mu)
		c.mu.Unlock()
		sched.InstMutexAF(927712935940, &c.mu)
	}()
	if c.cancel == nil {
		return
	}
	c.cancel()
	c.cancel = nil
	sched.InstMutexBF(927712935941,

		// block here
		&c.mu)
	c.mu.Unlock()
	sched.InstMutexAF(927712935941, &c.mu)
	sched.InstMutexBF(927712935942, &c.mu)
	c.mu.Lock()
	sched.InstMutexAF(927712935942, &c.mu)
}

type remoteClient struct {
	client *Client
	mu     sync.Mutex
}

func (r *remoteClient) acquire(ctx context.Context) error {
	for {
		sched.InstMutexBF(927712935943, &r.client.mu)
		r.client.mu.RLock()
		sched.InstMutexAF(927712935943, &r.client.mu)
		closed := r.client.cancel == nil
		sched.InstMutexBF(927712935944, &r.mu)
		r.mu.Lock()
		sched.InstMutexAF(927712935944, &r.mu)
		sched.InstMutexBF(927712935945, &r.mu)
		r.mu.Unlock()
		sched.InstMutexAF(927712935945, &r.mu)
		if closed {
			return ErrConnClosed // Missing RUnlock before return
		}
		sched.InstMutexBF(927712935946, &r.client.mu)
		r.client.mu.RUnlock()
		sched.InstMutexAF(927712935946, &r.client.mu)
	}
}

type kv struct {
	rc *remoteClient
}

func (kv *kv) Get(ctx context.Context) error {
	return kv.Do(ctx)
}

func (kv *kv) Do(ctx context.Context) error {
	for {
		err := kv.do(ctx)
		if err == nil {
			return nil
		}
		return err
	}
}

func (kv *kv) do(ctx context.Context) error {
	err := kv.getRemote(ctx)
	return err
}

func (kv *kv) getRemote(ctx context.Context) error {
	return kv.rc.acquire(ctx)
}

type KV interface {
	Get(ctx context.Context) error
	Do(ctx context.Context) error
}

func NewKV(c *Client) KV {
	return &kv{rc: &remoteClient{
		client: c,
	}}
}
func TestEtcd5509(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	cli := &Client{
		ctx:    ctx,
		cancel: cancel,
	}
	kv := NewKV(cli)
	donec := make(chan struct{})
	go func() {
		defer func() {
			sched.InstChBF(927712935937, donec)
			close(donec)
			sched.InstChAF(927712935937, donec)
		}()
		err := kv.Get(context.TODO())
		if err != nil && err != ErrConnClosed {
			fmt.Println("Expect ErrConnClosed")
		}
	}()

	cli.Close()
	sched.InstChBF(927712935938, donec)
	<-donec
	sched.InstChAF(927712935938, donec)
}
func TestEtcd5509_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	ctx, cancel := context.WithCancel(context.TODO())
	cli := &Client{
		ctx:    ctx,
		cancel: cancel,
	}
	kv := NewKV(cli)
	donec := make(chan struct{})
	go func() {
		defer func() {
			sched.InstChBF(927712935937, donec)
			close(donec)
			sched.InstChAF(927712935937, donec)
		}()
		err := kv.Get(context.TODO())
		if err != nil && err != ErrConnClosed {
			fmt.Println("Expect ErrConnClosed")
		}
	}()

	cli.Close()
	sched.InstChBF(927712935938, donec)
	<-donec
	sched.InstChAF(927712935938, donec)
}
