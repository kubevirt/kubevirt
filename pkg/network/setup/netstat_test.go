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

var _ = Describe("netstat", func() {
	const (
		iface0 = "iface0"
		iface1 = "iface1"
	)

	var netStat *netsetup.NetStat
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		netStat = netsetup.NewNetStat(&interfaceCacheFactoryStatusStub{})

		vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "123"}}
	})

	It("run status with no domain", func() {
		Expect(netStat.UpdateStatus(vmi, nil)).To(Succeed())
	})

	It("runs teardown that clears volatile cache", func() {
		data := &cache.PodCacheInterface{}
		netStat.CachePodInterfaceVolatileData(vmi, iface0, data)
		netStat.CachePodInterfaceVolatileData(vmi, iface1, data)

		netStat.Teardown(vmi)

		Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, iface0)).To(BeFalse())
		Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, iface1)).To(BeFalse())
	})
})

type interfaceCacheFactoryStatusStub struct {
	podInterfaceCacheStore podInterfaceCacheStoreStatusStub
}

func (i interfaceCacheFactoryStatusStub) CacheForVMI(vmi *v1.VirtualMachineInstance) cache.PodInterfaceCacheStore {
	return i.podInterfaceCacheStore
}
func (i interfaceCacheFactoryStatusStub) CacheDomainInterfaceForPID(pid string) cache.DomainInterfaceStore {
	return nil
}
func (i interfaceCacheFactoryStatusStub) CacheDHCPConfigForPid(pid string) cache.DHCPConfigStore {
	return nil
}

type podInterfaceCacheStoreStatusStub struct{ failRemove bool }

func (p podInterfaceCacheStoreStatusStub) Read(iface string) (*cache.PodCacheInterface, error) {
	return nil, nil
}

func (p podInterfaceCacheStoreStatusStub) Write(iface string, cacheInterface *cache.PodCacheInterface) error {
	return nil
}

func (p podInterfaceCacheStoreStatusStub) Remove() error {
	if p.failRemove {
		return fmt.Errorf("remove failed")
	}
	return nil
}
