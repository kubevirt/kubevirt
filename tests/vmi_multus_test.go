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
	"flag"
	"fmt"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	postUrl    = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
	ovsConfCRD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"type\": \"ovs\", \"bridge\": \"br1\", \"vlan\": 100 }"}}`
	ptpConfCRD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"name\": \"mynet\", \"type\": \"ptp\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
)

var _ = Describe("Multus Networking", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	nodes, err := virtClient.CoreV1().Nodes().List(v13.ListOptions{})
	tests.PanicOnError(err)

	nodeAffinity := &k8sv1.Affinity{
		NodeAffinity: &k8sv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sv1.NodeSelectorRequirement{
							{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{nodes.Items[0].Name}},
						},
					},
				},
			},
		},
	}

	var detachedVMI *v1.VirtualMachineInstance
	var vmiOne *v1.VirtualMachineInstance
	var vmiTwo *v1.VirtualMachineInstance

	createVMI := func(interfaces []v1.Interface, networks []v1.Network) *v1.VirtualMachineInstance {
		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskAlpine), "#!/bin/bash\n")
		vmi.Spec.Domain.Devices.Interfaces = interfaces
		vmi.Spec.Networks = networks
		vmi.Spec.Affinity = nodeAffinity

		_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		return vmi
	}

	tests.BeforeAll(func() {
		tests.SkipIfNoMultusProvider(virtClient)
		tests.BeforeTestCleanup()
		result := virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "ovs-net-vlan100")).
			Body([]byte(fmt.Sprintf(ovsConfCRD, "ovs-net-vlan100", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())

		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "ptp-conf")).
			Body([]byte(fmt.Sprintf(ptpConfCRD, "ptp-conf", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
	})

	Context("VirtualMachineInstance with cni ptp plugin interface", func() {
		AfterEach(func() {
			virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(detachedVMI.Name, &v13.DeleteOptions{})
			fmt.Printf("Waiting for vmi %s in %s namespace to be removed, this can take a while ...\n", detachedVMI.Name, tests.NamespaceTestDefault)
			EventuallyWithOffset(1, func() bool {
				return errors.IsNotFound(virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(detachedVMI.Name, nil))
			}, 180*time.Second, 1*time.Second).
				Should(BeTrue())
		})

		It("should create a virtual machine with one interface", func() {
			By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
			detachedVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n")
			detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			detachedVMI.Spec.Networks = []v1.Network{
				{Name: "ptp", NetworkSource: v1.NetworkSource{
					Multus: &v1.CniNetwork{NetworkName: "ptp-conf"},
				}},
			}

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

			pingVirtualMachine(detachedVMI, "10.1.1.1", "\\$ ")
		})

		It("should create a virtual machine with two interfaces", func() {
			By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
			detachedVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n")

			detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			detachedVMI.Spec.Networks = []v1.Network{
				{Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					}},
				{Name: "ptp", NetworkSource: v1.NetworkSource{
					Multus: &v1.CniNetwork{NetworkName: "ptp-conf"},
				}},
			}

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

			cmdCheck := "sudo /sbin/cirros-dhcpc up eth1 > /dev/null\n"
			err = tests.CheckForTextExpecter(detachedVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: cmdCheck},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "ip addr show eth1 | grep 10.1.1 | wc -l"},
				&expect.BExp{R: "1"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())

			By("checking virtual machine instance as two interfaces")
			checkInterface(detachedVMI, "eth0", "\\$ ")
			checkInterface(detachedVMI, "eth1", "\\$ ")

			pingVirtualMachine(detachedVMI, "10.1.1.1", "\\$ ")
		})
	})

	Context("VirtualMachineInstance with ovs-cni plugin interface", func() {
		AfterEach(func() {
			deleteVMIs(virtClient, []*v1.VirtualMachineInstance{vmiOne, vmiTwo})
		})

		It("should create two virtual machines with one interface", func() {
			By("checking virtual machine instance can ping the secondary virtual machine instance using ovs-cni plugin")
			interfaces := []v1.Interface{{Name: "ovs", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			networks := []v1.Network{{Name: "ovs", NetworkSource: v1.NetworkSource{Multus: &v1.CniNetwork{NetworkName: "ovs-net-vlan100"}}}}

			vmiOne = createVMI(interfaces, networks)
			vmiTwo = createVMI(interfaces, networks)

			tests.WaitUntilVMIReady(vmiOne, tests.LoggedInAlpineExpecter)
			tests.WaitUntilVMIReady(vmiTwo, tests.LoggedInAlpineExpecter)

			configInterface(vmiOne, "eth0", "10.1.1.1/24", "localhost:~#")
			By("checking virtual machine interface eth0 state")
			checkInterface(vmiOne, "eth0", "localhost:~#")

			configInterface(vmiTwo, "eth0", "10.1.1.2/24", "localhost:~#")
			By("checking virtual machine interface eth0 state")
			checkInterface(vmiTwo, "eth0", "localhost:~#")

			By("ping between virtual machines")
			pingVirtualMachine(vmiOne, "10.1.1.2", "localhost:~#")
		})

		It("should create two virtual machines with two interfaces", func() {
			By("checking the first virtual machine instance can ping 10.1.1.2 using ovs-cni plugin")
			interfaces := []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				{Name: "ovs", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			networks := []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
				{Name: "ovs", NetworkSource: v1.NetworkSource{Multus: &v1.CniNetwork{NetworkName: "ovs-net-vlan100"}}}}

			vmiOne = createVMI(interfaces, networks)
			vmiTwo = createVMI(interfaces, networks)

			tests.WaitUntilVMIReady(vmiOne, tests.LoggedInAlpineExpecter)
			tests.WaitUntilVMIReady(vmiTwo, tests.LoggedInAlpineExpecter)

			configInterface(vmiOne, "eth1", "10.1.1.1/24", "localhost:~#")
			By("checking virtual machine interface eth1 state")
			checkInterface(vmiOne, "eth1", "localhost:~#")

			configInterface(vmiTwo, "eth1", "10.1.1.2/24", "localhost:~#")
			By("checking virtual machine interface eth1 state")
			checkInterface(vmiTwo, "eth1", "localhost:~#")

			By("ping between virtual machines")
			pingVirtualMachine(vmiOne, "10.1.1.2", "localhost:~#")
		})
	})

	Context("Single VirtualMachineInstance with ovs-cni plugin interface", func() {
		AfterEach(func() {
			deleteVMIs(virtClient, []*v1.VirtualMachineInstance{vmiOne})
		})

		It("should report all interfaces in Status", func() {
			interfaces := []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				{Name: "ovs", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			networks := []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
				{Name: "ovs", NetworkSource: v1.NetworkSource{Multus: &v1.CniNetwork{NetworkName: "ovs-net-vlan100"}}}}

			vmiOne = createVMI(interfaces, networks)

			tests.WaitUntilVMIReady(vmiOne, tests.LoggedInAlpineExpecter)

			updatedVmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmiOne.ObjectMeta.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(len(updatedVmi.Status.Interfaces)).To(Equal(2))
			interfacesByName := make(map[string]v1.VirtualMachineInstanceNetworkInterface)
			for _, ifc := range updatedVmi.Status.Interfaces {
				interfacesByName[ifc.Name] = ifc
			}

			for _, network := range networks {
				ifc, is_present := interfacesByName[network.Name]
				Expect(is_present).To(BeTrue())
				Expect(ifc.MAC).To(Not(BeZero()))
			}
			Expect(interfacesByName["default"].MAC).To(Not(Equal(interfacesByName["ovs"].MAC)))

			err = tests.CheckForTextExpecter(updatedVmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("ip addr show eth0 | grep %s | wc -l", interfacesByName["default"].MAC)},
				&expect.BExp{R: "1"},
			}, 15)
			err = tests.CheckForTextExpecter(updatedVmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("ip addr show eth1 | grep %s | wc -l", interfacesByName["ovs"].MAC)},
				&expect.BExp{R: "1"},
			}, 15)
		})
	})
})

func deleteVMIs(virtClient kubecli.KubevirtClient, vmis []*v1.VirtualMachineInstance) {
	for _, deleteVMI := range vmis {
		virtClient.VirtualMachineInstance("default").Delete(deleteVMI.Name, &v13.DeleteOptions{})
		fmt.Printf("Waiting for vmi %s in %s namespace to be removed, this can take a while ...\n", deleteVMI.Name, tests.NamespaceTestDefault)
		EventuallyWithOffset(1, func() bool {
			return errors.IsNotFound(virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(deleteVMI.Name, nil))
		}, 180*time.Second, 1*time.Second).
			Should(BeTrue())
	}
}

func configInterface(vmi *v1.VirtualMachineInstance, interfaceName, interfaceAddress, prompt string) {
	cmdCheck := fmt.Sprintf("ip addr add %s dev %s\n", interfaceAddress, interfaceName)
	err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 15)
	Expect(err).ToNot(HaveOccurred())

	cmdCheck = fmt.Sprintf("ip link set %s up\n", interfaceName)
	err = tests.CheckForTextExpecter(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 15)
	Expect(err).ToNot(HaveOccurred())
}

func checkInterface(vmi *v1.VirtualMachineInstance, interfaceName, prompt string) {
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 15)
	Expect(err).ToNot(HaveOccurred())
}

func pingVirtualMachine(vmi *v1.VirtualMachineInstance, ipAddr, prompt string) {
	cmdCheck := fmt.Sprintf("ping %s -c 1 -w 5\n", ipAddr)
	err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 30)
	Expect(err).ToNot(HaveOccurred())
}
