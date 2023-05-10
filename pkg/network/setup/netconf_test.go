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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network_test

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"k8s.io/utils/pointer"

	kfs "kubevirt.io/kubevirt/pkg/os/fs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
)

var _ = Describe("netconf", func() {
	var netConf *netsetup.NetConf
	var vmi *v1.VirtualMachineInstance

	const launcherPid = 0

	BeforeEach(func() {
		netConf = netsetup.NewNetConfWithCustomFactory(nsNoopFactory, &tempCacheCreator{})
		vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "123"}}
	})

	It("runs setup successfully", func() {
		Expect(netConf.Setup(vmi, vmi.Spec.Networks, launcherPid, netPreSetupDummyNoop)).To(Succeed())
	})

	It("fails the pre-setup run", func() {
		Expect(netConf.Setup(vmi, vmi.Spec.Networks, launcherPid, netPreSetupFail)).NotTo(Succeed())
	})

	It("fails the setup run", func() {
		netConf := netsetup.NewNetConfWithCustomFactory(nsFailureFactory, &tempCacheCreator{})
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
		}}
		vmi.Spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}
		Expect(netConf.Setup(vmi, vmi.Spec.Networks, launcherPid, netPreSetupDummyNoop)).NotTo(Succeed())
	})

	It("fails the teardown run", func() {
		netConf := netsetup.NewNetConfWithCustomFactory(nil, failingCacheCreator{})
		Expect(netConf.Teardown(vmi)).NotTo(Succeed())
	})

	Context("HotUnplugInterfaces", func() {
		var shouldFail *bool
		var wasExecuted *bool

		const netName = "multusNet"
		const nadName = "blue"

		BeforeEach(func() {
			dutils.MockDefaultOwnershipManager()

			vmi.Spec.Networks = []v1.Network{{
				Name: netName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: nadName}},
			}}
			iface := v1.Interface{
				Name:                   netName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

			cacheCreator := tempCacheCreator{}
			shouldFail = pointer.Bool(false)
			wasExecuted = pointer.Bool(false)
			nsStub := netnsStub{shouldFail, wasExecuted}
			factory := func(_ int) netsetup.NSExecutor { return nsStub }
			netConf = netsetup.NewNetConfWithCustomFactory(factory, &cacheCreator)

			podIfaceCacheData := &cache.PodIfaceCacheData{State: cache.PodIfaceNetworkPreparationFinished}
			Expect(cache.WritePodInterfaceCache(&cacheCreator, string(vmi.UID), netName, podIfaceCacheData)).To(Succeed())
			_ = netConf.Setup(vmi, vmi.Spec.Networks, launcherPid, netPreSetupDummyNoop)
			vmi.Spec.Domain.Devices.Interfaces[0].State = v1.InterfaceStateAbsent
		})

		It("runs successfully", func() {
			Expect(netConf.HotUnplugInterfaces(vmi)).To(Succeed())
			Expect(*wasExecuted).To(BeTrue())
		})
		It("fails the unplug", func() {
			*shouldFail = true

			Expect(netConf.HotUnplugInterfaces(vmi)).NotTo(Succeed())
			Expect(*wasExecuted).To(BeTrue())
		})
	})
})

type netnsStub struct {
	shouldFail *bool
	executed   *bool
}

func (n netnsStub) Do(func() error) error {
	if n.executed != nil {
		if *n.executed == true {
			panic("executed should be false before execution")
		}
		*n.executed = true
	}
	if n.shouldFail != nil && *n.shouldFail == true {
		return fmt.Errorf("do-netns failure")
	}
	return nil
}
func nsNoopFactory(_ int) netsetup.NSExecutor    { return netnsStub{} }
func nsFailureFactory(_ int) netsetup.NSExecutor { return netnsStub{shouldFail: pointer.Bool(true)} }

func netPreSetupDummyNoop() error { return nil }

func netPreSetupFail() error { return fmt.Errorf("pre-setup failure") }

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
