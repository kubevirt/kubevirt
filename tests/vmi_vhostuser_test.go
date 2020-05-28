/*
 * This file is part of the kubevirt project
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

package tests_test

import (
	"encoding/json"
	"fmt"
	"time"

	netv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	vhostuserConfig = `{
		"cniVersion": "0.3.1",
		"type": "userspace",
		"name": "userspace-ovs-net-1",
		"kubeconfig": "/etc/cni/net.d/multus.d/multus.kubeconfig",
		"logFile": "/var/log/userspace-ovs-net-1-cni.log",
		"logLevel": "debug",
		"host": {
		        "engine": "ovs-dpdk",
		        "iftype": "vhostuser",
		        "netType": "bridge",
		        "vhost": {"mode": "client"},
		        "bridge": {"bridgeName": "br-dpdk0"}
		},
		"container": {
		        "engine": "ovs-dpdk",
		        "iftype": "vhostuser",
		        "netType": "interface",
		        "vhost": {"mode": "server"}
		},
		"ipam": {
		        "type": "host-local",
		        "subnet": "10.56.217.0/24",
		        "rangeStart": "10.56.217.131",
		        "rangeEnd": "10.56.217.190",
		        "routes": [{"dst": "0.0.0.0/0"}],
		        "gateway": "10.56.217.1"
		}
	}`
)

var _ = Describe("Vhostuser", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var nodes *k8sv1.NodeList

	BeforeEach(func() {
		// Multus tests need to ensure that old VMIs are gone
		Expect(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachineinstances").Do().Error()).To(Succeed())
		Expect(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestAlternative).Resource("virtualmachineinstances").Do().Error()).To(Succeed())
		Eventually(func() int {
			list1, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).List(&v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			list2, err := virtClient.VirtualMachineInstance(tests.NamespaceTestAlternative).List(&v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(list1.Items) + len(list2.Items)
		}, 6*time.Minute, 1*time.Second).Should(BeZero())
	})

	tests.BeforeAll(func() {
		tests.BeforeTestCleanup()

		nodes = tests.GetAllSchedulableNodes(virtClient)
		Expect(len(nodes.Items) > 0).To(BeTrue())

		var netAttachVhost1 netv1.NetworkAttachmentDefinition
		netAttachVhost1.APIVersion = "k8s.cni.cncf.io/v1"
		netAttachVhost1.Kind = "NetworkAttachmentDefinition"
		netAttachVhost1.Name = "vhostuser1"
		netAttachVhost1.Namespace = tests.NamespaceTestDefault
		netAttachVhost1.Spec.Config = vhostuserConfig
		netAttachVhostStr, err := json.Marshal(netAttachVhost1)
		Expect(err).ToNot(HaveOccurred())

		result := virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "vhostuser1")).
			Body(netAttachVhostStr).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())

	})

	Describe("[rfe_id:][crit:][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance using different types of interfaces.", func() {
		Context("VirtualMachineInstance with cni ptp plugin interface", func() {

			It("[test_id:]should create a virtual machine with one interface", func() {
				By("checking virtual machine instance can ping 10.1.1.1 ")
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}},
					{Name: "vhostnet1", InterfaceBindingMethod: v1.InterfaceBindingMethod{Vhostuser: &v1.InterfaceVhostuser{}}},
				}
				vmi.Spec.Networks = []v1.Network{
					{Name: "default", NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					}},
					{Name: "vhostnet1", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "vhostuser1"},
					}},
				}
				vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1Gi")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Hugepages: &v1.Hugepages{PageSize: "2Mi"},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

			})

		})

	})
})
