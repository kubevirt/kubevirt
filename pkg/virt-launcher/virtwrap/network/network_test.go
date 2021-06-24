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
	"fmt"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	cache "kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	podnic "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/podnic"
)

var _ = Describe("VMNetworkConfigurator", func() {
	var (
		mockNetworkHandler *netdriver.MockNetworkHandler
		mockpodNICFactory  *MockpodNICFactory
		ctrl               *gomock.Controller
		cacheFactory       cache.InterfaceCacheFactory
	)

	newVMNetworkConfiguratorWithMocks := func(vmi *v1.VirtualMachineInstance) *VMNetworkConfigurator {
		vmNetworkConfigurator := newVMNetworkConfiguratorWithHandlerAndCache(vmi, mockNetworkHandler, cacheFactory)
		vmNetworkConfigurator.podNICFactory = mockpodNICFactory
		return vmNetworkConfigurator
	}

	newVMI := func(namespace, name string) *v1.VirtualMachineInstance {
		vmi := v1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
		return vmi
	}

	newVMIBridgeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := newVMI(namespace, name)
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockpodNICFactory = NewMockpodNICFactory(ctrl)
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
		mockNetworkHandler = netdriver.NewMockNetworkHandler(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("when PodNIC constructor returns an error", func() {
		var (
			expectedError = fmt.Errorf("netdriver_test: forcing failure at NewPodNIC")
			vmi           *v1.VirtualMachineInstance
		)
		BeforeEach(func() {
			vmi = newVMIBridgeInterface("testnamespace", "testVmName")
		})
		Context("and phase1 is called", func() {
			var err error
			BeforeEach(func() {
				pid := 1
				mockpodNICFactory.EXPECT().NewPodNIC(vmi, &vmi.Spec.Networks[0], mockNetworkHandler, cacheFactory, gomock.Eq(&pid)).Times(1).Return(nil, expectedError)
				err = newVMNetworkConfiguratorWithMocks(vmi).SetupPodNetworkPhase1(pid)
			})
			It("should propagate the error", func() {
				Expect(err).To(MatchError(expectedError))
			})
		})
		Context("and phase2 is called", func() {
			var err error
			BeforeEach(func() {
				mockpodNICFactory.EXPECT().NewPodNIC(vmi, &vmi.Spec.Networks[0], mockNetworkHandler, cacheFactory, gomock.Nil()).Times(1).Return(nil, expectedError)
				var domain *api.Domain
				err = newVMNetworkConfiguratorWithMocks(vmi).SetupPodNetworkPhase2(domain)
			})
			It("should propagate the error", func() {
				Expect(err).To(MatchError(expectedError))
			})
		})
	})
	Context("when calling []podnic.PodNIC factory functions", func() {
		It("should accept empty network list", func() {
			vmi := newVMI("testnamespace", "testVmName")
			nics, err := newVMNetworkConfiguratorWithMocks(vmi).getNICs()
			Expect(err).ToNot(HaveOccurred())
			Expect(nics).To(BeEmpty())
		})

		It("should configure networking with multus and a default multus network", func() {
			vmi := newVMI("testnamespace", "testVmName")
			vmi.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "default", Default: true},
					},
				},
				{
					Name: "additional1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "additional1"},
					},
				},
				{
					Name: "additional2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "additional2"},
					},
				},
			}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{
					Name: "additional1",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				},
				{
					Name: "default",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				},
				{
					Name: "additional2",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				},
			}
			for i, _ := range vmi.Spec.Networks {
				mockpodNICFactory.EXPECT().NewPodNIC(vmi, &vmi.Spec.Networks[i], mockNetworkHandler, cacheFactory, gomock.Nil()).Times(1).Return(&podnic.PodNIC{}, nil).Times(1)
			}

			obtainedPodNICs, err := newVMNetworkConfiguratorWithMocks(vmi).getNICs()
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedPodNICs).To(HaveLen(len(vmi.Spec.Networks)))
		})
	})
})
