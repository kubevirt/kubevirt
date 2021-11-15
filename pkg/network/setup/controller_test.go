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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
)

var _ = Describe("controller", func() {
	var netCtrl netsetup.Controller
	var vmi *v1.VirtualMachineInstance

	const launcherPid = 0

	BeforeEach(func() {
		netCtrl = netsetup.NewController(&interfaceCacheFactoryStub{})

		vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "123"}}
	})

	It("runs setup successfully", func() {
		Expect(netCtrl.Setup(vmi, launcherPid, doNetNSDummyNoop, netPreSetupDummyNoop)).To(Succeed())
		Expect(netCtrl.SetupCompleted(vmi)).To(BeTrue())
	})

	It("runs teardown successfully", func() {
		Expect(netCtrl.Setup(vmi, launcherPid, doNetNSDummyNoop, netPreSetupDummyNoop)).To(Succeed())
		Expect(netCtrl.SetupCompleted(vmi)).To(BeTrue())
		Expect(netCtrl.Teardown(vmi)).To(Succeed())
		Expect(netCtrl.SetupCompleted(vmi)).To(BeFalse())
	})

	It("skips secondary setup runs", func() {
		Expect(netCtrl.Setup(vmi, launcherPid, doNetNSDummyNoop, netPreSetupDummyNoop)).To(Succeed())
		Expect(netCtrl.Setup(vmi, launcherPid, doNetNSDummyNoop, netPreSetupFail)).To(Succeed())
		Expect(netCtrl.SetupCompleted(vmi)).To(BeTrue())
	})

	It("fails the pre-setup run", func() {
		Expect(netCtrl.Setup(vmi, launcherPid, doNetNSDummyNoop, netPreSetupFail)).NotTo(Succeed())
		Expect(netCtrl.SetupCompleted(vmi)).To(BeFalse())
	})

	It("fails the setup run", func() {
		Expect(netCtrl.Setup(vmi, launcherPid, doNetNSFail, netPreSetupDummyNoop)).NotTo(Succeed())
		Expect(netCtrl.SetupCompleted(vmi)).To(BeFalse())
	})

	It("fails the teardown run", func() {
		factory := &interfaceCacheFactoryStub{podInterfaceCacheStoreStub{failRemove: true}}
		netCtrl = netsetup.NewController(factory)
		Expect(netCtrl.Teardown(vmi)).NotTo(Succeed())
		Expect(netCtrl.SetupCompleted(vmi)).To(BeFalse())
	})
})

type interfaceCacheFactoryStub struct{ podInterfaceCacheStore podInterfaceCacheStoreStub }

func (i interfaceCacheFactoryStub) CacheForVMI(vmi *v1.VirtualMachineInstance) cache.PodInterfaceCacheStore {
	return i.podInterfaceCacheStore
}
func (i interfaceCacheFactoryStub) CacheDomainInterfaceForPID(pid string) cache.DomainInterfaceStore {
	return nil
}
func (i interfaceCacheFactoryStub) CacheDHCPConfigForPid(pid string) cache.DHCPConfigStore {
	return nil
}

type podInterfaceCacheStoreStub struct{ failRemove bool }

func (p podInterfaceCacheStoreStub) Read(iface string) (*cache.PodCacheInterface, error) {
	return nil, nil
}

func (p podInterfaceCacheStoreStub) Write(iface string, cacheInterface *cache.PodCacheInterface) error {
	return nil
}

func (p podInterfaceCacheStoreStub) Remove() error {
	if p.failRemove {
		return fmt.Errorf("remove failed")
	}
	return nil
}

func doNetNSDummyNoop(func() error) error { return nil }
func netPreSetupDummyNoop() error         { return nil }

func doNetNSFail(func() error) error { return fmt.Errorf("do-netns failure") }
func netPreSetupFail() error         { return fmt.Errorf("pre-setup failure") }
