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

package network

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	shouldCreateNetwork = "should successfully create the network"
	ipLinkSetDev        = "ip link set dev "
)

const (
	postUrl            = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
	linuxBridgeConfNAD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"%s\", \"bridge\": \"%s\", \"vlan\": %d, \"ipam\": {%s}, \"macspoofchk\": %t, \"mtu\": 1400},{\"type\": \"tuning\"}]}"}}`
	ptpConfNAD         = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"ptp\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"%s\" }},{\"type\": \"tuning\"}]}"}}`
)

const (
	ptpSubnet     = "10.1.1.0/24"
	ptpSubnetMask = "/24"
	ptpSubnetIP1  = "10.1.1.1"
	ptpSubnetIP2  = "10.1.1.2"
	ptpConf1      = "ptp-conf-1"
	ptpConf2      = "ptp-conf-2"
)

const (
	masqueradeIfaceName          = "default"
	linuxBridgeIfaceName         = "linux-bridge"
	linuxBridgeWithIPAMIfaceName = "linux-bridge-with-ipam"
)

const (
	linuxBridgeVlan100Network           = "linux-bridge-net-vlan100"
	linuxBridgeVlan100WithIPAMNetwork   = "linux-bridge-net-ipam"
	linuxBridgeWithMACSpoofCheckNetwork = "linux-br-msc"
)

const (
	bridge10CNIType       = "bridge"
	bridge10Name          = "br10"
	bridge10MacSpoofCheck = false
)

var _ = SIGDescribe("[Serial]Multus", Serial, decorators.Multus, func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var nodes *k8sv1.NodeList

	defaultInterface := v1.Interface{
		Name: masqueradeIfaceName,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Masquerade: &v1.InterfaceMasquerade{},
		},
	}

	linuxBridgeInterface := v1.Interface{
		Name: linuxBridgeIfaceName,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Bridge: &v1.InterfaceBridge{},
		},
	}

	linuxBridgeInterfaceWithIPAM := v1.Interface{
		Name: linuxBridgeWithIPAMIfaceName,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Bridge: &v1.InterfaceBridge{},
		},
	}

	defaultNetwork := v1.Network{
		Name: masqueradeIfaceName,
		NetworkSource: v1.NetworkSource{
			Pod: &v1.PodNetwork{},
		},
	}

	linuxBridgeNetwork := v1.Network{
		Name: linuxBridgeIfaceName,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: linuxBridgeVlan100Network,
			},
		},
	}

	linuxBridgeWithIPAMNetwork := v1.Network{
		Name: linuxBridgeWithIPAMIfaceName,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: linuxBridgeVlan100WithIPAMNetwork,
			},
		},
	}

	createBridgeNetworkAttachmentDefinition := func(namespace, networkName string, bridgeCNIType string, bridgeName string, vlan int, ipam string, macSpoofCheck bool) error {
		bridgeNad := fmt.Sprintf(linuxBridgeConfNAD, networkName, namespace, bridgeCNIType, bridgeName, vlan, ipam, macSpoofCheck)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, bridgeNad)
	}
	createPtpNetworkAttachmentDefinition := func(namespace, networkName, subnet string) error {
		ptpNad := fmt.Sprintf(ptpConfNAD, networkName, namespace, subnet)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, ptpNad)
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		nodes = libnode.GetAllSchedulableNodes(virtClient)
		Expect(nodes.Items).NotTo(BeEmpty())

		const vlanID100 = 100
		Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), linuxBridgeVlan100Network, bridge10CNIType, bridge10Name, vlanID100, "", bridge10MacSpoofCheck)).To(Succeed())

		// Create ptp crds with tuning plugin enabled in two different namespaces
		Expect(createPtpNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), ptpConf1, ptpSubnet)).To(Succeed())
		Expect(createPtpNetworkAttachmentDefinition(testsuite.NamespaceTestAlternative, ptpConf2, ptpSubnet)).To(Succeed())

		// Multus tests need to ensure that old VMIs are gone
		Eventually(func() []v1.VirtualMachineInstance {
			list1, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).List(context.Background(), &v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			list2, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestAlternative).List(context.Background(), &v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return append(list1.Items, list2.Items...)
		}, 6*time.Minute, 1*time.Second).Should(BeEmpty())
	})

	createVMIOnNode := func(interfaces []v1.Interface, networks []v1.Network) *v1.VirtualMachineInstance {
		vmi := libvmi.NewAlpine()
		vmi.Spec.Domain.Devices.Interfaces = interfaces
		vmi.Spec.Networks = networks

		// Arbitrarily select one compute node in the cluster, on which it is possible to create a VMI
		// (i.e. a schedulable node).
		nodeName := nodes.Items[0].Name
		return tests.CreateVmiOnNode(vmi, nodeName)
	}

	Describe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance using different types of interfaces.", func() {
		const ptpGateway = ptpSubnetIP1
		Context("VirtualMachineInstance with cni ptp plugin interface", func() {
			var networkData string
			BeforeEach(func() {
				libnet.SkipWhenClusterNotSupportIpv4()
				networkData, err = libnet.NewNetworkData(
					libnet.WithEthernet("eth0",
						libnet.WithDHCP4Enabled(),
						libnet.WithNameserverFromCluster(),
					),
				)
				Expect(err).NotTo(HaveOccurred())
			})
			It("[test_id:1751]should create a virtual machine with one interface", func() {
				By("checking virtual machine instance can ping using ptp cni plugin")
				detachedVMI := libvmi.NewAlpineWithTestTooling(
					libvmi.WithCloudInitNoCloudNetworkData(networkData),
				)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: ptpConf1},
					}},
				}

				detachedVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitUntilVMIReady(detachedVMI, console.LoginToAlpine)

				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())
			})

			It("[test_id:1752]should create a virtual machine with one interface with network definition from different namespace", func() {
				checks.SkipIfOpenShift4("OpenShift 4 does not support usage of the network definition from the different namespace")
				By("checking virtual machine instance can ping using ptp cni plugin")
				detachedVMI := libvmi.NewAlpineWithTestTooling(
					libvmi.WithCloudInitNoCloudNetworkData(networkData),
				)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: fmt.Sprintf("%s/%s", testsuite.NamespaceTestAlternative, ptpConf2)},
					}},
				}

				detachedVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitUntilVMIReady(detachedVMI, console.LoginToAlpine)

				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())
			})

			It("[test_id:1753]should create a virtual machine with two interfaces", func() {
				By("checking virtual machine instance can ping using ptp cni plugin")
				detachedVMI := libvmi.NewCirros()

				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					defaultNetwork,
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: ptpConf1},
					}},
				}

				detachedVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitUntilVMIReady(detachedVMI, console.LoginToCirros)

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
				Expect(libnet.InterfaceExists(detachedVMI, "eth0")).To(Succeed())
				Expect(libnet.InterfaceExists(detachedVMI, "eth1")).To(Succeed())

				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())
			})
		})

		Context("VirtualMachineInstance with multus network as default network", func() {
			It("[test_id:1751]should create a virtual machine with one interface with multus default network definition", func() {
				libnet.SkipWhenClusterNotSupportIpv4()
				networkData, err := libnet.NewNetworkData(
					libnet.WithEthernet("eth0",
						libnet.WithDHCP4Enabled(),
						libnet.WithNameserverFromCluster(),
					),
				)
				Expect(err).NotTo(HaveOccurred())
				detachedVMI := libvmi.NewAlpineWithTestTooling(
					libvmi.WithCloudInitNoCloudNetworkData(networkData),
				)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: fmt.Sprintf("%s/%s", testsuite.GetTestNamespace(nil), ptpConf1),
							Default:     true,
						}}},
				}

				detachedVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitUntilVMIReady(detachedVMI, console.LoginToAlpine)

				By("checking virtual machine instance can ping using ptp cni plugin")
				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())

				By("checking virtual machine instance only has one interface")
				// lo0, eth0
				err = console.SafeExpectBatch(detachedVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "ip link show | grep -v lo | grep -c UP\n"},
					&expect.BExp{R: "1"},
				}, 15)
				Expect(err).ToNot(HaveOccurred())

				By("checking pod has only one interface")
				// lo0, eth0-nic, k6t-eth0, vnet0
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
							NetworkName: ptpConf1,
						},
					},
				}

				interfaces := []v1.Interface{ptpInterface}
				networks := []v1.Network{ptpNetwork}

				By("Creating a VM with custom MAC address on its ptp interface.")
				interfaces[0].MacAddress = customMacAddress

				vmiOne := createVMIOnNode(interfaces, networks)
				libwait.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)

				By("Configuring static IP address to ptp interface.")
				Expect(configInterface(vmiOne, "eth0", ptpSubnetIP1+ptpSubnetMask)).To(Succeed())

				By("Verifying the desired custom MAC is the one that was actually configured on the interface.")
				ipLinkShow := fmt.Sprintf("ip link show eth0 | grep -i \"%s\" | wc -l\n", customMacAddress)
				err = console.SafeExpectBatch(vmiOne, []expect.Batcher{
					&expect.BSnd{S: ipLinkShow},
					&expect.BExp{R: "1"},
				}, 15)
				Expect(err).ToNot(HaveOccurred())

				By("Verifying the desired custom MAC is not configured inside the pod namespace.")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, testsuite.GetTestNamespace(vmiOne))
				out, err := exec.ExecuteCommandOnPod(
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
			getIfaceIPByNetworkName := func(vmiName, networkName string) (string, error) {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmiName, &metav1.GetOptions{})
				if err != nil {
					return "", err
				}

				for _, iface := range vmi.Status.Interfaces {
					if iface.Name == networkName {
						return iface.IP, nil
					}
				}

				return "", fmt.Errorf("couldn't find iface %s on vmi %s", networkName, vmiName)
			}

			generateIPAMConfig := func(ipamType string, subnet string) string {
				return fmt.Sprintf("\\\"type\\\": \\\"%s\\\", \\\"subnet\\\": \\\"%s\\\"", ipamType, subnet)
			}

			DescribeTable("should be able to ping between two vms", func(interfaces []v1.Interface, networks []v1.Network, ifaceName, staticIPVm1, staticIPVm2 string) {
				if staticIPVm2 == "" || staticIPVm1 == "" {
					ipam := generateIPAMConfig("host-local", ptpSubnet)
					Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), linuxBridgeVlan100WithIPAMNetwork, bridge10CNIType, bridge10Name, 0, ipam, bridge10MacSpoofCheck)).To(Succeed())
				}

				vmiOne := createVMIOnNode(interfaces, networks)
				vmiTwo := createVMIOnNode(interfaces, networks)

				libwait.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)
				libwait.WaitUntilVMIReady(vmiTwo, console.LoginToAlpine)

				Expect(configureAlpineInterfaceIP(vmiOne, ifaceName, staticIPVm1)).To(Succeed())
				By(fmt.Sprintf("checking virtual machine interface %s state", ifaceName))
				Expect(libnet.InterfaceExists(vmiOne, ifaceName)).To(Succeed())

				Expect(configureAlpineInterfaceIP(vmiTwo, ifaceName, staticIPVm2)).To(Succeed())
				By(fmt.Sprintf("checking virtual machine interface %s state", ifaceName))
				Expect(libnet.InterfaceExists(vmiTwo, ifaceName)).To(Succeed())
				ipAddr := ""
				if staticIPVm2 != "" {
					ipAddr, err = libnet.CidrToIP(staticIPVm2)
				} else {
					const secondaryNetworkIndex = 1
					ipAddr, err = getIfaceIPByNetworkName(vmiTwo.Name, networks[secondaryNetworkIndex].Name)
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(ipAddr).ToNot(BeEmpty())

				By("ping between virtual machines")
				Expect(libnet.PingFromVMConsole(vmiOne, ipAddr)).To(Succeed())
			},
				Entry("[test_id:1577]with secondary network only", []v1.Interface{linuxBridgeInterface}, []v1.Network{linuxBridgeNetwork}, "eth0", ptpSubnetIP1+ptpSubnetMask, ptpSubnetIP2+ptpSubnetMask),
				Entry("[test_id:1578]with default network and secondary network", []v1.Interface{defaultInterface, linuxBridgeInterface}, []v1.Network{defaultNetwork, linuxBridgeNetwork}, "eth1", ptpSubnetIP1+ptpSubnetMask, ptpSubnetIP2+ptpSubnetMask),
				Entry("with default network and secondary network with IPAM", []v1.Interface{defaultInterface, linuxBridgeInterfaceWithIPAM}, []v1.Network{defaultNetwork, linuxBridgeWithIPAMNetwork}, "eth1", "", ""),
			)
		})

		Context("VirtualMachineInstance with Linux bridge CNI plugin interface and custom MAC address.", func() {
			customMacAddress := "50:00:00:00:90:0d"

			BeforeEach(func() {
				By("Creating a VM with Linux bridge CNI network interface and default MAC address.")
				vmiTwo := libvmi.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterface),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByDevice("eth1", ptpSubnetIP2+ptpSubnetMask)))
				vmiTwo = tests.CreateVmiOnNode(vmiTwo, nodes.Items[0].Name)
				libwait.WaitUntilVMIReady(vmiTwo, console.LoginToFedora)
			})

			It("[test_id:676]should configure valid custom MAC address on Linux bridge CNI interface.", func() {
				By("Creating another VM with custom MAC address on its Linux bridge CNI interface.")
				linuxBridgeInterfaceWithCustomMac := linuxBridgeInterface
				linuxBridgeInterfaceWithCustomMac.MacAddress = customMacAddress
				vmiOne := libvmi.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterfaceWithCustomMac),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByMac(linuxBridgeInterfaceWithCustomMac.Name, customMacAddress, ptpSubnetIP1+ptpSubnetMask)))
				vmiOne = tests.CreateVmiOnNode(vmiOne, nodes.Items[0].Name)

				vmiOne = libwait.WaitUntilVMIReady(vmiOne, console.LoginToFedora)
				Eventually(matcher.ThisVMI(vmiOne), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Verifying the desired custom MAC is the one that were actually configured on the interface.")
				vmiIfaceStatusByName := libvmi.IndexInterfaceStatusByName(vmiOne)
				Expect(vmiIfaceStatusByName).To(HaveKey(linuxBridgeInterfaceWithCustomMac.Name), "should set linux bridge interface with the custom MAC address at VMI Status")
				Expect(vmiIfaceStatusByName[linuxBridgeInterfaceWithCustomMac.Name].MAC).To(Equal(customMacAddress), "should set linux bridge interface with the custom MAC address at VMI")

				By("Verifying the desired custom MAC is not configured inside the pod namespace.")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, vmiOne.Namespace)
				out, err := exec.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					"compute",
					[]string{"sh", "-c", "ip a"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Contains(out, customMacAddress)).To(BeFalse())

				By("Ping from the VM with the custom MAC to the other VM.")
				Expect(libnet.PingFromVMConsole(vmiOne, ptpSubnetIP2)).To(Succeed())
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

				libwait.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)

				updatedVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmiOne)).Get(context.Background(), vmiOne.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedVmi.Status.Interfaces).To(HaveLen(2))
				interfacesByName := make(map[string]v1.VirtualMachineInstanceNetworkInterface)
				for _, ifc := range updatedVmi.Status.Interfaces {
					interfacesByName[ifc.Name] = ifc
				}

				for _, network := range networks {
					ifc, is_present := interfacesByName[network.Name]
					Expect(is_present).To(BeTrue())
					Expect(ifc.MAC).To(Not(BeZero()))
				}
				Expect(interfacesByName[masqueradeIfaceName].MAC).To(Not(Equal(interfacesByName[linuxBridgeIfaceName].MAC)))
				Expect(runSafeCommand(vmiOne, fmt.Sprintf("ip addr show eth0 | grep %s\n", interfacesByName["default"].MAC))).To(Succeed())
				Expect(runSafeCommand(vmiOne, fmt.Sprintf("ip addr show eth1 | grep %s\n", interfacesByName[linuxBridgeIfaceName].MAC))).To(Succeed())
			})

			It("should have the correct MTU on the secondary interface with no dhcp server", func() {
				getPodInterfaceMtu := func(vmi *v1.VirtualMachineInstance) string {
					vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
					output, err := exec.ExecuteCommandOnPod(
						virtClient,
						vmiPod,
						"compute",
						[]string{"cat", "/sys/class/net/pod72ad293a5c9/mtu"},
					)
					ExpectWithOffset(1, err).ToNot(HaveOccurred())

					return strings.TrimSuffix(output, "\n")
				}

				getVmiInterfaceMtu := func(vmi *v1.VirtualMachineInstance) string {
					res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
						&expect.BSnd{S: fmt.Sprintf("cat %s\n", "/sys/class/net/eth0/mtu")},
						&expect.BExp{R: console.RetValue("[0-9]+")},
					}, 15)
					ExpectWithOffset(1, err).ToNot(HaveOccurred())

					re := regexp.MustCompile("\r\n[0-9]+\r\n")
					mtu := strings.TrimSpace(re.FindString(res[0].Match[0]))
					return mtu

				}

				vmi := libvmi.NewFedora(
					libvmi.WithInterface(linuxBridgeInterface),
					libvmi.WithNetwork(&linuxBridgeNetwork),
				)

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
				Expect(getPodInterfaceMtu(vmi)).To(Equal(getVmiInterfaceMtu(vmi)))
			})
		})

		Context("VirtualMachineInstance with invalid MAC address", func() {

			It("[test_id:1713]should failed to start with invalid MAC address", func() {
				By("Start VMI")
				linuxBridgeIfIdx := 1

				vmi := libvmi.NewAlpine()
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					linuxBridgeInterface,
				}
				vmi.Spec.Domain.Devices.Interfaces[linuxBridgeIfIdx].MacAddress = "de:00c:00c:00:00:de:abc"

				vmi.Spec.Networks = []v1.Network{
					defaultNetwork,
					linuxBridgeNetwork,
				}

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).To(HaveOccurred())
				testErr := err.(*errors.StatusError)
				Expect(testErr.ErrStatus.Reason).To(BeEquivalentTo("Invalid"))
			})
		})

		Context("Security", func() {
			BeforeEach(func() {
				const (
					bridge11CNIType       = "cnv-bridge"
					bridge11Name          = "br11"
					bridge11MACSpoofCheck = true
				)

				Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil),
					linuxBridgeWithMACSpoofCheckNetwork,
					bridge11CNIType,
					bridge11Name,
					0,
					"",
					bridge11MACSpoofCheck)).To(Succeed())
			})

			It("Should allow outbound communication from VM under test - only if original MAC address is unchanged", func() {
				const (
					vmUnderTestIPAddress = "10.2.1.1"
					targetVMIPAddress    = "10.2.1.2"
					bridgeSubnetMask     = "/24"
				)

				initialMacAddress, err := GenerateRandomMac()
				Expect(err).NotTo(HaveOccurred())
				initialMacAddressStr := initialMacAddress.String()

				spoofedMacAddress, err := GenerateRandomMac()
				Expect(err).NotTo(HaveOccurred())
				spoofedMacAddressStr := spoofedMacAddress.String()

				linuxBridgeInterfaceWithMACSpoofCheck := libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeWithMACSpoofCheckNetwork)

				By("Creating a VM with custom MAC address on its Linux bridge CNI interface.")
				linuxBridgeInterfaceWithCustomMac := linuxBridgeInterfaceWithMACSpoofCheck
				libvmi.InterfaceWithMac(&linuxBridgeInterfaceWithCustomMac, initialMacAddressStr)

				vmiUnderTest := libvmi.NewFedora(
					libvmi.WithInterface(linuxBridgeInterfaceWithCustomMac),
					libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeWithMACSpoofCheckNetwork, linuxBridgeWithMACSpoofCheckNetwork)),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByMac(linuxBridgeInterfaceWithCustomMac.Name, linuxBridgeInterfaceWithCustomMac.MacAddress, vmUnderTestIPAddress+bridgeSubnetMask)))
				vmiUnderTest = tests.CreateVmiOnNode(vmiUnderTest, nodes.Items[0].Name)

				By("Creating a target VM with Linux bridge CNI network interface and default MAC address.")
				targetVmi := libvmi.NewFedora(
					libvmi.WithInterface(linuxBridgeInterfaceWithMACSpoofCheck),
					libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeWithMACSpoofCheckNetwork, linuxBridgeWithMACSpoofCheckNetwork)),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByDevice("eth0", targetVMIPAddress+bridgeSubnetMask)))
				targetVmi = tests.CreateVmiOnNode(targetVmi, nodes.Items[0].Name)

				vmiUnderTest = libwait.WaitUntilVMIReady(vmiUnderTest, console.LoginToFedora)
				libwait.WaitUntilVMIReady(targetVmi, console.LoginToFedora)

				Expect(libnet.PingFromVMConsole(vmiUnderTest, targetVMIPAddress)).To(Succeed(), "Ping target IP with original MAC should succeed")

				Expect(changeInterfaceMACAddress(vmiUnderTest, linuxBridgeInterfaceWithCustomMac.Name, spoofedMacAddressStr)).To(Succeed())
				Expect(libnet.PingFromVMConsole(vmiUnderTest, targetVMIPAddress)).NotTo(Succeed(), "Ping target IP with modified MAC should fail")

				Expect(changeInterfaceMACAddress(vmiUnderTest, linuxBridgeInterfaceWithCustomMac.Name, initialMacAddressStr)).To(Succeed())
				Expect(libnet.PingFromVMConsole(vmiUnderTest, targetVMIPAddress)).To(Succeed(), "Ping target IP with restored original MAC should succeed")
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
                    ip link add ep1 type veth peer name ep2
                    ip addr add %s dev ep1
                    ip addr add %s dev ep2
                    ip addr add %s dev ep1
                    ip addr add %s dev ep2
                `, ep1Cidr, ep2Cidr, ep1CidrV6, ep2CidrV6)
				agentVMI := libvmi.NewFedora(libvmi.WithCloudInitNoCloudUserData(userdata, false))

				agentVMI.Spec.Domain.Devices.Interfaces = interfaces
				agentVMI.Spec.Networks = networks
				agentVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

				By("Starting a VirtualMachineInstance")
				agentVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				libwait.WaitForSuccessfulVMIStart(agentVMI)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(agentVMI), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				getOptions := &metav1.GetOptions{}
				Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
					updatedVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Get(context.Background(), agentVMI.Name, getOptions)
					if err != nil {
						return nil
					}
					return updatedVmi.Status.Interfaces
				}, 420*time.Second, 4).Should(HaveLen(4), "Should have interfaces in vmi status")

				updatedVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Get(context.Background(), agentVMI.Name, getOptions)
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedVmi.Status.Interfaces).To(HaveLen(4))
				interfaceByIfcName := make(map[string]v1.VirtualMachineInstanceNetworkInterface)
				for _, ifc := range updatedVmi.Status.Interfaces {
					interfaceByIfcName[ifc.InterfaceName] = ifc
				}
				Expect(interfaceByIfcName["eth0"].Name).To(Equal(masqueradeIfaceName))
				Expect(interfaceByIfcName["eth0"].InterfaceName).To(Equal("eth0"))

				Expect(interfaceByIfcName["eth1"].Name).To(Equal(linuxBridgeIfaceName))
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

func changeInterfaceMACAddress(vmi *v1.VirtualMachineInstance, interfaceName string, newMACAddress string) error {
	const maxCommandTimeout = 5 * time.Second

	commands := []string{
		ipLinkSetDev + interfaceName + " down",
		ipLinkSetDev + interfaceName + " address " + newMACAddress,
		ipLinkSetDev + interfaceName + " up",
	}

	for _, cmd := range commands {
		err := console.RunCommand(vmi, cmd, maxCommandTimeout)
		if err != nil {
			return fmt.Errorf("failed to run command: %q on VMI %s, error: %v", cmd, vmi.Name, err)
		}
	}

	return nil
}

func createNetworkAttachmentDefinition(virtClient kubecli.KubevirtClient, name, namespace, nad string) error {
	return virtClient.RestClient().
		Post().
		RequestURI(fmt.Sprintf(postUrl, namespace, name)).
		Body([]byte(nad)).
		Do(context.Background()).
		Error()
}

func configInterface(vmi *v1.VirtualMachineInstance, interfaceName, interfaceAddress string, userModifierPrefix ...string) error {
	setStaticIpCmd := fmt.Sprintf("%sip addr add %s dev %s\n", strings.Join(userModifierPrefix, " "), interfaceAddress, interfaceName)
	err := runSafeCommand(vmi, setStaticIpCmd)

	if err != nil {
		return fmt.Errorf("could not configure address %s for interface %s on VMI %s: %w", interfaceAddress, interfaceName, vmi.Name, err)
	}

	return setInterfaceUp(vmi, interfaceName)
}

func checkMacAddress(vmi *v1.VirtualMachineInstance, interfaceName, macAddress string) error {
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	err := console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: macAddress},
		&expect.BSnd{S: tests.EchoLastReturnValue},
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
		&expect.BSnd{S: tests.EchoLastReturnValue},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
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

// If staticIP is empty the interface would get a dynamic IP
func configureAlpineInterfaceIP(vmi *v1.VirtualMachineInstance, ifaceName, staticIP string) error {
	if staticIP == "" {
		return activateDHCPOnVMInterfaces(vmi, ifaceName)
	}

	return configInterface(vmi, ifaceName, staticIP)
}

func activateDHCPOnVMInterfaces(vmi *v1.VirtualMachineInstance, ifacesNames ...string) error {
	interfacesConfig := "auto lo\\niface lo inet loopback\\n\\n"

	for idx := range ifacesNames {
		interfacesConfig += fmt.Sprintf("auto %s\\niface %s inet dhcp\\nhostname localhost\\n\\n",
			ifacesNames[idx],
			ifacesNames[idx])
	}

	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $'" + interfacesConfig + "' > /etc/network/interfaces\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "/etc/init.d/networking restart\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: tests.EchoLastReturnValue},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
}
