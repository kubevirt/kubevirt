package network

import (
	"fmt"
	"os"
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/cache"
	kfs "kubevirt.io/kubevirt/pkg/os/fs"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"
	"kubevirt.io/client-go/testutils"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestNetwork(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

func newVMIBridgeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := api2.NewMinimalVMIWithNS(namespace, name)
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func newVMIMasqueradeInterface(namespace, name, masqueradeCidr, masqueradeIpv6Cidr string) *v1.VirtualMachineInstance {
	vmi := api2.NewMinimalVMIWithNS(namespace, name)
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
	network := vmi.Spec.Networks[0]
	network.Pod.VMNetworkCIDR = masqueradeCidr
	network.Pod.VMIPv6NetworkCIDR = masqueradeIpv6Cidr
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewDomainWithBridgeInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: v1.VirtIO,
		},
		Type: "bridge",
		Source: api.InterfaceSource{
			Bridge: api.DefaultBridgeName,
		},
		Alias: api.NewUserDefinedAlias("default"),
	},
	}
	return domain
}

type tempCacheCreator struct {
	once   sync.Once
	tmpDir string
}

func (c *tempCacheCreator) New(filePath string) *cache.Cache {
	c.once.Do(func() {
		tmpDir, err := os.MkdirTemp("", "temp-cache")
		if err != nil {
			panic("Unable to create temp cache directory")
		}
		c.tmpDir = tmpDir
	})
	return cache.NewCustomCache(filePath, kfs.NewWithRootPath(c.tmpDir))
}

type configStateCacheStub struct {
	stateCache map[string]cache.PodIfaceState
	readErr    error
	writeErr   error
}

func newConfigStateCacheStub() configStateCacheStub {
	return configStateCacheStub{map[string]cache.PodIfaceState{}, nil, nil}
}

func (c configStateCacheStub) Read(key string) (cache.PodIfaceState, error) {
	return c.stateCache[key], c.readErr
}

func (c configStateCacheStub) Write(key string, state cache.PodIfaceState) error {
	if c.writeErr != nil {
		return c.writeErr
	}
	c.stateCache[key] = state
	return nil
}

func (c configStateCacheStub) Delete(key string) error {
	delete(c.stateCache, key)
	return nil
}

type nsExecutorStub struct {
	shouldNotBeExecuted bool
}

func (n nsExecutorStub) Do(f func() error) error {
	Expect(n.shouldNotBeExecuted).To(BeFalse(), "The namespace executor shouldn't be invoked")
	return f()
}

type ConfigStateStub struct {
	UnplugShouldFail  bool
	UnplugWasExecuted bool
	RunWasExecuted    bool
}

func (c *ConfigStateStub) Unplug(_ []v1.Network, _ func([]v1.Network) ([]string, error), _ func(string) error) error {
	c.UnplugWasExecuted = true
	if c.UnplugShouldFail {
		return fmt.Errorf("Unplug failure")
	}
	return nil
}

func (c *ConfigStateStub) Run(_ []podNIC, _ func(func() error) error) error {
	c.RunWasExecuted = true
	return nil
}
