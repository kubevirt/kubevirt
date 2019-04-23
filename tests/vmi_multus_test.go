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
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	postUrl              = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
	ovsConfCRD           = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"type\": \"ovs\", \"bridge\": \"br1\", \"vlan\": 100 }"}}`
	ptpConfCRD           = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"name\": \"mynet\", \"type\": \"ptp\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	ptpConfWithTuningCRD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"ptp\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" }},{\"type\": \"tuning\"}]}"}}`
	sriovConfCRD         = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"intel.com/sriov"}},"spec":{"config":"{ \"name\": \"sriov\", \"type\": \"sriov\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
)

var _ = Describe("Multus", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var nodes *k8sv1.NodeList

	defaultInterface := v1.Interface{
		Name: "default",
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Bridge: &v1.InterfaceBridge{},
		},
	}

	ovsInterface := v1.Interface{
		Name: "ovs",
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Bridge: &v1.InterfaceBridge{},
		},
	}

	defaultNetwork := v1.Network{
		Name: "default",
		NetworkSource: v1.NetworkSource{
			Pod: &v1.PodNetwork{},
		},
	}

	ovsNetwork := v1.Network{
		Name: "ovs",
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: "ovs-net-vlan100",
			},
		},
	}

	AfterEach(func() {
		// Multus tests need to ensure that old VMIs are gone
		Expect(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachineinstances").Do().Error()).To(Succeed())
		Expect(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestAlternative).Resource("virtualmachineinstances").Do().Error()).To(Succeed())
		Eventually(func() int {
			list1, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).List(&v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			list2, err := virtClient.VirtualMachineInstance(tests.NamespaceTestAlternative).List(&v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(list1.Items) + len(list2.Items)
		}, 180*time.Second, 1*time.Second).Should(BeZero())
	})

	createVMIOnNode := func(interfaces []v1.Interface, networks []v1.Network) *v1.VirtualMachineInstance {
		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskAlpine), "#!/bin/bash\n")
		vmi.Spec.Domain.Devices.Interfaces = interfaces
		vmi.Spec.Networks = networks

		// Arbitrarily select one compute node in the cluster, on which it is possible to create a VMI
		// (i.e. a schedulable node).
		nodeName := nodes.Items[0].Name
		tests.StartVmOnNode(vmi, nodeName)

		return vmi
	}

	tests.BeforeAll(func() {
		tests.BeforeTestCleanup()

		nodes = tests.GetAllSchedulableNodes(virtClient)
		Expect(len(nodes.Items) > 0).To(BeTrue())

		result := virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "ovs-net-vlan100")).
			Body([]byte(fmt.Sprintf(ovsConfCRD, "ovs-net-vlan100", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())

		// Create identical ptp crds in two different namespaces
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "ptp-conf")).
			Body([]byte(fmt.Sprintf(ptpConfCRD, "ptp-conf", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestAlternative, "ptp-conf-2")).
			Body([]byte(fmt.Sprintf(ptpConfCRD, "ptp-conf-2", tests.NamespaceTestAlternative))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())

		// Create ptp crd with tuning plugin enabled
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "ptp-conf-tuning")).
			Body([]byte(fmt.Sprintf(ptpConfWithTuningCRD, "ptp-conf-tuning", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())

		// Create two sriov networks referring to the same resource name
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "sriov")).
			Body([]byte(fmt.Sprintf(sriovConfCRD, "sriov", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "sriov2")).
			Body([]byte(fmt.Sprintf(sriovConfCRD, "sriov2", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
	})

	Describe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance using different types of interfaces.", func() {
		Context("VirtualMachineInstance with cni ptp plugin interface", func() {

			It("[test_id:1751]should create a virtual machine with one interface", func() {
				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "ptp-conf"},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

				pingVirtualMachine(detachedVMI, "10.1.1.1", "\\$ ")
			})

			It("[test_id:1752]should create a virtual machine with one interface with network definition from different namespace", func() {
				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: fmt.Sprintf("%s/%s", tests.NamespaceTestAlternative, "ptp-conf-2")},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

				pingVirtualMachine(detachedVMI, "10.1.1.1", "\\$ ")
			})

			It("[test_id:1753]should create a virtual machine with two interfaces", func() {
				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					defaultNetwork,
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "ptp-conf"},
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

				By("checking virtual machine instance has two interfaces")
				checkInterface(detachedVMI, "eth0", "\\$ ")
				checkInterface(detachedVMI, "eth1", "\\$ ")

				pingVirtualMachine(detachedVMI, "10.1.1.1", "\\$ ")
			})
		})

		Context("VirtualMachineInstance with multus network as default network", func() {

			It("should create a virtual machine with one interface with multus default network definition", func() {
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: fmt.Sprintf("%s/%s", tests.NamespaceTestDefault, "ptp-conf"),
							Default:     true,
						}}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				pingVirtualMachine(detachedVMI, "10.1.1.1", "\\$ ")

				By("checking virtual machine instance only has one interface")
				// lo0, eth0
				err = tests.CheckForTextExpecter(detachedVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "ip link show | grep -c UP\n"},
					&expect.BExp{R: "2"},
				}, 15)
				Expect(err).ToNot(HaveOccurred())

				By("checking pod has only one interface")
				// lo0, eth0, k6t-eth0, vnet0
				output := tests.RunCommandOnVmiPod(detachedVMI, []string{"/bin/bash", "-c", "/usr/sbin/ip link show|grep -c UP"})
				ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("4"))
			})
		})

		Context("VirtualMachineInstance with cni ptp plugin interface with custom MAC address", func() {

			table.DescribeTable("configure valid custom MAC address on ptp interface", func(networkName string) {
				customMacAddress := "50:00:00:00:90:0d"
				ptpInterface := v1.Interface{
					Name: "ptp",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				}
				ptpNetwork := v1.Network{
					Name: "ptp",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: networkName,
						},
					},
				}

				interfaces := []v1.Interface{ptpInterface}
				networks := []v1.Network{ptpNetwork}

				By("Creating a VM with custom MAC address on its ptp interface.")
				interfaces[0].MacAddress = customMacAddress
				vmiOne := createVMIOnNode(interfaces, networks)
				tests.WaitUntilVMIReady(vmiOne, tests.LoggedInAlpineExpecter)

				By("Configuring static IP address to ptp interface.")
				configInterface(vmiOne, "eth0", "10.1.1.1/24", "localhost:~#")

				By("Verifying the desired custom MAC is the one that was actually configured on the interface.")
				ipLinkShow := fmt.Sprintf("ip link show eth0 | grep -i \"%s\" | wc -l\n", customMacAddress)
				err = tests.CheckForTextExpecter(vmiOne, []expect.Batcher{
					&expect.BSnd{S: ipLinkShow},
					&expect.BExp{R: "1"},
				}, 15)
				Expect(err).ToNot(HaveOccurred())

				By("Verifying the desired custom MAC is not configured inside the pod namespace.")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, tests.NamespaceTestDefault)
				out, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"sh", "-c", "ip a"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Contains(out, customMacAddress)).To(BeFalse())
			},
				table.Entry("when not using tuning plugin", "ptp-conf"),
				table.Entry("when using tuning plugin", "ptp-conf-tuning"),
			)
		})

		Context("VirtualMachineInstance with sriov plugin interface", func() {
			BeforeEach(func() {
				tests.SkipIfNoSriovDevicePlugin(virtClient)
			})

			It("[test_id:1754]should create a virtual machine with sriov interface", func() {
				// since neither cirros nor alpine has drivers for Intel NICs, we are left with fedora
				userData := "#cloud-config\npassword: fedora\nchpasswd: { expire: False }\n"
				vmiOne := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskFedora), userData)
				tests.AddExplicitPodNetworkInterface(vmiOne)

				iface := v1.Interface{Name: "sriov", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}
				network := v1.Network{Name: "sriov", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "sriov"}}}
				vmiOne.Spec.Domain.Devices.Interfaces = append(vmiOne.Spec.Domain.Devices.Interfaces, iface)
				vmiOne.Spec.Networks = append(vmiOne.Spec.Networks, network)

				// fedora requires some more memory to boot without kernel panics
				vmiOne.Spec.Domain.Resources.Requests[k8sv1.ResourceName("memory")] = resource.MustParse("1024M")

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmiOne)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(vmiOne, tests.LoggedInFedoraExpecter)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, tests.NamespaceTestDefault)
				out, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"sh", "-c", "echo $KUBEVIRT_RESOURCE_NAME_sriov"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(out).To(Equal("intel.com/sriov\n"))

				By("checking default interface is present")
				_, err = tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"ip", "address", "show", "eth0"},
				)
				Expect(err).ToNot(HaveOccurred())

				By("checking default interface is attached to VMI")
				_, err = tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"ip", "address", "show", "k6t-eth0"},
				)
				Expect(err).ToNot(HaveOccurred())

				By("checking virtual machine instance has two interfaces")
				checkInterface(vmiOne, "eth0", "#")
				checkInterface(vmiOne, "eth1", "#")

				// there is little we can do beyond just checking two devices are present: PCI slots are different inside
				// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
				// it's hard to match them.
			})

			It("[test_id:1755]should create a virtual machine with two sriov interfaces referring the same resource", func() {
				// since neither cirros nor alpine has drivers for Intel NICs, we are left with fedora
				userData := "#cloud-config\npassword: fedora\nchpasswd: { expire: False }\n"
				vmiOne := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskFedora), userData)
				tests.AddExplicitPodNetworkInterface(vmiOne)

				for _, name := range []string{"sriov", "sriov2"} {
					iface := v1.Interface{Name: name, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}
					network := v1.Network{Name: name, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: name}}}
					vmiOne.Spec.Domain.Devices.Interfaces = append(vmiOne.Spec.Domain.Devices.Interfaces, iface)
					vmiOne.Spec.Networks = append(vmiOne.Spec.Networks, network)
				}

				// fedora requires some more memory to boot without kernel panics
				vmiOne.Spec.Domain.Resources.Requests[k8sv1.ResourceName("memory")] = resource.MustParse("1024M")

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmiOne)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(vmiOne, tests.LoggedInFedoraExpecter)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variables are defined in pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, tests.NamespaceTestDefault)
				for _, name := range []string{"sriov", "sriov"} {
					out, err := tests.ExecuteCommandOnPod(
						virtClient,
						vmiPod,
						"compute",
						[]string{"sh", "-c", fmt.Sprintf("echo $KUBEVIRT_RESOURCE_NAME_%s", name)},
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(out).To(Equal("intel.com/sriov\n"))
				}

				By("checking default interface is present")
				_, err = tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"ip", "address", "show", "eth0"},
				)
				Expect(err).ToNot(HaveOccurred())

				By("checking default interface is attached to VMI")
				_, err = tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"ip", "address", "show", "k6t-eth0"},
				)
				Expect(err).ToNot(HaveOccurred())

				By("checking virtual machine instance has three interfaces")
				checkInterface(vmiOne, "eth0", "#")
				checkInterface(vmiOne, "eth1", "#")
				checkInterface(vmiOne, "eth2", "#")

				// there is little we can do beyond just checking two devices are present: PCI slots are different inside
				// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
				// it's hard to match them.
			})
		})

		Context("VirtualMachineInstance with ovs-cni plugin interface", func() {

			It("[test_id:1577]should create two virtual machines with one interface", func() {
				By("checking virtual machine instance can ping the secondary virtual machine instance using ovs-cni plugin")
				interfaces := []v1.Interface{ovsInterface}
				networks := []v1.Network{ovsNetwork}

				vmiOne := createVMIOnNode(interfaces, networks)
				vmiTwo := createVMIOnNode(interfaces, networks)

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

			It("[test_id:1578]should create two virtual machines with two interfaces", func() {
				By("checking the first virtual machine instance can ping 10.1.1.2 using ovs-cni plugin")
				interfaces := []v1.Interface{
					defaultInterface,
					ovsInterface,
				}
				networks := []v1.Network{
					defaultNetwork,
					ovsNetwork,
				}

				vmiOne := createVMIOnNode(interfaces, networks)
				vmiTwo := createVMIOnNode(interfaces, networks)

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

		Context("VirtualMachineInstance with ovs-cni plugin interface and custom MAC address.", func() {
			interfaces := []v1.Interface{ovsInterface}
			networks := []v1.Network{ovsNetwork}
			ovsIfIdx := 0
			customMacAddress := "50:00:00:00:90:0d"

			It("[test_id:676]should configure valid custom MAC address on ovs-cni interface.", func() {
				By("Creating a VM with ovs-cni network interface and default MAC address.")
				vmiTwo := createVMIOnNode(interfaces, networks)
				tests.WaitUntilVMIReady(vmiTwo, tests.LoggedInAlpineExpecter)

				By("Creating another VM with custom MAC address on its ovs-cni interface.")
				interfaces[ovsIfIdx].MacAddress = customMacAddress
				vmiOne := createVMIOnNode(interfaces, networks)
				tests.WaitUntilVMIReady(vmiOne, tests.LoggedInAlpineExpecter)

				By("Configuring static IP address to the ovs interface.")
				configInterface(vmiOne, "eth0", "10.1.1.1/24", "localhost:~#")
				configInterface(vmiTwo, "eth0", "10.1.1.2/24", "localhost:~#")

				By("Verifying the desired custom MAC is the one that were actually configured on the interface.")
				ipLinkShow := fmt.Sprintf("ip link show eth0 | grep -i \"%s\" | wc -l\n", customMacAddress)
				err = tests.CheckForTextExpecter(vmiOne, []expect.Batcher{
					&expect.BSnd{S: ipLinkShow},
					&expect.BExp{R: "1"},
				}, 15)
				Expect(err).ToNot(HaveOccurred())

				By("Verifying the desired custom MAC is not configured inside the pod namespace.")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, tests.NamespaceTestDefault)
				out, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"sh", "-c", "ip a"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Contains(out, customMacAddress)).To(BeFalse())

				By("Ping from the VM with the custom MAC to the other VM.")
				pingVirtualMachine(vmiOne, "10.1.1.2", "localhost:~#")
			})
		})
		Context("Single VirtualMachineInstance with ovs-cni plugin interface", func() {

			It("[test_id:1756]should report all interfaces in Status", func() {
				interfaces := []v1.Interface{
					defaultInterface,
					ovsInterface,
				}
				networks := []v1.Network{
					defaultNetwork,
					ovsNetwork,
				}

				vmiOne := createVMIOnNode(interfaces, networks)

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

		Context("VirtualMachineInstance with invalid MAC addres", func() {
			BeforeEach(func() {
				tests.BeforeTestCleanup()
			})

			It("[test_id:1713]should failed to start with invalid MAC address", func() {
				By("Start VMI")
				ovsIfIdx := 1

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskAlpine), "#!/bin/bash\n")
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					ovsInterface,
				}
				vmi.Spec.Domain.Devices.Interfaces[ovsIfIdx].MacAddress = "de:00c:00c:00:00:de:abc"

				vmi.Spec.Networks = []v1.Network{
					defaultNetwork,
					ovsNetwork,
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred())
				testErr := err.(*errors.StatusError)
				Expect(testErr.ErrStatus.Reason).To(BeEquivalentTo("Invalid"))
			})
		})
	})

	Describe("[rfe_id:1758][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance definition", func() {
		Context("with quemu guest agent", func() {

			It("[test_id:1757] should report guest interfaces in VMI status", func() {
				interfaces := []v1.Interface{
					defaultInterface,
					ovsInterface,
				}
				networks := []v1.Network{
					defaultNetwork,
					ovsNetwork,
				}

				ep1Ip := "1.0.0.10/24"
				ep2Ip := "1.0.0.11/24"
				ep1IpV6 := "fe80::ce3d:82ff:fe52:24c0/64"
				ep2IpV6 := "fe80::ce3d:82ff:fe52:24c1/64"
				userdata := fmt.Sprintf(`#!/bin/bash
                    echo "fedora" |passwd fedora --stdin
                    setenforce 0
                    ip link add ep1 type veth peer name ep2
                    ip addr add %s dev ep1
	                ip addr add %s dev ep2
	                ip addr add %s dev ep1
	                ip addr add %s dev ep2
                    mkdir -p /usr/local/bin
                    curl %s > /usr/local/bin/qemu-ga
                    chmod +x /usr/local/bin/qemu-ga
                    systemd-run --unit=guestagent /usr/local/bin/qemu-ga
                `, ep1Ip, ep2Ip, ep1IpV6, ep2IpV6, tests.GuestAgentHttpUrl)
				agentVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskFedora), userdata)

				agentVMI.Spec.Domain.Devices.Interfaces = interfaces
				agentVMI.Spec.Networks = networks
				agentVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

				By("Starting a VirtualMachineInstance")
				agentVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				tests.WaitForSuccessfulVMIStart(agentVMI)

				getOptions := &metav1.GetOptions{}
				var updatedVmi *v1.VirtualMachineInstance

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				Eventually(func() bool {
					updatedVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, getOptions)
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVmi.Status.Conditions {
						if condition.Type == "AgentConnected" && condition.Status == "True" {
							return true
						}
					}
					return false
				}, 420*time.Second, 2).Should(BeTrue(), "Should have agent connected condition")

				Eventually(func() bool {
					updatedVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, getOptions)
					return len(updatedVmi.Status.Interfaces) == 4
				}, 420*time.Second, 4).Should(BeTrue(), "Should have interfaces in vmi status")

				updatedVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, getOptions)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(updatedVmi.Status.Interfaces)).To(Equal(4))
				interfaceByIfcName := make(map[string]v1.VirtualMachineInstanceNetworkInterface)
				for _, ifc := range updatedVmi.Status.Interfaces {
					interfaceByIfcName[ifc.InterfaceName] = ifc
				}
				Expect(interfaceByIfcName["eth0"].Name).To(Equal("default"))
				Expect(interfaceByIfcName["eth0"].InterfaceName).To(Equal("eth0"))

				Expect(interfaceByIfcName["eth1"].Name).To(Equal("ovs"))
				Expect(interfaceByIfcName["eth1"].InterfaceName).To(Equal("eth1"))

				Expect(interfaceByIfcName["ep1"].Name).To(Equal(""))
				Expect(interfaceByIfcName["ep1"].InterfaceName).To(Equal("ep1"))
				Expect(interfaceByIfcName["ep1"].IP).To(Equal(ep1Ip))
				Expect(interfaceByIfcName["ep1"].IPs).To(Equal([]string{ep1Ip, ep1IpV6}))

				Expect(interfaceByIfcName["ep2"].Name).To(Equal(""))
				Expect(interfaceByIfcName["ep2"].InterfaceName).To(Equal("ep2"))
				Expect(interfaceByIfcName["ep2"].IP).To(Equal(ep2Ip))
				Expect(interfaceByIfcName["ep2"].IPs).To(Equal([]string{ep2Ip, ep2IpV6}))
			})
		})
	})
})

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
