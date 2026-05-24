package kubernetes82550

import (
	"testing"
	callstack "toolkit/pkg/callstack"
)

type DockerConfig map[string]DockerConfigEntry

type DockerConfigEntry struct{}

type CachingDockerConfigProvider struct {
	cacheDockerConfig DockerConfig
}

func (d *CachingDockerConfigProvider) Provide() DockerConfig {
	defer callstack.Trace(377957122049)()
	return DockerConfig{}
}

type lazyEcrProvider struct {
	actualProvider *CachingDockerConfigProvider
}

func (p *lazyEcrProvider) LazyProvide() *DockerConfigEntry {
	defer callstack.Trace(377957122050)()
	if p.actualProvider == nil {
		p.actualProvider = &CachingDockerConfigProvider{}
	}
	entry := p.actualProvider.Provide()["0"]
	return &entry
}

func TestKubernetes82550(t *testing.T) {
	defer callstack.Trace(377957122051)()
	provider := &lazyEcrProvider{}
	for i := 0; i < 10; i++ {
		go provider.LazyProvide()
	}
}
func TestKubernetes82550_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(377957122051)()
	provider := &lazyEcrProvider{}
	for i := 0; i < 10; i++ {
		go provider.LazyProvide()
	}
}
