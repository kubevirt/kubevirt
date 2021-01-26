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
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/assert"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const (
	postUrl                = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
	linuxBridgeConfCRD     = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"bridge\", \"bridge\": \"br10\", \"vlan\": 100, \"ipam\": {}},{\"type\": \"tuning\"}]}"}}`
	ptpConfCRD             = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"ptp\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" }},{\"type\": \"tuning\"}]}"}}`
	sriovConfCRD           = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	sriovLinkEnableConfCRD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	macvtapNetworkConf     = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s", "annotations": {"k8s.v1.cni.cncf.io/resourceName": "macvtap.network.kubevirt.io/%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"%s\", \"type\": \"macvtap\"}"}}`
	sriovConfVlanCRD       = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"vlan\": 200, \"ipam\":{}}"}}`
)

const (
	sriovnet1 = "sriov"
	sriovnet2 = "sriov2"
	sriovnet3 = "sriov3"
)

var _ = Describe("[Serial]Multus", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var nodes *k8sv1.NodeList

	defaultInterface := v1.Interface{
		Name: "default",
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Masquerade: &v1.InterfaceMasquerade{},
		},
	}

	linuxBridgeInterface := v1.Interface{
		Name: "linux-bridge",
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

	linuxBridgeNetwork := v1.Network{
		Name: "linux-bridge",
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: "linux-bridge-net-vlan100",
			},
		},
	}

	tests.BeforeAll(func() {
		tests.BeforeTestCleanup()

		nodes = tests.GetAllSchedulableNodes(virtClient)
		Expect(len(nodes.Items) > 0).To(BeTrue())

		configureNodeNetwork(virtClient)

		result := virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "linux-bridge-net-vlan100")).
			Body([]byte(fmt.Sprintf(linuxBridgeConfCRD, "linux-bridge-net-vlan100", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())

		// Create ptp crds with tuning plugin enabled in two different namespaces
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "ptp-conf-1")).
			Body([]byte(fmt.Sprintf(ptpConfCRD, "ptp-conf-1", tests.NamespaceTestDefault))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())

		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestAlternative, "ptp-conf-2")).
			Body([]byte(fmt.Sprintf(ptpConfCRD, "ptp-conf-2", tests.NamespaceTestAlternative))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

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

	createVMIOnNode := func(interfaces []v1.Interface, networks []v1.Network) *v1.VirtualMachineInstance {
		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskAlpine), "#!/bin/bash\n")
		vmi.Spec.Domain.Devices.Interfaces = interfaces
		vmi.Spec.Networks = networks

		// Arbitrarily select one compute node in the cluster, on which it is possible to create a VMI
		// (i.e. a schedulable node).
		nodeName := nodes.Items[0].Name
		tests.StartVmOnNode(vmi, nodeName)

		return vmi
	}

	Describe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance using different types of interfaces.", func() {
		Context("VirtualMachineInstance with cni ptp plugin interface", func() {
			It("[test_id:1751]should create a virtual machine with one interface", func() {
				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "ptp-conf-1"},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, libnet.WithIPv6(console.LoginToCirros))

				Expect(libnet.PingFromVMConsole(detachedVMI, "10.1.1.1")).To(Succeed())
			})

			It("[test_id:1752]should create a virtual machine with one interface with network definition from different namespace", func() {
				tests.SkipIfOpenShift4("OpenShift 4 does not support usage of the network definition from the different namespace")
				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: fmt.Sprintf("%s/%s", tests.NamespaceTestAlternative, "ptp-conf-2")},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, libnet.WithIPv6(console.LoginToCirros))

				Expect(libnet.PingFromVMConsole(detachedVMI, "10.1.1.1")).To(Succeed())
			})

			It("[test_id:1753]should create a virtual machine with two interfaces", func() {
				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					defaultNetwork,
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "ptp-conf-1"},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, libnet.WithIPv6(console.LoginToCirros))

				cmdCheck := "sudo /sbin/cirros-dhcpc up eth1 > /dev/null\n"
				err = console.SafeExpectBatch(detachedVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: cmdCheck},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "ip addr show eth1 | grep 10.1.1 | wc -l\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 15)
				Expect(err).ToNot(HaveOccurred())

				By("checking virtual machine instance has two interfaces")
				Expect(checkInterface(detachedVMI, "eth0")).To(Succeed())
				Expect(checkInterface(detachedVMI, "eth1")).To(Succeed())

				Expect(libnet.PingFromVMConsole(detachedVMI, "10.1.1.1")).To(Succeed())
			})
		})

		Context("VirtualMachineInstance with multus network as default network", func() {
			It("[test_id:1751]should create a virtual machine with one interface with multus default network definition", func() {
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: fmt.Sprintf("%s/%s", tests.NamespaceTestDefault, "ptp-conf-1"),
							Default:     true,
						}}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, libnet.WithIPv6(console.LoginToCirros))

				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				Expect(libnet.PingFromVMConsole(detachedVMI, "10.1.1.1")).To(Succeed())

				By("checking virtual machine instance only has one interface")
				// lo0, eth0
				err = console.SafeExpectBatch(detachedVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
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
			It("[test_id:1705]should configure valid custom MAC address on ptp interface when using tuning plugin", func() {
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
							NetworkName: "ptp-conf-1",
						},
					},
				}

				interfaces := []v1.Interface{ptpInterface}
				networks := []v1.Network{ptpNetwork}

				By("Creating a VM with custom MAC address on its ptp interface.")
				interfaces[0].MacAddress = customMacAddress
				vmiOne := createVMIOnNode(interfaces, networks)
				tests.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)

				By("Configuring static IP address to ptp interface.")
				Expect(configInterface(vmiOne, "eth0", "10.1.1.1/24")).To(Succeed())

				By("Verifying the desired custom MAC is the one that was actually configured on the interface.")
				ipLinkShow := fmt.Sprintf("ip link show eth0 | grep -i \"%s\" | wc -l\n", customMacAddress)
				err = console.SafeExpectBatch(vmiOne, []expect.Batcher{
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
			})
		})

		Context("VirtualMachineInstance with Linux bridge plugin interface", func() {
			It("[test_id:1577]should create two virtual machines with one interface", func() {
				By("checking virtual machine instance can ping the secondary virtual machine instance using Linux bridge CNI plugin")
				interfaces := []v1.Interface{linuxBridgeInterface}
				networks := []v1.Network{linuxBridgeNetwork}

				vmiOne := createVMIOnNode(interfaces, networks)
				vmiTwo := createVMIOnNode(interfaces, networks)

				tests.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)
				tests.WaitUntilVMIReady(vmiTwo, console.LoginToAlpine)

				Expect(configInterface(vmiOne, "eth0", "10.1.1.1/24")).To(Succeed())
				By("checking virtual machine interface eth0 state")
				Expect(checkInterface(vmiOne, "eth0")).To(Succeed())

				Expect(configInterface(vmiTwo, "eth0", "10.1.1.2/24")).To(Succeed())
				By("checking virtual machine interface eth0 state")
				Expect(checkInterface(vmiTwo, "eth0")).To(Succeed())

				By("ping between virtual machines")
				Expect(libnet.PingFromVMConsole(vmiOne, "10.1.1.2")).To(Succeed())
			})

			It("[test_id:1578]should create two virtual machines with two interfaces", func() {
				By("checking the first virtual machine instance can ping 10.1.1.2 using Linux bridge CNI plugin")
				interfaces := []v1.Interface{
					defaultInterface,
					linuxBridgeInterface,
				}
				networks := []v1.Network{
					defaultNetwork,
					linuxBridgeNetwork,
				}

				vmiOne := createVMIOnNode(interfaces, networks)
				vmiTwo := createVMIOnNode(interfaces, networks)

				tests.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)
				tests.WaitUntilVMIReady(vmiTwo, console.LoginToAlpine)

				Expect(configInterface(vmiOne, "eth1", "10.1.1.1/24")).To(Succeed())
				By("checking virtual machine interface eth1 state")
				Expect(checkInterface(vmiOne, "eth1")).To(Succeed())

				Expect(configInterface(vmiTwo, "eth1", "10.1.1.2/24")).To(Succeed())
				By("checking virtual machine interface eth1 state")
				Expect(checkInterface(vmiTwo, "eth1")).To(Succeed())

				By("ping between virtual machines")
				Expect(libnet.PingFromVMConsole(vmiOne, "10.1.1.2")).To(Succeed())
			})
		})

		Context("VirtualMachineInstance with Linux bridge CNI plugin interface and custom MAC address.", func() {
			customMacAddress := "50:00:00:00:90:0d"
			It("[test_id:676]should configure valid custom MAC address on Linux bridge CNI interface.", func() {
				By("Creating a VM with Linux bridge CNI network interface and default MAC address.")
				vmiTwo := libvmi.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterface),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloudUserData(tests.GetGuestAgentUserData(), false),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByDevice("eth1", "10.1.1.2/24"), false))
				vmiTwo = tests.StartVmOnNode(vmiTwo, nodes.Items[0].Name)

				By("Creating another VM with custom MAC address on its Linux bridge CNI interface.")
				linuxBridgeInterfaceWithCustomMac := linuxBridgeInterface
				linuxBridgeInterfaceWithCustomMac.MacAddress = customMacAddress
				vmiOne := libvmi.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterfaceWithCustomMac),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloudUserData(tests.GetGuestAgentUserData(), false),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByMac(linuxBridgeInterfaceWithCustomMac.Name, customMacAddress, "10.1.1.1/24"), false))
				vmiOne = tests.StartVmOnNode(vmiOne, nodes.Items[0].Name)

				vmiOne = tests.WaitUntilVMIReady(vmiOne, console.LoginToFedora)
				tests.WaitAgentConnected(virtClient, vmiOne)

				By("Verifying the desired custom MAC is the one that were actually configured on the interface.")
				vmiIfaceStatusByName := libvmi.IndexInterfaceStatusByName(vmiOne)
				Expect(vmiIfaceStatusByName).To(HaveKey(linuxBridgeInterfaceWithCustomMac.Name), "should set linux bridge interface with the custom MAC address at VMI Status")
				Expect(vmiIfaceStatusByName[linuxBridgeInterfaceWithCustomMac.Name].MAC).To(Equal(customMacAddress), "should set linux bridge interface with the custom MAC address at VMI")

				By("Verifying the desired custom MAC is not configured inside the pod namespace.")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, vmiOne.Namespace)
				out, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"sh", "-c", "ip a"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Contains(out, customMacAddress)).To(BeFalse())

				By("Ping from the VM with the custom MAC to the other VM.")
				tests.WaitUntilVMIReady(vmiTwo, console.LoginToFedora)
				Expect(libnet.PingFromVMConsole(vmiOne, "10.1.1.2")).To(Succeed())
			})
		})

		Context("Single VirtualMachineInstance with Linux bridge CNI plugin interface", func() {
			It("[test_id:1756]should report all interfaces in Status", func() {
				interfaces := []v1.Interface{
					defaultInterface,
					linuxBridgeInterface,
				}
				networks := []v1.Network{
					defaultNetwork,
					linuxBridgeNetwork,
				}

				vmiOne := createVMIOnNode(interfaces, networks)

				tests.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)

				updatedVmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmiOne.Name, &metav1.GetOptions{})
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
				Expect(interfacesByName["default"].MAC).To(Not(Equal(interfacesByName["linux-bridge"].MAC)))

				err = console.SafeExpectBatch(updatedVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("ip addr show eth0 | grep %s | wc -l", interfacesByName["default"].MAC)},
					&expect.BExp{R: "1"},
				}, 15)
				err = console.SafeExpectBatch(updatedVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("ip addr show eth1 | grep %s | wc -l", interfacesByName["linux-bridge"].MAC)},
					&expect.BExp{R: "1"},
				}, 15)
			})
		})

		Context("VirtualMachineInstance with invalid MAC address", func() {

			It("[test_id:1713]should failed to start with invalid MAC address", func() {
				By("Start VMI")
				linuxBridgeIfIdx := 1

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskAlpine), "#!/bin/bash\n")
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					linuxBridgeInterface,
				}
				vmi.Spec.Domain.Devices.Interfaces[linuxBridgeIfIdx].MacAddress = "de:00c:00c:00:00:de:abc"

				vmi.Spec.Networks = []v1.Network{
					defaultNetwork,
					linuxBridgeNetwork,
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred())
				testErr := err.(*errors.StatusError)
				Expect(testErr.ErrStatus.Reason).To(BeEquivalentTo("Invalid"))
			})
		})
	})

	Describe("[rfe_id:1758][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance definition", func() {
		Context("with qemu guest agent", func() {

			It("[test_id:1757] should report guest interfaces in VMI status", func() {
				interfaces := []v1.Interface{
					defaultInterface,
					linuxBridgeInterface,
				}
				networks := []v1.Network{
					defaultNetwork,
					linuxBridgeNetwork,
				}

				v4Mask := "/24"
				ep1Ip := "1.0.0.10"
				ep2Ip := "1.0.0.11"
				ep1Cidr := ep1Ip + v4Mask
				ep2Cidr := ep2Ip + v4Mask

				v6Mask := "/64"
				ep1IpV6 := "fe80::ce3d:82ff:fe52:24c0"
				ep2IpV6 := "fe80::ce3d:82ff:fe52:24c1"
				ep1CidrV6 := ep1IpV6 + v6Mask
				ep2CidrV6 := ep2IpV6 + v6Mask

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
		    curl %s > /lib64/libpixman-1.so.0
                    systemd-run --unit=guestagent /usr/local/bin/qemu-ga
                `, ep1Cidr, ep2Cidr, ep1CidrV6, ep2CidrV6, tests.GetUrl(tests.GuestAgentHttpUrl), tests.GetUrl(tests.PixmanUrl))
				agentVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskFedora), userdata)

				agentVMI.Spec.Domain.Devices.Interfaces = interfaces
				agentVMI.Spec.Networks = networks
				agentVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

				By("Starting a VirtualMachineInstance")
				agentVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				tests.WaitForSuccessfulVMIStart(agentVMI)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, agentVMI)

				getOptions := &metav1.GetOptions{}
				Eventually(func() bool {
					updatedVmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, getOptions)
					if err != nil {
						return false
					}
					return len(updatedVmi.Status.Interfaces) == 4
				}, 420*time.Second, 4).Should(BeTrue(), "Should have interfaces in vmi status")

				updatedVmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, getOptions)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(updatedVmi.Status.Interfaces)).To(Equal(4))
				interfaceByIfcName := make(map[string]v1.VirtualMachineInstanceNetworkInterface)
				for _, ifc := range updatedVmi.Status.Interfaces {
					interfaceByIfcName[ifc.InterfaceName] = ifc
				}
				Expect(interfaceByIfcName["eth0"].Name).To(Equal("default"))
				Expect(interfaceByIfcName["eth0"].InterfaceName).To(Equal("eth0"))

				Expect(interfaceByIfcName["eth1"].Name).To(Equal("linux-bridge"))
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

var _ = Describe("[Serial]SRIOV", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	sriovResourceName := os.Getenv("SRIOV_RESOURCE_NAME")

	if sriovResourceName == "" {
		sriovResourceName = "openshift.io/sriov_net"
	}

	createNetworkAttachementDefinition := func(networkName string, namespace string, networkAttachmentDefinition string) error {
		return virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, namespace, networkName)).
			Body([]byte(fmt.Sprintf(networkAttachmentDefinition, networkName, namespace, sriovResourceName))).
			Do().Error()
	}

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
		// Check if the hardware supports SRIOV
		sriovcheck := checkSriovEnabled(virtClient, sriovResourceName)
		if !sriovcheck {
			Skip("Sriov is not enabled in this environment. Skip these tests using - export FUNC_TEST_ARGS='--ginkgo.skip=SRIOV'")
		}

		Expect(createNetworkAttachementDefinition(sriovnet1, tests.NamespaceTestDefault, sriovConfCRD)).To((Succeed()), "should successfully create the network")
		Expect(createNetworkAttachementDefinition(sriovnet2, tests.NamespaceTestDefault, sriovConfCRD)).To((Succeed()), "should successfully create the network")
		Expect(createNetworkAttachementDefinition(sriovnet3, tests.NamespaceTestDefault, sriovLinkEnableConfCRD)).To((Succeed()), "should successfully create the network")
	})

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

	Context("VirtualMachineInstance with sriov plugin interface", func() {

		getSriovVmi := func(networks []string, cloudInitNetworkData string) *v1.VirtualMachineInstance {

			withVmiOptions := []libvmi.Option{
				libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkData, false),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			}
			// sriov network interfaces
			for _, name := range networks {
				withVmiOptions = append(withVmiOptions,
					libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(name)),
					libvmi.WithNetwork(libvmi.MultusNetwork(name)),
				)
			}
			return libvmi.NewSriovFedora(withVmiOptions...)
		}

		startVmi := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			return vmi
		}

		waitVmi := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			// Need to wait for cloud init to finish and start the agent inside the vmi.
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToFedora))
			tests.WaitAgentConnected(virtClient, vmi)
			return vmi
		}

		checkDefaultInterfaceInPod := func(vmi *v1.VirtualMachineInstance) {
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

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
		}

		checkInterfacesInGuest := func(vmi *v1.VirtualMachineInstance, interfaces []string) {
			for _, iface := range interfaces {
				Expect(checkInterface(vmi, iface)).To(Succeed())
			}
		}

		It("[test_id:1754]should create a virtual machine with sriov interface", func() {
			vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi = startVmi(vmi)
			vmi = waitVmi(vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(validatePodKubevirtResourceName(virtClient, vmiPod, sriovnet1, sriovResourceName)).To(Succeed())

			checkDefaultInterfaceInPod(vmi)

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})

			// there is little we can do beyond just checking two devices are present: PCI slots are different inside
			// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
			// it's hard to match them.
		})

		It("[test_id:1754]should create a virtual machine with sriov interface with all pci devices on the root bus", func() {
			vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi.Annotations = map[string]string{
				v1.PlacePCIDevicesOnRootComplex: "true",
			}
			vmi = startVmi(vmi)
			vmi = waitVmi(vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(validatePodKubevirtResourceName(virtClient, vmiPod, sriovnet1, sriovResourceName)).To(Succeed())

			checkDefaultInterfaceInPod(vmi)

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})

			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			rootPortController := []api.Controller{}
			for _, c := range domSpec.Devices.Controllers {
				if c.Model == "pcie-root-port" {
					rootPortController = append(rootPortController, c)
				}
			}
			Expect(rootPortController).To(HaveLen(0), "libvirt should not add additional buses to the root one")
		})

		It("[test_id:3959]should create a virtual machine with sriov interface and dedicatedCPUs", func() {
			// In addition to verifying that we can start a VMI with CPU pinning
			// this also tests if we've correctly calculated the overhead for VFIO devices.
			vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				DedicatedCPUPlacement: true,
			}
			vmi = startVmi(vmi)
			vmi = waitVmi(vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(validatePodKubevirtResourceName(virtClient, vmiPod, sriovnet1, sriovResourceName)).To(Succeed())

			checkDefaultInterfaceInPod(vmi)

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})
		})

		It("[test_id:3985]should create a virtual machine with sriov interface with custom MAC address", func() {
			const mac = "de:ad:00:00:be:ef"
			vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

			vmi = startVmi(vmi)
			vmi = waitVmi(vmi)

			var interfaceName string
			Eventually(func() error {
				var err error
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				interfaceName, err = getInterfaceNameByMAC(vmi, mac)
				return err
			}, 140*time.Second, 5*time.Second).Should(Succeed())

			By("checking virtual machine instance has an interface with the requested MAC address")
			Expect(checkMacAddress(vmi, interfaceName, mac)).To(Succeed())
			By("checking virtual machine instance reports the expected network name")
			Expect(getInterfaceNetworkNameByMAC(vmi, mac)).To(Equal(sriovnet1))
		})

		It("[test_id:1755]should create a virtual machine with two sriov interfaces referring the same resource", func() {
			sriovNetworks := []string{sriovnet1, sriovnet2}
			vmi := getSriovVmi(sriovNetworks, defaultCloudInitNetworkData())
			vmi = startVmi(vmi)
			vmi = waitVmi(vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variables are defined in pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			for _, name := range sriovNetworks {
				Expect(validatePodKubevirtResourceName(virtClient, vmiPod, name, sriovResourceName)).To(Succeed())
			}

			checkDefaultInterfaceInPod(vmi)

			By("checking virtual machine instance has three interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1", "eth2"})

			// there is little we can do beyond just checking three devices are present: PCI slots are different inside
			// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
			// it's hard to match them.
		})

		// createSriovVMs instantiates two VMs connected through SR-IOV.
		// Note: test case assumes interconnectivity between SR-IOV
		// interfaces. It can be achieved either by configuring the external switch
		// properly, or via in-PF switching for VFs (works for some NIC models)
		createSriovVMs := func(networkNameA, networkNameB, cidrA, cidrB string) (*v1.VirtualMachineInstance, *v1.VirtualMachineInstance) {
			// Explicitly choose different random mac addresses instead of relying on kubemacpool to do it:
			// 1) we don't at the moment deploy kubemacpool in kind providers
			// 2) even if we would do, it's probably a good idea to have the suite not depend on this fact
			//
			// This step is needed to guarantee that no VFs on the PF carry a duplicate MAC address that may affect
			// ability of VMIs to send and receive ICMP packets on their ports.
			mac1, err := tests.GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())

			mac2, err := tests.GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())

			// start peer machines with sriov interfaces from the same resource pool
			// manually configure IP/link on sriov interfaces because there is
			// no DHCP server to serve the address to the guest
			vmi1 := getSriovVmi([]string{networkNameA}, cloudInitNetworkDataWithStaticIPsByMac(networkNameA, mac1.String(), cidrA))
			vmi2 := getSriovVmi([]string{networkNameB}, cloudInitNetworkDataWithStaticIPsByMac(networkNameB, mac2.String(), cidrB))

			vmi1.Spec.Domain.Devices.Interfaces[1].MacAddress = mac1.String()
			vmi2.Spec.Domain.Devices.Interfaces[1].MacAddress = mac2.String()

			vmi1 = startVmi(vmi1)
			vmi2 = startVmi(vmi2)
			vmi1 = waitVmi(vmi1)
			vmi2 = waitVmi(vmi2)

			vmi1, err = virtClient.VirtualMachineInstance(vmi1.Namespace).Get(vmi1.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi2, err = virtClient.VirtualMachineInstance(vmi2.Namespace).Get(vmi2.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			return vmi1, vmi2
		}

		It("[test_id:3956]should connect to another machine with sriov interface over IPv4", func() {
			cidrA := "192.168.1.1/24"
			cidrB := "192.168.1.2/24"
			//create two vms on the smae sriov network
			vmi1, vmi2 := createSriovVMs(sriovnet3, sriovnet3, cidrA, cidrB)

			assert.XFail("suspected cloud-init issue: https://github.com/kubevirt/kubevirt/issues/4642", func() {
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi1, cidrToIP(cidrB))
				}, 15*time.Second, time.Second).Should(Succeed())
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi2, cidrToIP(cidrA))
				}, 15*time.Second, time.Second).Should(Succeed())
			})
		})

		It("[test_id:3957]should connect to another machine with sriov interface over IPv6", func() {
			cidrA := "fc00::1/64"
			cidrB := "fc00::2/64"
			//create two vms on the smae sriov network
			vmi1, vmi2 := createSriovVMs(sriovnet3, sriovnet3, cidrA, cidrB)

			assert.XFail("suspected cloud-init issue: https://github.com/kubevirt/kubevirt/issues/4642", func() {
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi1, cidrToIP(cidrB))
				}, 15*time.Second, time.Second).Should(Succeed())
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi2, cidrToIP(cidrA))
				}, 15*time.Second, time.Second).Should(Succeed())
			})
		})

		Context("With VLAN", func() {
			const (
				cidrVlaned1          = "192.168.0.1/24"
				sriovVlanNetworkName = "sriov-vlan"
			)

			BeforeEach(func() {
				createNetworkAttachementDefinition(sriovVlanNetworkName, tests.NamespaceTestDefault, sriovConfVlanCRD)
			})

			It("should be able to ping between two VMIs with the same VLAN over SRIOV network", func() {
				_, vlanedVMI2 := createSriovVMs(sriovVlanNetworkName, sriovVlanNetworkName, cidrVlaned1, "192.168.0.2/24")

				assert.XFail("suspected cloud-init issue: https://github.com/kubevirt/kubevirt/issues/4642", func() {
					By("pinging from vlanedVMI2 and the anonymous vmi over vlan")
					Eventually(func() error {
						return libnet.PingFromVMConsole(vlanedVMI2, cidrToIP(cidrVlaned1))
					}, 15*time.Second, time.Second).ShouldNot(HaveOccurred())
				})
			})

			It("should NOT be able to ping between Vlaned VMI and a non Vlaned VMI", func() {
				_, nonVlanedVMI := createSriovVMs(sriovVlanNetworkName, sriovnet3, cidrVlaned1, "192.168.0.3/24")

				assert.XFail("suspected cloud-init issue: https://github.com/kubevirt/kubevirt/issues/4642", func() {
					By("pinging between nonVlanedVMIand the anonymous vmi")
					Eventually(func() error {
						return libnet.PingFromVMConsole(nonVlanedVMI, cidrToIP(cidrVlaned1))
					}, 15*time.Second, time.Second).Should(HaveOccurred())
				})
			})
		})
	})
})

var _ = Describe("[Serial]Macvtap", func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var macvtapLowerDevice string
	var macvtapNetworkName string

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		macvtapLowerDevice = "eth0"
		macvtapNetworkName = "net1"

		// cleanup the environment
		tests.BeforeTestCleanup()
	})

	BeforeEach(func() {
		tests.EnableFeatureGate(virtconfig.MacvtapGate)
	})

	BeforeEach(func() {
		result := virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, macvtapNetworkName)).
			Body([]byte(fmt.Sprintf(macvtapNetworkConf, macvtapNetworkName, tests.NamespaceTestDefault, macvtapLowerDevice, macvtapNetworkName))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred(), "A macvtap network named %s should be provisioned", macvtapNetworkName)
	})

	AfterEach(func() {
		tests.DisableFeatureGate(virtconfig.MacvtapGate)
	})

	newCirrosVMIWithMacvtapNetwork := func(macvtapNetworkName string) *v1.VirtualMachineInstance {
		return libvmi.NewCirros(
			libvmi.WithInterface(
				*v1.DefaultMacvtapNetworkInterface(macvtapNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName)))
	}

	newCirrosVMIWithExplicitMac := func(macvtapNetworkName string, mac string) *v1.VirtualMachineInstance {
		return libvmi.NewCirros(
			libvmi.WithInterface(
				*libvmi.InterfaceWithMac(
					v1.DefaultMacvtapNetworkInterface(macvtapNetworkName), mac)),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName)))
	}

	newFedoraVMIWithExplicitMacAndGuestAgent := func(macvtapNetworkName string, mac string) *v1.VirtualMachineInstance {
		return libvmi.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithInterface(
				*libvmi.InterfaceWithMac(
					v1.DefaultMacvtapNetworkInterface(macvtapNetworkName), mac)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName)),
			libvmi.WithCloudInitNoCloudUserData(tests.GetGuestAgentUserData(), false))
	}

	createCirrosVMIStaticIPOnNode := func(nodeName string, networkName string, ifaceName string, ipCIDR string, mac *string) *v1.VirtualMachineInstance {
		var vmi *v1.VirtualMachineInstance
		if mac != nil {
			vmi = newCirrosVMIWithExplicitMac(networkName, *mac)
		} else {
			vmi = newCirrosVMIWithMacvtapNetwork(networkName)
		}
		vmi = tests.WaitUntilVMIReady(
			tests.StartVmOnNode(vmi, nodeName),
			console.LoginToCirros)
		// configure the client VMI
		Expect(configVMIInterfaceWithSudo(vmi, ifaceName, ipCIDR)).To(Succeed())
		return vmi
	}

	createCirrosVMIRandomNode := func(networkName string, mac string) (*v1.VirtualMachineInstance, error) {
		runningVMI := tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(
			newCirrosVMIWithExplicitMac(networkName, mac),
			180,
			false)
		err := console.LoginToCirros(runningVMI)
		return runningVMI, err
	}

	createFedoraVMIRandomNode := func(networkName string, mac string) (*v1.VirtualMachineInstance, error) {
		runningVMI := tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(
			newFedoraVMIWithExplicitMacAndGuestAgent(networkName, mac),
			180,
			false)
		err := console.LoginToFedora(runningVMI)
		return runningVMI, err
	}

	Context("a virtual machine with one macvtap interface, with a custom MAC address", func() {
		var serverVMI *v1.VirtualMachineInstance
		var chosenMAC string
		var serverCIDR string
		var nodeList *k8sv1.NodeList
		var nodeName string

		BeforeEach(func() {
			nodeList = tests.GetAllSchedulableNodes(virtClient)
			Expect(nodeList.Items).NotTo(BeEmpty(), "schedulable kubernetes nodes must be present")
			nodeName = nodeList.Items[0].Name
			chosenMAC = "de:ad:00:00:be:af"
			serverCIDR = "192.0.2.102/24"

			serverVMI = createCirrosVMIStaticIPOnNode(nodeName, macvtapNetworkName, "eth0", serverCIDR, &chosenMAC)
		})

		It("should have the specified MAC address reported back via the API", func() {
			Expect(len(serverVMI.Status.Interfaces)).To(Equal(1), "should have a single interface")
			Expect(serverVMI.Status.Interfaces[0].MAC).To(Equal(chosenMAC), "the expected MAC address should be set in the VMI")
		})

		Context("and another virtual machine connected to the same network", func() {
			var clientVMI *v1.VirtualMachineInstance
			BeforeEach(func() {
				clientVMI = createCirrosVMIStaticIPOnNode(nodeName, macvtapNetworkName, "eth0", "192.0.2.101/24", nil)
			})

			It("can communicate with the virtual machine in the same network", func() {
				Expect(libnet.PingFromVMConsole(clientVMI, cidrToIP(serverCIDR))).To(Succeed())
			})
		})
	})

	Context("VMI migration", func() {
		var clientVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			nodes := tests.GetAllSchedulableNodes(virtClient)
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

			if len(nodes.Items) < 2 {
				Skip("Migration tests require at least 2 nodes")
			}

			if !tests.HasLiveMigration() {
				Skip("Migration tests require the 'LiveMigration' feature gate")
			}
		})

		BeforeEach(func() {
			macAddress := "02:03:04:05:06:07"
			clientVMI, err = createCirrosVMIRandomNode(macvtapNetworkName, macAddress)
			Expect(err).NotTo(HaveOccurred(), "must succeed creating a VMI on a random node")
		})

		It("should be successful when the VMI MAC address is defined in its spec", func() {
			By("starting the migration")
			migration := tests.NewRandomMigration(clientVMI.Name, clientVMI.Namespace)
			migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, migrationWaitTime)

			// check VMI, confirm migration state
			tests.ConfirmVMIPostMigration(virtClient, clientVMI, migrationUID)
		})

		Context("with live traffic", func() {
			var serverVMI *v1.VirtualMachineInstance
			var serverIP string

			getVMMacvtapIfaceIP := func(vmi *v1.VirtualMachineInstance, macAddress string) (string, error) {
				var vmiIP string
				err := wait.PollImmediate(time.Second, 2*time.Minute, func() (done bool, err error) {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
					if err != nil {
						return false, err
					}

					for _, iface := range vmi.Status.Interfaces {
						if iface.MAC == macAddress {
							vmiIP = iface.IP
							return true, nil
						}
					}

					return false, nil
				})
				if err != nil {
					return "", err
				}

				return vmiIP, nil
			}

			BeforeEach(func() {
				macAddress := "02:03:04:05:06:aa"

				serverVMI, err = createFedoraVMIRandomNode(macvtapNetworkName, macAddress)
				Expect(err).NotTo(HaveOccurred(), "must have succeeded creating a fedora VMI on a random node")
				Expect(serverVMI.Status.Interfaces).NotTo(BeEmpty(), "a migrate-able VMI must have network interfaces")

				serverIP, err = getVMMacvtapIfaceIP(serverVMI, macAddress)
				Expect(err).NotTo(HaveOccurred(), "should have managed to figure out the IP of the server VMI")
			})

			BeforeEach(func() {
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *before* migrating the VMI")
			})

			It("should keep connectivity after a migration", func() {
				migration := tests.NewRandomMigration(serverVMI.Name, serverVMI.GetNamespace())
				_ = tests.RunMigrationAndExpectCompletion(virtClient, migration, migrationWaitTime)

				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *after* migrating the VMI")
			})
		})
	})
})

func cidrToIP(cidr string) string {
	ip, _, err := net.ParseCIDR(cidr)
	Expect(err).ToNot(HaveOccurred(), "Should be able to parse IP and prefix length from CIDR")
	return ip.String()
}

func configVMIInterfaceWithSudo(vmi *v1.VirtualMachineInstance, interfaceName, interfaceAddress string) error {
	return configInterface(vmi, interfaceName, interfaceAddress, "sudo ")
}

func configInterface(vmi *v1.VirtualMachineInstance, interfaceName, interfaceAddress string, userModifierPrefix ...string) error {
	setStaticIpCmd := fmt.Sprintf("%sip addr add %s dev %s\n", strings.Join(userModifierPrefix, " "), interfaceAddress, interfaceName)
	err := runSafeCommand(vmi, setStaticIpCmd)

	if err != nil {
		return fmt.Errorf("could not configure address %s for interface %s on VMI %s: %w", interfaceAddress, interfaceName, vmi.Name, err)
	}

	return setInterfaceUp(vmi, interfaceName)
}

func checkInterface(vmi *v1.VirtualMachineInstance, interfaceName string) error {
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	err := runSafeCommand(vmi, cmdCheck)

	if err != nil {
		return fmt.Errorf("could not check interface: interface %s was not found in the VMI %s: %w", interfaceName, vmi.Name, err)
	}

	return nil
}

func checkMacAddress(vmi *v1.VirtualMachineInstance, interfaceName, macAddress string) error {
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	err := console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: macAddress},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)

	if err != nil {
		return fmt.Errorf("could not check mac address of interface %s: MAC %s was not found in the VMI %s: %w", interfaceName, macAddress, vmi.Name, err)
	}

	return nil
}

func setInterfaceUp(vmi *v1.VirtualMachineInstance, interfaceName string) error {
	setUpCmd := fmt.Sprintf("ip link set %s up\n", interfaceName)
	err := runSafeCommand(vmi, setUpCmd)

	if err != nil {
		return fmt.Errorf("could not set interface %s up on VMI %s: %w", interfaceName, vmi.Name, err)
	}

	return nil
}

func runSafeCommand(vmi *v1.VirtualMachineInstance, command string) error {
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: command},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
}

func getInterfaceNameByMAC(vmi *v1.VirtualMachineInstance, mac string) (string, error) {
	for _, iface := range vmi.Status.Interfaces {
		if iface.MAC == mac {
			return iface.InterfaceName, nil
		}
	}

	return "", fmt.Errorf("could not get sriov interface by MAC: no interface on VMI %s with MAC %s", vmi.Name, mac)
}

func getInterfaceNetworkNameByMAC(vmi *v1.VirtualMachineInstance, macAddress string) string {
	for _, iface := range vmi.Status.Interfaces {
		if iface.MAC == macAddress {
			return iface.Name
		}
	}

	return ""
}

// Tests in Multus suite are expecting a Linux bridge to be available on each node, with iptables allowing
// traffic to go through. This function creates a Daemon Set on the cluster (if not exists yet), this Daemon
// Set creates a linux bridge and configures the firewall. We use iptables-compat in order to work with
// both iptables and newer nftables.
// TODO: Once kubernetes-nmstate is ready, we should use it instead
func configureNodeNetwork(virtClient kubecli.KubevirtClient) {

	// Fetching the kubevirt-operator image from the pod makes this independent from the installation method / image used
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(metav1.ListOptions{LabelSelector: "kubevirt.io=virt-handler"})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())

	virtHandlerImage := pods.Items[0].Spec.Containers[0].Image

	// Privileged DaemonSet configuring host networking as needed
	networkConfigDaemonSet := appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "network-config",
			Namespace: metav1.NamespaceSystem,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "network-config"},
			},
			Template: k8sv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": "network-config"},
				},
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{
							Name: "network-config",
							// Reuse image which is already installed in the cluster. All we need is chroot.
							// Local OKD cluster doesn't allow us to pull from the outside.
							Image: virtHandlerImage,
							Command: []string{
								"sh",
								"-c",
								"set -x; chroot /host ip link add br10 type bridge; chroot /host iptables -I FORWARD 1 -i br10 -j ACCEPT; touch /tmp/ready; sleep INF",
							},
							SecurityContext: &k8sv1.SecurityContext{
								Privileged: pointer.BoolPtr(true),
								RunAsUser:  pointer.Int64Ptr(0),
							},
							ReadinessProbe: &k8sv1.Probe{
								Handler: k8sv1.Handler{
									Exec: &k8sv1.ExecAction{
										Command: []string{"cat", "/tmp/ready"},
									},
								},
							},
							VolumeMounts: []k8sv1.VolumeMount{
								k8sv1.VolumeMount{
									Name:      "host",
									MountPath: "/host",
								},
							},
						},
					},
					Volumes: []k8sv1.Volume{
						k8sv1.Volume{
							Name: "host",
							VolumeSource: k8sv1.VolumeSource{
								HostPath: &k8sv1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					},
					HostNetwork: true,
				},
			},
		},
	}

	// Helper function returning existing network-config DaemonSet if exists
	getNetworkConfigDaemonSet := func() *appsv1.DaemonSet {
		daemonSet, err := virtClient.AppsV1().DaemonSets(metav1.NamespaceSystem).Get(networkConfigDaemonSet.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil
		}
		Expect(err).NotTo(HaveOccurred())
		return daemonSet
	}

	// If the DaemonSet haven't been created yet, do so
	runningNetworkConfigDaemonSet := getNetworkConfigDaemonSet()
	if runningNetworkConfigDaemonSet == nil {
		_, err := virtClient.AppsV1().DaemonSets(metav1.NamespaceSystem).Create(&networkConfigDaemonSet)
		Expect(err).NotTo(HaveOccurred())
	}

	// Make sure that all pods in the Daemon Set finished the configuration
	nodes := tests.GetAllSchedulableNodes(virtClient)
	Eventually(func() int {
		daemonSet := getNetworkConfigDaemonSet()
		return int(daemonSet.Status.NumberAvailable)
	}, time.Minute, time.Second).Should(Equal(len(nodes.Items)))
}

func checkSriovEnabled(virtClient kubecli.KubevirtClient, sriovResourceName string) bool {
	nodes := tests.GetAllSchedulableNodes(virtClient)
	Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

	for _, node := range nodes.Items {
		resourceList := node.Status.Allocatable
		for k, v := range resourceList {
			if string(k) == sriovResourceName {
				if v.Value() > 0 {
					return true
				}
			}
		}
	}
	return false
}

func validatePodKubevirtResourceName(virtClient kubecli.KubevirtClient, vmiPod *k8sv1.Pod, networkName, sriovResourceName string) error {
	out, err := tests.ExecuteCommandOnPod(
		virtClient,
		vmiPod,
		"compute",
		[]string{"sh", "-c", fmt.Sprintf("echo $KUBEVIRT_RESOURCE_NAME_%s", networkName)},
	)
	if err != nil {
		return err
	}

	out = strings.TrimSuffix(out, "\n")
	if out != sriovResourceName {
		return fmt.Errorf("env settings %s didnt match %s", out, sriovResourceName)
	}

	return nil
}

func defaultCloudInitNetworkData() string {
	networkData, err := libnet.CreateDefaultCloudInitNetworkData()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should successfully create default cloud init network data for SRIOV")
	return networkData
}

func cloudInitNetworkDataWithStaticIPsByMac(nicName, macAddress, ipAddress string) string {
	networkData, err := libnet.NewNetworkData(
		libnet.WithEthernet(nicName,
			libnet.WithAddresses(ipAddress),
			libnet.WithNameserverFromCluster(),
			libnet.WithMatchingMAC(macAddress),
		),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should successfully create static IPs by mac address cloud init network data")
	return networkData
}

func cloudInitNetworkDataWithStaticIPsByDevice(deviceName, ipAddress string) string {
	networkData, err := libnet.NewNetworkData(
		libnet.WithEthernet(deviceName,
			libnet.WithAddresses(ipAddress),
			libnet.WithNameserverFromCluster(),
		),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should successfully create static IPs by device name cloud init network data")
	return networkData
}
