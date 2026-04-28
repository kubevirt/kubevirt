/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("VMNetworkConfigurator", func() {
	var baseCacheCreator tempCacheCreator

	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})
	Context("interface configuration", func() {
		Context("when calling []podNIC factory functions", func() {
			It("should not process SR-IOV networks", func() {
				const networkName = "sriov"
				vmi := libvmi.New(
					libvmi.WithNamespace("testnamespace"),
					libvmi.WithName("testVmName"),
					libvmi.WithNetwork(&v1.Network{
						Name: networkName,
						NetworkSource: v1.NetworkSource{
							Multus: &v1.MultusNetwork{NetworkName: "sriov-nad"},
						},
					}),
					libvmi.WithInterface(v1.Interface{
						Name: networkName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							SRIOV: &v1.InterfaceSRIOV{},
						},
					}),
				)

				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, nil)

				nics, err := vmNetworkConfigurator.getPhase2NICs(&api.Domain{}, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(BeEmpty())
			})
		})
	})
})
