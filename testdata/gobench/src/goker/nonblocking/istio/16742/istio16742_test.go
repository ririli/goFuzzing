package istio16742

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

var (
	adsClients      = map[string]*XdsConnection{}
	adsClientsMutex sync.RWMutex
)

type Collection []struct{}

func BuildSidecarVirtualHostsFromConfigAndRegistry(proxyLabels Collection) {}

type ConfigGenerator interface {
	BuildHTTPRoutes(node *Proxy)
}

type ConfigGeneratorImpl struct{}

func (configgen *ConfigGeneratorImpl) BuildHTTPRoutes(node *Proxy) {
	defer callstack.Trace(403726925825)()
	configgen.buildSidecarOutboundHTTPRouteConfig(node)
}

func (configgen *ConfigGeneratorImpl) buildSidecarOutboundHTTPRouteConfig(node *Proxy) {
	defer callstack.Trace(403726925826)()
	BuildSidecarVirtualHostsFromConfigAndRegistry(node.WorkloadLabels)
}

type Proxy struct {
	WorkloadLabels Collection
}

type XdsConnection struct {
	modelNode *Proxy
}

func newXdsConnection() *XdsConnection {
	defer callstack.Trace(403726925827)()
	return &XdsConnection{
		modelNode: &Proxy{},
	}
}

type DiscoveryServer struct {
	ConfigGenerator ConfigGenerator
}

func (s *DiscoveryServer) addCon(con *XdsConnection) {
	defer callstack.Trace(403726925828)()
	adsClientsMutex.Lock()
	defer adsClientsMutex.Unlock()
	adsClients["1"] = con
}

func (s *DiscoveryServer) StreamAggregatedResources() {
	defer callstack.Trace(403726925829)()
	con := newXdsConnection()
	s.addCon(con)
	s.pushRoute(con)
}

func (s *DiscoveryServer) generateRawRoutes(con *XdsConnection) {
	defer callstack.Trace(403726925830)()
	s.ConfigGenerator.BuildHTTPRoutes(con.modelNode)
}

func (s *DiscoveryServer) pushRoute(con *XdsConnection) {
	defer callstack.Trace(403726925831)()
	s.generateRawRoutes(con)
}

func (s *DiscoveryServer) WorkloadUpdate() {
	defer callstack.Trace(403726925832)()
	adsClientsMutex.RLock()
	for _, connection := range adsClients {
		connection.modelNode.WorkloadLabels = nil
	}
	adsClientsMutex.RUnlock()
}

type XDSUpdater interface {
	WorkloadUpdate()
}

type MemServiceDiscovery struct {
	EDSUpdater XDSUpdater
}

func (sd *MemServiceDiscovery) AddWorkload() {
	defer callstack.Trace(403726925833)()
	sd.EDSUpdater.WorkloadUpdate()
}

func TestIstio16742(t *testing.T) {
	defer callstack.Trace(403726925834)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(403726925835)()
		defer wg.Done()
		registry := &MemServiceDiscovery{
			EDSUpdater: &DiscoveryServer{
				ConfigGenerator: &ConfigGeneratorImpl{},
			},
		}
		go func() {
			defer callstack.Trace(403726925836)()
			defer wg.Done()
			registry.EDSUpdater.(*DiscoveryServer).StreamAggregatedResources()
		}()
		go func() {
			defer callstack.Trace(403726925837)()
			defer wg.Done()
			registry.AddWorkload()
		}()
	}()
	wg.Wait()
}
func TestIstio16742_1(t *testing.T) {
	defer callstack.Trace(403726925834)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(403726925835)()
		defer wg.Done()
		registry := &MemServiceDiscovery{
			EDSUpdater: &DiscoveryServer{
				ConfigGenerator: &ConfigGeneratorImpl{},
			},
		}
		go func() {
			defer callstack.Trace(403726925836)()
			defer wg.Done()
			registry.EDSUpdater.(*DiscoveryServer).StreamAggregatedResources()
		}()
		go func() {
			defer callstack.Trace(403726925837)()
			defer wg.Done()
			registry.AddWorkload()
		}()
	}()
	wg.Wait()
}
