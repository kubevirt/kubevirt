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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("VMNetworkConfigurator", func() {
	var baseCacheCreator tempCacheCreator

	const launcherPID = 0

	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})
	Context("interface configuration", func() {

		It("when vm has no network source should propagate errors when phase2 is called", func() {
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Networks = []v1.Network{{
				Name:          "default",
				NetworkSource: v1.NetworkSource{},
			}}
			vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator)
			var domain *api.Domain
			err := vmNetworkConfigurator.SetupPodNetworkPhase2(domain, vmi.Spec.Networks)
			Expect(err).To(MatchError("Network not implemented"))
		})

		Context("when calling []podNIC factory functions", func() {
			It("should not process SR-IOV networks", func() {
				vmi := api2.NewMinimalVMIWithNS("testnamespace", "testVmName")
				const networkName = "sriov"
				vmi.Spec.Networks = []v1.Network{{
					Name: networkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "sriov-nad"},
					},
				}}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
					Name: networkName, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
				}}

				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, nil)

				nics, err := vmNetworkConfigurator.getPhase2NICs(&api.Domain{}, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(BeEmpty())
			})
		})
	})
})
