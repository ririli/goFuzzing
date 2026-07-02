package istio16742

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
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
	configgen.buildSidecarOutboundHTTPRouteConfig(node)
}

func (configgen *ConfigGeneratorImpl) buildSidecarOutboundHTTPRouteConfig(node *Proxy) {
	BuildSidecarVirtualHostsFromConfigAndRegistry(node.WorkloadLabels)
}

type Proxy struct {
	WorkloadLabels Collection
}

type XdsConnection struct {
	modelNode *Proxy
}

func newXdsConnection() *XdsConnection {
	return &XdsConnection{
		modelNode: &Proxy{},
	}
}

type DiscoveryServer struct {
	ConfigGenerator ConfigGenerator
}

func (s *DiscoveryServer) addCon(con *XdsConnection) {
	adsClientsMutex.Lock()
	defer adsClientsMutex.Unlock()
	adsClients["1"] = con
}

func (s *DiscoveryServer) StreamAggregatedResources() {
	con := newXdsConnection()
	s.addCon(con)
	s.pushRoute(con)
}

func (s *DiscoveryServer) generateRawRoutes(con *XdsConnection) {
	s.ConfigGenerator.BuildHTTPRoutes(con.modelNode)
}

func (s *DiscoveryServer) pushRoute(con *XdsConnection) {
	s.generateRawRoutes(con)
}

func (s *DiscoveryServer) WorkloadUpdate() {
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
	sd.EDSUpdater.WorkloadUpdate()
}

func TestIstio16742(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(120259084292)
	wg.Add(3)
	sched.InstWgAF(120259084292, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(120259084289, _parentGid)
		defer goroutine.Exit(120259084289)
		func() {
			defer func() {
				sched.InstWgBF(120259084293)
				wg.Done()
				sched.InstWgAF(120259084293, &wg, "done")
			}()
			registry := &MemServiceDiscovery{
				EDSUpdater: &DiscoveryServer{
					ConfigGenerator: &ConfigGeneratorImpl{},
				},
			}
			go func(_parentGid uint64) {
				goroutine.Enter(120259084290, _parentGid)
				defer goroutine.Exit(120259084290)
				func() {
					defer func() {
						sched.InstWgBF(120259084294)
						wg.Done()
						sched.InstWgAF(120259084294, &wg, "done")
					}()
					registry.EDSUpdater.(*DiscoveryServer).StreamAggregatedResources()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(120259084291, _parentGid)
				defer goroutine.Exit(120259084291)
				func() {
					defer func() {
						sched.InstWgBF(120259084295)
						wg.Done()
						sched.InstWgAF(120259084295, &wg, "done")
					}()
					registry.AddWorkload()
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestIstio16742_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(120259084292)
	wg.Add(3)
	sched.InstWgAF(120259084292, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(120259084289, _parentGid)
		defer goroutine.Exit(120259084289)
		func() {
			defer func() {
				sched.InstWgBF(120259084293)
				wg.Done()
				sched.InstWgAF(120259084293, &wg, "done")
			}()
			registry := &MemServiceDiscovery{
				EDSUpdater: &DiscoveryServer{
					ConfigGenerator: &ConfigGeneratorImpl{},
				},
			}
			go func(_parentGid uint64) {
				goroutine.Enter(120259084290, _parentGid)
				defer goroutine.Exit(120259084290)
				func() {
					defer func() {
						sched.InstWgBF(120259084294)
						wg.Done()
						sched.InstWgAF(120259084294, &wg, "done")
					}()
					registry.EDSUpdater.(*DiscoveryServer).StreamAggregatedResources()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(120259084291, _parentGid)
				defer goroutine.Exit(120259084291)
				func() {
					defer func() {
						sched.InstWgBF(120259084295)
						wg.Done()
						sched.InstWgAF(120259084295, &wg, "done")
					}()
					registry.AddWorkload()
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
