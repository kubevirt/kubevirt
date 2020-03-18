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
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	postUrl                = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
	linuxBridgeConfCRD     = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"bridge\", \"bridge\": \"br10\", \"vlan\": 100, \"ipam\": {}},{\"type\": \"tuning\"}]}"}}`
	ptpConfCRD             = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"ptp\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" }},{\"type\": \"tuning\"}]}"}}`
	sriovConfCRD           = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"name\": \"sriov\", \"type\": \"sriov\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	sriovLinkEnableConfCRD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
)

var _ = Describe("Multus", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

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

	Describe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance using different types of interfaces.", func() {
		Context("VirtualMachineInstance with cni ptp plugin interface", func() {

			It("[test_id:1751]should create a virtual machine with one interface", func() {
				By("checking virtual machine instance can ping 10.1.1.1 using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "ptp-conf-1"},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

				pingVirtualMachine(detachedVMI, "10.1.1.1", "\\$ ")
			})

			It("[test_id:1752]should create a virtual machine with one interface with network definition from different namespace", func() {
				tests.SkipIfOpenShift4("OpenShift 4 does not support usage of the network definition from the different namespace")
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
						Multus: &v1.MultusNetwork{NetworkName: "ptp-conf-1"},
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

			It("[test_id:1751]should create a virtual machine with one interface with multus default network definition", func() {
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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
			})
		})

		Context("VirtualMachineInstance with Linux bridge plugin interface", func() {

			It("[test_id:1577]should create two virtual machines with one interface", func() {
				By("checking virtual machine instance can ping the secondary virtual machine instance using Linux bridge CNI plugin")
				interfaces := []v1.Interface{linuxBridgeInterface}
				networks := []v1.Network{linuxBridgeNetwork}

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

		Context("VirtualMachineInstance with Linux bridge CNI plugin interface and custom MAC address.", func() {
			interfaces := []v1.Interface{linuxBridgeInterface}
			networks := []v1.Network{linuxBridgeNetwork}
			linuxBridgeIfIdx := 0
			customMacAddress := "50:00:00:00:90:0d"

			It("[test_id:676]should configure valid custom MAC address on Linux bridge CNI interface.", func() {
				By("Creating a VM with Linux bridge CNI network interface and default MAC address.")
				vmiTwo := createVMIOnNode(interfaces, networks)
				tests.WaitUntilVMIReady(vmiTwo, tests.LoggedInAlpineExpecter)

				By("Creating another VM with custom MAC address on its Linux bridge CNI interface.")
				interfaces[linuxBridgeIfIdx].MacAddress = customMacAddress
				vmiOne := createVMIOnNode(interfaces, networks)
				tests.WaitUntilVMIReady(vmiOne, tests.LoggedInAlpineExpecter)

				By("Configuring static IP address to the Linux bridge interface.")
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
				Expect(interfacesByName["default"].MAC).To(Not(Equal(interfacesByName["linux-bridge"].MAC)))

				err = tests.CheckForTextExpecter(updatedVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("ip addr show eth0 | grep %s | wc -l", interfacesByName["default"].MAC)},
					&expect.BExp{R: "1"},
				}, 15)
				err = tests.CheckForTextExpecter(updatedVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("ip addr show eth1 | grep %s | wc -l", interfacesByName["linux-bridge"].MAC)},
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
				linuxBridgeIfIdx := 1

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskAlpine), "#!/bin/bash\n")
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
		Context("with quemu guest agent", func() {

			It("[test_id:1757] should report guest interfaces in VMI status", func() {
				interfaces := []v1.Interface{
					defaultInterface,
					linuxBridgeInterface,
				}
				networks := []v1.Network{
					defaultNetwork,
					linuxBridgeNetwork,
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
                `, ep1Ip, ep2Ip, ep1IpV6, ep2IpV6, tests.GetUrl(tests.GuestAgentHttpUrl))
				agentVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskFedora), userdata)

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

var _ = Describe("SRIOV", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	sriovResourceName := os.Getenv("SRIOV_RESOURCE_NAME")

	if sriovResourceName == "" {
		sriovResourceName = "openshift.io/sriov_net"
	}

	tests.BeforeAll(func() {
		tests.BeforeTestCleanup()
		// Create two sriov networks referring to the same resource name
		result := virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "sriov")).
			Body([]byte(fmt.Sprintf(sriovConfCRD, "sriov", tests.NamespaceTestDefault, sriovResourceName))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "sriov2")).
			Body([]byte(fmt.Sprintf(sriovConfCRD, "sriov2", tests.NamespaceTestDefault, sriovResourceName))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
		result = virtClient.RestClient().
			Post().
			RequestURI(fmt.Sprintf(postUrl, tests.NamespaceTestDefault, "sriov-link-enabled")).
			Body([]byte(fmt.Sprintf(sriovLinkEnableConfCRD, "sriov-link-enabled", tests.NamespaceTestDefault, sriovResourceName))).
			Do()
		Expect(result.Error()).NotTo(HaveOccurred())
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
		getSriovVmi := func(networks []string) (vmi *v1.VirtualMachineInstance) {
			// If we run on a host with Mellanox SR-IOV cards then we'll need to load in corresponding kernel modules.
			// Stop NetworkManager to not interfere with manual IP configuration for SR-IOV interfaces.
			// Use agent to signal about cloud-init phase completion.
			userData := fmt.Sprintf(`#!/bin/sh
			    echo "fedora" |passwd fedora --stdin
			    dnf install -y kernel-modules-$(uname -r)
			    modprobe mlx5_ib
			    systemctl stop NetworkManager
			    mkdir -p /usr/local/bin
			    curl %s > /usr/local/bin/qemu-ga
			    chmod +x /usr/local/bin/qemu-ga
			    systemd-run --unit=guestagent /usr/local/bin/qemu-ga`, tests.GetUrl(tests.GuestAgentHttpUrl))

			ports := []v1.Port{}
			vmi = tests.NewRandomVMIWithMasqueradeInterfaceEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskFedora), userData, ports)

			for _, name := range networks {
				iface := v1.Interface{Name: name, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}
				network := v1.Network{Name: name, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: name}}}
				vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, iface)
				vmi.Spec.Networks = append(vmi.Spec.Networks, network)
			}

			// fedora requires some more memory to boot without kernel panics
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceName("memory")] = resource.MustParse("1024M")

			return
		}

		startVmi := func(vmi *v1.VirtualMachineInstance) {
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			return
		}

		waitVmi := func(vmi *v1.VirtualMachineInstance) {
			// Need to wait for cloud init to finish and start the agent inside the vmi.
			tests.WaitAgentConnected(virtClient, vmi)
			tests.WaitUntilVMIReady(vmi, tests.LoggedInFedoraExpecter)
			return
		}

		checkDefaultInterfaceInPod := func(vmi *v1.VirtualMachineInstance) {
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

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
				checkInterface(vmi, iface, "#")
			}
		}

		It("[test_id:1754]should create a virtual machine with sriov interface", func() {
			vmi := getSriovVmi([]string{"sriov"})
			startVmi(vmi)
			waitVmi(vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
			out, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				"compute",
				[]string{"sh", "-c", "echo $KUBEVIRT_RESOURCE_NAME_sriov"},
			)
			Expect(err).ToNot(HaveOccurred())

			expectedSriovResourceName := fmt.Sprintf("%s\n", sriovResourceName)
			Expect(out).To(Equal(expectedSriovResourceName))

			checkDefaultInterfaceInPod(vmi)

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})

			// there is little we can do beyond just checking two devices are present: PCI slots are different inside
			// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
			// it's hard to match them.
		})

		It("should create a virtual machine with sriov interface and dedicatedCPUs", func() {
			// In addition to verifying that we can start a VMI with CPU pinning
			// this also tests if we've correctly calculated the overhead for VFIO devices.
			vmi := getSriovVmi([]string{"sriov"})
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				DedicatedCPUPlacement: true,
			}
			startVmi(vmi)
			waitVmi(vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
			out, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				"compute",
				[]string{"sh", "-c", "echo $KUBEVIRT_RESOURCE_NAME_sriov"},
			)
			Expect(err).ToNot(HaveOccurred())

			expectedSriovResourceName := fmt.Sprintf("%s\n", sriovResourceName)
			Expect(out).To(Equal(expectedSriovResourceName))

			checkDefaultInterfaceInPod(vmi)

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})

		})

		It("should create a virtual machine with sriov interface with custom MAC address", func() {
			vmi := getSriovVmi([]string{"sriov"})
			vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = "de:ad:00:00:be:ef"
			startVmi(vmi)
			waitVmi(vmi)

			By("checking virtual machine instance has an interface with the requested MAC address")
			checkMacAddress(vmi, "eth1", "de:ad:00:00:be:ef")
		})

		It("[test_id:1755]should create a virtual machine with two sriov interfaces referring the same resource", func() {
			vmi := getSriovVmi([]string{"sriov", "sriov2"})
			startVmi(vmi)
			waitVmi(vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variables are defined in pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
			for _, name := range []string{"sriov", "sriov"} {
				out, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"sh", "-c", fmt.Sprintf("echo $KUBEVIRT_RESOURCE_NAME_%s", name)},
				)
				Expect(err).ToNot(HaveOccurred())

				expectedSriovResourceName := fmt.Sprintf("%s\n", sriovResourceName)
				Expect(out).To(Equal(expectedSriovResourceName))
			}

			checkDefaultInterfaceInPod(vmi)

			By("checking virtual machine instance has three interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1", "eth2"})

			// there is little we can do beyond just checking three devices are present: PCI slots are different inside
			// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
			// it's hard to match them.
		})

		// Note: test case assumes interconnectivity between SR-IOV
		// interfaces. It can be achieved either by configuring the external switch
		//properly, or via in-PF switching for VFs (works for some NIC models)
		It("should connect to another machine with sriov interface", func() {
			// start peer machines with sriov interfaces from the same resource pool
			vmi1 := getSriovVmi([]string{"sriov-link-enabled"})
			vmi2 := getSriovVmi([]string{"sriov-link-enabled"})

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

			vmi1.Spec.Domain.Devices.Interfaces[1].MacAddress = mac1.String()
			vmi2.Spec.Domain.Devices.Interfaces[1].MacAddress = mac2.String()

			startVmi(vmi1)
			startVmi(vmi2)
			waitVmi(vmi1)
			waitVmi(vmi2)

			// manually configure IP/link on sriov interfaces because there is
			// no DHCP server to serve the address to the guest
			configInterface(vmi1, "eth1", "192.168.1.1/24", "#")
			configInterface(vmi2, "eth1", "192.168.1.2/24", "#")

			// now check ICMP goes both ways
			pingVirtualMachine(vmi1, "192.168.1.2", "#")
			pingVirtualMachine(vmi2, "192.168.1.1", "#")
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
	Expect(err).ToNot(HaveOccurred(), "Failed to configure address %s for interface %s on VMI %s", interfaceAddress, interfaceName, vmi.Name)

	cmdCheck = fmt.Sprintf("ip link set %s up\n", interfaceName)
	err = tests.CheckForTextExpecter(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 15)
	Expect(err).ToNot(HaveOccurred(), "Failed to set interface %s up on VMI %s", interfaceName, vmi.Name)
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
	Expect(err).ToNot(HaveOccurred(), "Interface %q was not found in the VMI %s within the given timeout", interfaceName, vmi.Name)
}

func checkMacAddress(vmi *v1.VirtualMachineInstance, interfaceName, macAddress string) {
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "#"},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: macAddress},
		&expect.BExp{R: "#"},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 15)
	Expect(err).ToNot(HaveOccurred(), "MAC %q was not found in the VMI %s within the given timeout", macAddress, vmi.Name)
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
	Expect(err).ToNot(HaveOccurred(), "Failed to ping VMI %s within the given timeout", vmi.Name)
}

// Tests in Multus suite are expecting a Linux bridge to be available on each node, with iptables allowing
// traffic to go through. This function creates a Daemon Set on the cluster (if not exists yet), this Daemon
// Set creates a linux bridge and configures the firewall. We use iptables-compat in order to work with
// both iptables and newer nftables.
// TODO: Once kubernetes-nmstate is ready, we should use it instead
func configureNodeNetwork(virtClient kubecli.KubevirtClient) {

	// Fetching the kubevirt-operator image from the pod makes this independent from the installation method / image used
	pods, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{LabelSelector: "kubevirt.io=virt-operator"})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())

	virtOperatorImage := pods.Items[0].Spec.Containers[0].Image

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
							Image: virtOperatorImage,
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
