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

package libvirt

import (
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("libvirt spec generator", func() {
	var (
		specGenerator SpecGenerator
		domain        *api.Domain
		iface         *v1.Interface
	)
	NewDomainWithSlirpInterface := func() *api.Domain {
		domain := &api.Domain{}
		domain.Spec.Devices.Interfaces = []api.Interface{{
			Model: &api.Model{
				Type: "e1000",
			},
			Type:  "user",
			Alias: api.NewUserDefinedAlias("default"),
		},
		}

		// Create network interface
		if domain.Spec.QEMUCmd == nil {
			domain.Spec.QEMUCmd = &api.Commandline{}
		}

		if domain.Spec.QEMUCmd.QEMUArg == nil {
			domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
		}

		return domain
	}
	NewDomainWithMacvtapInterface := func(macvtapName string) *api.Domain {
		domain := &api.Domain{}
		domain.Spec.Devices.Interfaces = []api.Interface{{
			Alias: api.NewUserDefinedAlias(macvtapName),
			Model: &api.Model{
				Type: "virtio",
			},
			Type: "ethernet",
		}}
		return domain
	}
	Context("when slirp generator is selected with a domain with one slirp interface", func() {
		BeforeEach(func() {
			domain = NewDomainWithSlirpInterface()
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			iface = v1.DefaultSlirpNetworkInterface()
			specGenerator = NewSlirpSpecGenerator(iface, domain)
		})
		It("Should create an interface in the qemu command line and remove it from the interfaces", func() {
			Expect(specGenerator.Generate(api.Interface{})).To(Succeed())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(0))
			Expect(domain.Spec.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
		})
		Context("and we set the MAC address on the interface", func() {
			BeforeEach(func() {
				iface.MacAddress = "de-ad-00-00-be-af"
			})
			It("Should append the MAC address to qemu arguments", func() {
				Expect(specGenerator.Generate(api.Interface{})).To(Succeed())
				Expect(domain.Spec.Devices.Interfaces).To(HaveLen(0))
				Expect(domain.Spec.QEMUCmd.QEMUArg).To(HaveLen(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default,mac=de-ad-00-00-be-af"}))
			})
		})
		Context("and we append a non slirp interface to the domain", func() {
			BeforeEach(func() {
				domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, api.Interface{
					Model: &api.Model{
						Type: "virtio",
					},
					Type: "bridge",
					Source: api.InterfaceSource{
						Bridge: api.DefaultBridgeName,
					},
					Alias: api.NewUserDefinedAlias("default"),
				})
			})
			It("Should create an interface in the qemu command line, remove it from the interfaces and leave the other interfaces inplace", func() {
				Expect(specGenerator.Generate(api.Interface{})).To(Succeed())
				Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
				Expect(domain.Spec.QEMUCmd.QEMUArg).To(HaveLen(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
			})
		})
	})
	Context("when macvtap generator is selected", func() {
		const ifaceName = "macvtap0"
		var (
			specGenerator SpecGenerator
			domain        *api.Domain
			iface         *v1.Interface
		)
		BeforeEach(func() {
			domain = NewDomainWithMacvtapInterface("default")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			iface = v1.DefaultMacvtapNetworkInterface("default")
			specGenerator = NewMacvtapSpecGenerator(iface, domain)
		})
		It("Should pass a non-privileged macvtap interface to qemu", func() {
			mac := &api.MAC{MAC: "12:34:56:78:9A:BC"}
			mtu := &api.MTU{Size: "1420"}
			target := &api.InterfaceTarget{Device: ifaceName, Managed: "no"}
			Expect(specGenerator.Generate(api.Interface{MAC: mac, MTU: mtu, Target: target})).To(Succeed())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(domain.Spec.Devices.Interfaces[0].Target).To(Equal(target), "should have an unmanaged interface")
			Expect(domain.Spec.Devices.Interfaces[0].MAC).To(Equal(mac), "should have the expected MAC address")
			Expect(domain.Spec.Devices.Interfaces[0].MTU).To(Equal(mtu), "should have the expected MTU")

		})
	})
})
