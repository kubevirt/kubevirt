/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package network_test

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	kfs "kubevirt.io/kubevirt/pkg/os/fs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod"
)

var _ = Describe("netconf", func() {
	const (
		testNetworkName = "default"
	)
	var (
		netConf  *netsetup.NetConf
		vmi      *v1.VirtualMachineInstance
		stateMap map[string]*netpod.State

		stateCache stateCacheStub
		ns         nsExecutorStub
	)

	const launcherPid = 0

	BeforeEach(func() {
		stateCache = newConfigStateCacheStub()
		ns = nsExecutorStub{}
		stateMap = map[string]*netpod.State{}
		netConf = netsetup.NewNetConfWithCustomFactoryAndConfigState(nsNoopFactory, &tempCacheCreator{}, stateMap, cConfigStub{})
		vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "123", Name: "vmi1"}}
	})

	It("runs setup successfully without networks", func() {
		Expect(netConf.Setup(vmi, vmi.Spec.Networks, launcherPid)).To(Succeed())
	})

	It("runs setup successfully with networks", func() {
		stateMap[string(vmi.UID)] = netpod.NewState(stateCache, ns)
		Expect(stateCache.Write(testNetworkName, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())

		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   testNetworkName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
		}}
		vmi.Spec.Networks = []v1.Network{{
			Name:          testNetworkName,
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}
		Expect(netConf.Setup(vmi, vmi.Spec.Networks, launcherPid)).To(Succeed())
		Expect(stateCache.Read(testNetworkName)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
	})

	DescribeTable("setup ignores specific network bindings", func(binding v1.InterfaceBindingMethod) {
		netConf = netsetup.NewNetConfWithCustomFactoryAndConfigState(nsFailureFactory, &tempCacheCreator{}, stateMap, cConfigStub{})

		stateMap[string(vmi.UID)] = netpod.NewState(stateCache, ns)

		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   testNetworkName,
			InterfaceBindingMethod: binding,
		}}
		emptyBindingMethod := v1.InterfaceBindingMethod{}
		if binding == emptyBindingMethod {
			vmi.Spec.Domain.Devices.Interfaces[0].Binding = &v1.PluginBinding{}
		}
		vmi.Spec.Networks = []v1.Network{{
			Name:          testNetworkName,
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}
		Expect(netConf.Setup(vmi, vmi.Spec.Networks, launcherPid)).To(Succeed())
		Expect(stateCache.stateCache).To(BeEmpty())
	},
		Entry("SR-IOV", v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}),

		// Macvtap is removed in v1.3. This scenario is tracking old VMIs that are still processed in the reconcile loop.
		Entry("macvtap", v1.InterfaceBindingMethod{DeprecatedMacvtap: &v1.DeprecatedInterfaceMacvtap{}}),
	)

	It("fails the setup run", func() {
		netConf := netsetup.NewNetConfWithCustomFactoryAndConfigState(nsFailureFactory, &tempCacheCreator{}, stateMap, cConfigStub{})
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   testNetworkName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
		}}
		vmi.Spec.Networks = []v1.Network{{
			Name:          testNetworkName,
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}
		Expect(netConf.Setup(vmi, vmi.Spec.Networks, launcherPid)).NotTo(Succeed())
	})

	It("fails the teardown run", func() {
		netConf := netsetup.NewNetConfWithCustomFactoryAndConfigState(nil, failingCacheCreator{}, stateMap, cConfigStub{})
		Expect(netConf.Teardown(vmi)).NotTo(Succeed())
	})
})

type netnsStub struct {
	shouldFail bool
}

func (n netnsStub) Do(func() error) error {
	if n.shouldFail {
		return fmt.Errorf("do-netns failure")
	}
	return nil
}
func nsNoopFactory(_ int) netsetup.NSExecutor    { return netnsStub{} }
func nsFailureFactory(_ int) netsetup.NSExecutor { return netnsStub{shouldFail: true} }

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

type failingCacheCreator struct{}

func (c failingCacheCreator) New(path string) *cache.Cache {
	return cache.NewCustomCache(path, stubFS{failRemove: true})
}

type stubFS struct{ failRemove bool }

func (f stubFS) Stat(name string) (os.FileInfo, error)                          { return nil, nil }
func (f stubFS) MkdirAll(path string, perm os.FileMode) error                   { return nil }
func (f stubFS) ReadFile(filename string) ([]byte, error)                       { return nil, nil }
func (f stubFS) WriteFile(filename string, data []byte, perm fs.FileMode) error { return nil }
func (f stubFS) RemoveAll(path string) error {
	if f.failRemove {
		return fmt.Errorf("remove failed")
	}
	return nil
}

type stateCacheStub struct {
	stateCache map[string]cache.PodIfaceState
}

func newConfigStateCacheStub() stateCacheStub {
	return stateCacheStub{map[string]cache.PodIfaceState{}}
}

func (c stateCacheStub) Read(key string) (cache.PodIfaceState, error) {
	return c.stateCache[key], nil
}

func (c stateCacheStub) Write(key string, state cache.PodIfaceState) error {
	c.stateCache[key] = state
	return nil
}

func (c stateCacheStub) Delete(key string) error {
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

type cConfigStub struct{}

func (c cConfigStub) GetNetworkBindings() map[string]v1.InterfaceBindingPlugin {
	return map[string]v1.InterfaceBindingPlugin{}
}
