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

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Slirp Network", func() {
	var mockNetwork *MockNetworkHandler
	var ctrl *gomock.Controller
	var vm *v1.VirtualMachine
	var domain *api.Domain
	var dnsname string
	var iface *v1.Interface
	var network *v1.Network

	_, dnsnamelist, _ := getResolvConfDetailsFromPod()
	for _, dnsSearchName := range dnsnamelist {
		dnsname += fmt.Sprintf(",dnssearch=%s", dnsSearchName)
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = NewMockNetworkHandler(ctrl)
		Handler = mockNetwork
		iface = &v1.Interface{Name: "testnet", InterfaceBindingMethod: v1.InterfaceBindingMethod{}}
		network = &v1.Network{Name: "testnet", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}

		vm = newVM("testnamespace", "testvm")

		domain = DomainWithSlirpNetwork()
		api.SetObjectDefaults_Domain(domain)
	})

	Context("on successful setup", func() {
		It("should fail if podPort not exist", func() {
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{Ports: []v1.Port{v1.Port{VMPort: 80, Protocol: "TCP"}}}
			vm.Spec.Domain.Devices.Interfaces[0] = *iface

			SlirpBinding, err := getSlirpBinding(iface, network, domain)
			Expect(err).ToNot(HaveOccurred())

			err = SlirpBinding.configPortForward()
			Expect(err).To(HaveOccurred())
		})
		It("should add tcp if protocol not exist", func() {
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{Ports: []v1.Port{v1.Port{PodPort: 80}}}
			vm.Spec.Domain.Devices.Interfaces[0] = *iface

			SlirpBinding, err := getSlirpBinding(iface, network, domain)
			Expect(err).ToNot(HaveOccurred())
			err = SlirpBinding.configPortForward()
			Expect(err).ToNot(HaveOccurred())
			err = SlirpBinding.CommitConfiguration()
			Expect(err).NotTo(HaveOccurred())
			Expect(domain.Spec.QEMUCmd.QEMUArg[3].Value).To(Equal("user,id=testnet,hostfwd=tcp::80-:80"))

		})
		It("Should fail if not validated protocol gived", func() {
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{Ports: []v1.Port{v1.Port{Protocol: "test", PodPort: 80}}}
			vm.Spec.Domain.Devices.Interfaces[0] = *iface

			SlirpBinding, err := getSlirpBinding(iface, network, domain)
			Expect(err).ToNot(HaveOccurred())
			err = SlirpBinding.configPortForward()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Unknow protocol only TCP or UDP allowed"))
		})
		It("Should create the qemu configuration for interface", func() {
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{Ports: []v1.Port{v1.Port{PodPort: 80, VMPort: 80, Protocol: "TCP"}}}
			vm.Spec.Domain.Devices.Interfaces[0] = *iface

			SlirpBinding, err := getSlirpBinding(iface, network, domain)
			Expect(err).NotTo(HaveOccurred())
			err = SlirpBinding.configVMCIDR()
			Expect(err).NotTo(HaveOccurred())
			err = SlirpBinding.configDNSSearchName()
			Expect(err).NotTo(HaveOccurred())
			err = SlirpBinding.configPortForward()
			Expect(err).NotTo(HaveOccurred())
			err = SlirpBinding.CommitConfiguration()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(4))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1].Value).To(Equal("virtio,netdev=testnet"))
			// fmt.Println(domain.Spec.QEMUCmd.QEMUArg[3].Value)
			// fmt.Println("user,id=testnet,net=10.0.2.0/24" + dnsname + ",hostfwd=tcp::80-:80")
			Expect(domain.Spec.QEMUCmd.QEMUArg[3].Value).To(Equal("user,id=testnet,net=10.0.2.0/24" + dnsname + ",hostfwd=tcp::80-:80"))

		})
	})
})

func DomainWithSlirpNetwork() *api.Domain {
	domain := &api.Domain{}

	if domain.Spec.QEMUCmd == nil {
		domain.Spec.QEMUCmd = &api.Commandline{}
	}

	if domain.Spec.QEMUCmd.QEMUArg == nil {
		domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
	}

	domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: fmt.Sprintf("%s,netdev=%s", "virtio", "testnet")})

	return domain
}
