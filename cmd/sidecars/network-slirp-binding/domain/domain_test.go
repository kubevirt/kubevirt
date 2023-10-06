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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vmschema "kubevirt.io/api/core/v1"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/cmd/sidecars/network-slirp-binding/domain"
)

var _ = Describe("QEMU slirp networking", func() {
	Context("configure domain spec slirp interface QMEU command line", func() {
		var testSearchDomain = []string{"dns.com"}

		DescribeTable("should fail, given",
			func(iface vmschema.Interface, network vmschema.Network) {
				_, err := domain.NewSlirpNetworkConfigurator(nil, []vmschema.Network{network}, nil)
				Expect(err).To(HaveOccurred())
			},
			Entry("no pod network",
				vmschema.Interface{},
				vmschema.Network{Name: "secondary", NetworkSource: vmschema.NetworkSource{Multus: &vmschema.MultusNetwork{}}},
			),
			Entry("no iface",
				vmschema.Interface{Name: "other", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}}},
				vmschema.Network{Name: "secondary", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
			),
			Entry("iface with no binding method or network binding plugin",
				vmschema.Interface{Name: "secondary"},
				vmschema.Network{Name: "secondary", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
			),
			Entry("iface with no slirp binding method",
				vmschema.Interface{Name: "secondary",
					InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}},
				},
				vmschema.Network{Name: "secondary", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
			),
			Entry("iface with no slirp network binding plugin",
				vmschema.Interface{Name: "secondary",
					Binding: &vmschema.PluginBinding{Name: "sriov"},
				},
				vmschema.Network{Name: "secondary", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
			),
		)

		DescribeTable("should succeed, given",
			func(iface vmschema.Interface, network vmschema.Network, expectedQEMUCmdArgs []domainschema.Arg) {
				testDomainSpec := &domainschema.NewMinimalDomain("test").Spec

				expectedDomainSpec := &domainschema.NewMinimalDomain("test").Spec
				expectedDomainSpec.QEMUCmd = &domainschema.Commandline{QEMUArg: expectedQEMUCmdArgs}

				testMutator, err := domain.NewSlirpNetworkConfigurator([]vmschema.Interface{iface}, []vmschema.Network{network}, testSearchDomain)
				Expect(err).ToNot(HaveOccurred())
				Expect(testMutator.Mutate(testDomainSpec)).To(Equal(expectedDomainSpec))
			},
			Entry("interface with slirp binding method",
				*vmschema.DefaultSlirpNetworkInterface(),
				*vmschema.DefaultPodNetwork(),
				[]domainschema.Arg{
					{Value: `-netdev`},
					{Value: `user,id=default,net=10.0.2.0/24,dnssearch=dns.com`},
					{Value: `-device`},
					{Value: `{"driver":"e1000","netdev":"default","id":"default"}`},
				},
			),
			Entry("custom CIDR",
				vmschema.Interface{Name: "slirpTest", Binding: &vmschema.PluginBinding{Name: domain.SlirpPluginName}},
				vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{
					VMNetworkCIDR: "192.168.100.0/24",
				}}},
				[]domainschema.Arg{
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=192.168.100.0/24,dnssearch=dns.com`},
					{Value: `-device`}, {Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest"}`},
				},
			),
			Entry("ports",
				vmschema.Interface{Name: "slirpTest", Binding: &vmschema.PluginBinding{Name: domain.SlirpPluginName},
					Ports: []vmschema.Port{
						{Name: "http", Protocol: "TCP", Port: 80},
						{Port: 8080},
					},
				},
				vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
				[]domainschema.Arg{
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com,hostfwd=tcp::80-:80,hostfwd=tcp::8080-:8080`},
					{Value: `-device`}, {Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest"}`},
				},
			),
			Entry("slirp interface with virtio model type - should be changed to e1000",
				vmschema.Interface{Name: "slirpTest", Binding: &vmschema.PluginBinding{Name: domain.SlirpPluginName},
					Model: "virtio",
				},
				vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
				[]domainschema.Arg{
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com`},
					{Value: `-device`}, {Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest"}`},
				},
			),
			Entry("custom MAC address",
				vmschema.Interface{Name: "slirpTest", Binding: &vmschema.PluginBinding{Name: domain.SlirpPluginName},
					MacAddress: "02:02:02:02:02:02",
				},
				vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
				[]domainschema.Arg{
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com`},
					{Value: `-device`}, {Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest","mac":"02:02:02:02:02:02"}`},
				},
			),
			Entry("iface model 'rtl8139'",
				vmschema.Interface{Name: "slirpTest", Binding: &vmschema.PluginBinding{Name: domain.SlirpPluginName},
					Model: "rtl8139",
				},
				vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
				[]domainschema.Arg{
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com`},
					{Value: `-device`}, {Value: `{"driver":"rtl8139","netdev":"slirpTest","id":"slirpTest"}`},
				},
			),
		)

		DescribeTable("should not override existing QEMU cmd args, given",
			func(iface vmschema.Interface, network vmschema.Network, existingArgs, expectedArgs []domainschema.Arg) {
				testMutator, err := domain.NewSlirpNetworkConfigurator([]vmschema.Interface{iface}, []vmschema.Network{network}, testSearchDomain)
				Expect(err).ToNot(HaveOccurred())

				domSpec := &domainschema.DomainSpec{
					QEMUCmd: &domainschema.Commandline{QEMUArg: existingArgs},
				}

				mutatedDomSpec, err := testMutator.Mutate(domSpec)
				Expect(err).ToNot(HaveOccurred())

				Expect(mutatedDomSpec.QEMUCmd.QEMUArg).To(Equal(expectedArgs))
			},
			Entry("other qmeu cmd args exist",
				vmschema.Interface{Name: "slirpTest", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Slirp: &vmschema.InterfaceSlirp{}}},
				vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
				[]domainschema.Arg{
					{Value: "-device"}, {Value: "foo"},
					{Value: "-M"},
				},
				[]domainschema.Arg{
					{Value: "-device"}, {Value: "foo"},
					{Value: "-M"},
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com`},
					{Value: `-device`}, {Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest"}`},
				},
			),
			Entry("slirp iface qemu cmd args already exist",
				vmschema.Interface{Name: "slirpTest", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Slirp: &vmschema.InterfaceSlirp{}}},
				vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}},
				[]domainschema.Arg{
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com`},
					{Value: `-device`}, {Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest"}`},
				},
				[]domainschema.Arg{
					{Value: `-netdev`}, {Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com`},
					{Value: `-device`}, {Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest"}`},
				},
			),
		)

		It("should set QEMU cmd args correctly when executed more than once", func() {
			iface := vmschema.Interface{Name: "slirpTest", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Slirp: &vmschema.InterfaceSlirp{}}}
			network := vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}}
			existingArgs := []domainschema.Arg{
				{Value: "-device"}, {Value: "foo"},
				{Value: "-M"},
			}

			testMutator, err := domain.NewSlirpNetworkConfigurator([]vmschema.Interface{iface}, []vmschema.Network{network}, testSearchDomain)
			Expect(err).ToNot(HaveOccurred())

			domSpec := &domainschema.DomainSpec{
				QEMUCmd: &domainschema.Commandline{QEMUArg: existingArgs},
			}

			expectedArgs := append(
				existingArgs,
				domainschema.Arg{Value: `-netdev`}, domainschema.Arg{Value: `user,id=slirpTest,net=10.0.2.0/24,dnssearch=dns.com`},
				domainschema.Arg{Value: `-device`}, domainschema.Arg{Value: `{"driver":"e1000","netdev":"slirpTest","id":"slirpTest"}`},
			)

			mutatedDomSpec, err := testMutator.Mutate(domSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.QEMUCmd.QEMUArg).To(Equal(expectedArgs))

			mutatedDomSpec, err = testMutator.Mutate(domSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.QEMUCmd.QEMUArg).To(Equal(expectedArgs))
		})

		It("should fail given invalid custom CIDR", func() {
			network := vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{
				VMNetworkCIDR: "592.468.300.0/24",
			}}}
			iface := vmschema.Interface{Name: "slirpTest", Binding: &vmschema.PluginBinding{Name: domain.SlirpPluginName}}

			testMutator, err := domain.NewSlirpNetworkConfigurator([]vmschema.Interface{iface}, []vmschema.Network{network}, testSearchDomain)
			Expect(err).ToNot(HaveOccurred())
			_, err = testMutator.Mutate(&domainschema.DomainSpec{})
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("should fail given invalid port",
			func(port int32) {
				iface := vmschema.Interface{Name: "slirpTest", Binding: &vmschema.PluginBinding{Name: domain.SlirpPluginName},
					Ports: []vmschema.Port{{Port: port}},
				}
				network := vmschema.Network{Name: "slirpTest", NetworkSource: vmschema.NetworkSource{Pod: &vmschema.PodNetwork{}}}

				testMutator, err := domain.NewSlirpNetworkConfigurator([]vmschema.Interface{iface}, []vmschema.Network{network}, testSearchDomain)
				Expect(err).ToNot(HaveOccurred())
				_, err = testMutator.Mutate(&domainschema.DomainSpec{})
				Expect(err).To(HaveOccurred())
			},
			Entry("invalid port: -1", int32(-1)),
			Entry("out off range port: 0", int32(0)),
		)
	})
})
