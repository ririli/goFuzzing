package kubernetes82550

import (
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type DockerConfig map[string]DockerConfigEntry

type DockerConfigEntry struct{}

type CachingDockerConfigProvider struct {
	cacheDockerConfig DockerConfig
}

func (d *CachingDockerConfigProvider) Provide() DockerConfig {
	return DockerConfig{}
}

type lazyEcrProvider struct {
	actualProvider *CachingDockerConfigProvider
}

func (p *lazyEcrProvider) LazyProvide() *DockerConfigEntry {
	if p.actualProvider == nil {
		p.actualProvider = &CachingDockerConfigProvider{}
	}
	entry := p.actualProvider.Provide()["0"]
	return &entry
}

func TestKubernetes82550(t *testing.T) {
	provider := &lazyEcrProvider{}
	for i := 0; i < 10; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(13814836215381229569, _parentGid)
			defer goroutine.Exit(13814836215381229569)
			provider.LazyProvide()
		}(goroutine.CurrentGid())
	}
}
func TestKubernetes82550_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	provider := &lazyEcrProvider{}
	for i := 0; i < 10; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(13814836215381229569, _parentGid)
			defer goroutine.Exit(13814836215381229569)
			provider.LazyProvide()
		}(goroutine.CurrentGid())
	}
}
