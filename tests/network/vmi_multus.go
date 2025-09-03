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
 * Copyright The KubeVirt Authors.
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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	shouldCreateNetwork = "should successfully create the network"
	ipLinkSetDev        = "ip link set dev "
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
	bridge10Name          = "br10"
	bridge10MacSpoofCheck = false
)

var _ = Describe(SIG("Multus", Serial, decorators.Multus, func() {
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

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		nodes = libnode.GetAllSchedulableNodes(virtClient)
		Expect(nodes.Items).NotTo(BeEmpty())

		const vlanID100 = 100
		Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), linuxBridgeVlan100Network,
			bridge10Name, vlanID100, nil, bridge10MacSpoofCheck)).To(Succeed())

		Expect(createPtpNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), ptpConf1, ptpSubnet)).To(Succeed())
		Expect(createPtpNetworkAttachmentDefinition(testsuite.NamespaceTestAlternative, ptpConf2, ptpSubnet)).To(Succeed())

		// Multus tests need to ensure that old VMIs are gone
		Eventually(func() []v1.VirtualMachineInstance {
			list1, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).List(context.Background(), v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			list2, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestAlternative).List(context.Background(), v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return append(list1.Items, list2.Items...)
		}, 6*time.Minute, 1*time.Second).Should(BeEmpty())
	})

	createVMIOnNode := func(interfaces []v1.Interface, networks []v1.Network) *v1.VirtualMachineInstance {
		// Arbitrarily select one compute node in the cluster, on which it is possible to create a VMI
		// (i.e. a schedulable node).
		vmi := libvmifact.NewAlpine(libvmi.WithNodeAffinityFor(nodes.Items[0].Name))
		vmi.Spec.Domain.Devices.Interfaces = interfaces
		vmi.Spec.Networks = networks
		vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return vmi
	}

	Describe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance using different types of interfaces.", func() {
		const ptpGateway = ptpSubnetIP1
		Context("VirtualMachineInstance with cni ptp plugin interface", func() {
			var networkData string
			BeforeEach(func() {
				libnet.SkipWhenClusterNotSupportIpv4()
				networkData, err = cloudinit.NewNetworkData(
					cloudinit.WithEthernet("eth0",
						cloudinit.WithDHCP4Enabled(),
						cloudinit.WithNameserverFromCluster(),
					),
				)
				Expect(err).NotTo(HaveOccurred())
			})

			It("[test_id:1752]should create a virtual machine with one interface with network definition from different namespace", func() {
				By("checking virtual machine instance can ping using ptp cni plugin")
				detachedVMI := libvmifact.NewAlpineWithTestTooling(
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
				)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: fmt.Sprintf("%s/%s", testsuite.NamespaceTestAlternative, ptpConf2)},
					}},
				}

				detachedVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), detachedVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitUntilVMIReady(detachedVMI, console.LoginToAlpine)

				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())
			})

			It("[test_id:1753]should create a virtual machine with two interfaces", func() {
				By("checking virtual machine instance can ping using ptp cni plugin")
				const secondaryNetName = "ptp"
				detachedVMI := libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, ptpConf1)),
				)

				detachedVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), detachedVMI, metav1.CreateOptions{})
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
				networkData, err := cloudinit.NewNetworkData(
					cloudinit.WithEthernet("eth0",
						cloudinit.WithDHCP4Enabled(),
						cloudinit.WithNameserverFromCluster(),
					),
				)
				Expect(err).NotTo(HaveOccurred())
				detachedVMI := libvmifact.NewAlpineWithTestTooling(
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
				)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: fmt.Sprintf("%s/%s", testsuite.GetTestNamespace(nil), ptpConf1),
							Default:     true,
						},
					}},
				}

				detachedVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), detachedVMI, metav1.CreateOptions{})
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
				output := libpod.RunCommandOnVmiPod(detachedVMI, []string{"/bin/bash", "-c", "/usr/sbin/ip link show|grep -c UP"})
				ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("4"))
			})
		})

		Context("VirtualMachineInstance with Linux bridge plugin interface", func() {
			getIfaceIPByNetworkName := func(vmiName, networkName string) (string, error) {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmiName, metav1.GetOptions{})
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

			DescribeTable("should be able to ping between two vms", func(interfaces []v1.Interface,
				networks []v1.Network, ifaceName, staticIPVm1, staticIPVm2 string) {
				if staticIPVm2 == "" || staticIPVm1 == "" {
					ipam := map[string]string{"type": "host-local", "subnet": ptpSubnet}
					Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), linuxBridgeVlan100WithIPAMNetwork, bridge10Name, 0, ipam, bridge10MacSpoofCheck)).To(Succeed())
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
				networkData, err := cloudinit.NewNetworkData(
					cloudinit.WithEthernet("eth1",
						cloudinit.WithAddresses(ptpSubnetIP2+ptpSubnetMask),
					),
				)
				Expect(err).NotTo(HaveOccurred())

				vmiTwo := libvmifact.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterface),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
					libvmi.WithNodeAffinityFor(nodes.Items[0].Name),
				)
				vmiTwo, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmiTwo)).Create(context.Background(), vmiTwo, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitUntilVMIReady(vmiTwo, console.LoginToFedora)
			})

			It("[test_id:676]should configure valid custom MAC address on Linux bridge CNI interface.", func() {
				By("Creating another VM with custom MAC address on its Linux bridge CNI interface.")
				linuxBridgeInterfaceWithCustomMac := linuxBridgeInterface
				linuxBridgeInterfaceWithCustomMac.MacAddress = customMacAddress

				networkData, err := cloudinit.NewNetworkData(
					cloudinit.WithEthernet("eth1",
						cloudinit.WithAddresses(ptpSubnetIP1+ptpSubnetMask),
						cloudinit.WithMatchingMAC(linuxBridgeInterfaceWithCustomMac.MacAddress),
					),
				)
				Expect(err).NotTo(HaveOccurred())

				vmiOne := libvmifact.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterfaceWithCustomMac),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
					libvmi.WithNodeAffinityFor(nodes.Items[0].Name),
				)

				vmiOne, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmiOne)).Create(context.Background(), vmiOne, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmiOne = libwait.WaitUntilVMIReady(vmiOne, console.LoginToFedora)
				Eventually(matcher.ThisVMI(vmiOne), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Verifying the desired custom MAC is the one that were actually configured on the interface.")
				vmiIfaceStatusByName := indexInterfaceStatusByName(vmiOne)
				Expect(vmiIfaceStatusByName).To(HaveKey(linuxBridgeInterfaceWithCustomMac.Name), "should set linux bridge interface with the custom MAC address at VMI Status")
				Expect(vmiIfaceStatusByName[linuxBridgeInterfaceWithCustomMac.Name].MAC).To(Equal(customMacAddress), "should set linux bridge interface with the custom MAC address at VMI")

				By("Verifying the desired custom MAC is not configured inside the pod namespace.")
				vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmiOne, vmiOne.Namespace)
				Expect(err).NotTo(HaveOccurred())

				podInterfaceName := "72ad293a5c9-nic"
				out, err := exec.ExecuteCommandOnPod(
					vmiPod,
					"compute",
					[]string{"sh", "-c", fmt.Sprintf("ip a show %s", podInterfaceName)},
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

				updatedVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmiOne)).Get(context.Background(), vmiOne.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedVmi.Status.Interfaces).To(HaveLen(2))
				interfacesByName := make(map[string]v1.VirtualMachineInstanceNetworkInterface)
				for _, ifc := range updatedVmi.Status.Interfaces {
					interfacesByName[ifc.Name] = ifc
				}

				for _, network := range networks {
					ifc, isPresent := interfacesByName[network.Name]
					Expect(isPresent).To(BeTrue())
					Expect(ifc.MAC).To(Not(BeZero()))
				}
				Expect(interfacesByName[masqueradeIfaceName].MAC).To(Not(Equal(interfacesByName[linuxBridgeIfaceName].MAC)))
				const timeout = time.Second * 5
				Expect(console.RunCommand(vmiOne, fmt.Sprintf("ip addr show eth0 | grep %s\n", interfacesByName["default"].MAC), timeout)).To(Succeed())
				Expect(console.RunCommand(vmiOne, fmt.Sprintf("ip addr show eth1 | grep %s\n", interfacesByName[linuxBridgeIfaceName].MAC), timeout)).To(Succeed())
			})

			It("should have the correct MTU on the secondary interface with no dhcp server", func() {
				getPodInterfaceMtu := func(vmi *v1.VirtualMachineInstance) string {
					vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).NotTo(HaveOccurred())

					output, err := exec.ExecuteCommandOnPod(
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

				vmi := libvmifact.NewFedora(
					libvmi.WithInterface(linuxBridgeInterface),
					libvmi.WithNetwork(&linuxBridgeNetwork),
				)

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
				Expect(getPodInterfaceMtu(vmi)).To(Equal(getVmiInterfaceMtu(vmi)))
			})
		})

		Context("Security", func() {
			BeforeEach(func() {
				const (
					bridge11Name          = "br11"
					bridge11MACSpoofCheck = true
				)

				Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil),
					linuxBridgeWithMACSpoofCheckNetwork,
					bridge11Name,
					0,
					nil,
					bridge11MACSpoofCheck)).To(Succeed())
			})

			It("Should allow outbound communication from VM under test - only if original MAC address is unchanged",
				func() {
					const (
						vmUnderTestIPAddress = "10.2.1.1"
						targetVMIPAddress    = "10.2.1.2"
						bridgeSubnetMask     = "/24"
					)

					initialMacAddress, err := libnet.GenerateRandomMac()
					Expect(err).NotTo(HaveOccurred())
					initialMacAddressStr := initialMacAddress.String()

					spoofedMacAddress, err := libnet.GenerateRandomMac()
					Expect(err).NotTo(HaveOccurred())
					spoofedMacAddressStr := spoofedMacAddress.String()

					linuxBridgeInterfaceWithMACSpoofCheck := libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeWithMACSpoofCheckNetwork)

					By("Creating a VM with custom MAC address on its Linux bridge CNI interface.")
					linuxBridgeInterfaceWithCustomMac := linuxBridgeInterfaceWithMACSpoofCheck
					libvmi.InterfaceWithMac(&linuxBridgeInterfaceWithCustomMac, initialMacAddressStr)

					networkData, err := cloudinit.NewNetworkData(
						cloudinit.WithEthernet(linuxBridgeInterfaceWithCustomMac.Name,
							cloudinit.WithAddresses(vmUnderTestIPAddress+bridgeSubnetMask),
							cloudinit.WithMatchingMAC(linuxBridgeInterfaceWithCustomMac.MacAddress),
						),
					)
					Expect(err).NotTo(HaveOccurred())

					vmiUnderTest := libvmifact.NewFedora(
						libvmi.WithInterface(linuxBridgeInterfaceWithCustomMac),
						libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeWithMACSpoofCheckNetwork, linuxBridgeWithMACSpoofCheckNetwork)),
						libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
						libvmi.WithNodeAffinityFor(nodes.Items[0].Name),
					)
					vmiUnderTest, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmiUnderTest)).Create(context.Background(), vmiUnderTest, metav1.CreateOptions{})
					ExpectWithOffset(1, err).ToNot(HaveOccurred())

					By("Creating a target VM with Linux bridge CNI network interface and default MAC address.")
					targetNetworkData, err := cloudinit.NewNetworkData(
						cloudinit.WithEthernet("eth0",
							cloudinit.WithAddresses(targetVMIPAddress+bridgeSubnetMask),
						),
					)
					Expect(err).NotTo(HaveOccurred())

					targetVmi := libvmifact.NewFedora(
						libvmi.WithInterface(linuxBridgeInterfaceWithMACSpoofCheck),
						libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeWithMACSpoofCheckNetwork, linuxBridgeWithMACSpoofCheckNetwork)),
						libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(targetNetworkData)),
						libvmi.WithNodeAffinityFor(nodes.Items[0].Name),
					)
					targetVmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(targetVmi)).Create(context.Background(), targetVmi, metav1.CreateOptions{})
					ExpectWithOffset(1, err).ToNot(HaveOccurred())

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
				agentVMI := libvmifact.NewFedora(libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(userdata)))

				agentVMI.Spec.Domain.Devices.Interfaces = interfaces
				agentVMI.Spec.Networks = networks
				agentVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

				By("Starting a VirtualMachineInstance")
				agentVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), agentVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				libwait.WaitForSuccessfulVMIStart(agentVMI)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(agentVMI), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				getOptions := metav1.GetOptions{}
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
}))

func changeInterfaceMACAddress(vmi *v1.VirtualMachineInstance, interfaceName, newMACAddress string) error {
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

// If staticIP is empty the interface would get a dynamic IP
func configureAlpineInterfaceIP(vmi *v1.VirtualMachineInstance, ifaceName, staticIP string) error {
	if staticIP == "" {
		return activateDHCPOnVMInterfaces(vmi, ifaceName)
	}
	if err := libnet.AddIPAddress(vmi, ifaceName, staticIP); err != nil {
		return err
	}

	return libnet.SetInterfaceUp(vmi, ifaceName)
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
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
}

func indexInterfaceStatusByName(vmi *v1.VirtualMachineInstance) map[string]v1.VirtualMachineInstanceNetworkInterface {
	interfaceStatusByName := map[string]v1.VirtualMachineInstanceNetworkInterface{}
	for _, interfaceStatus := range vmi.Status.Interfaces {
		interfaceStatusByName[interfaceStatus.Name] = interfaceStatus
	}
	return interfaceStatusByName
}

func createBridgeNetworkAttachmentDefinition(namespace, networkName string, bridgeName string, vlan int, ipam map[string]string, macSpoofCheck bool) error {
	netAttachDef := libnet.NewBridgeNetAttachDef(
		networkName,
		bridgeName,
		libnet.WithMTU(1400),
		libnet.WithVLAN(vlan),
		libnet.WithIPAM(ipam),
		libnet.WithMacSpoofChk(macSpoofCheck),
	)
	_, err := libnet.CreateNetAttachDef(context.Background(), namespace, netAttachDef)
	return err
}

func createPtpNetworkAttachmentDefinition(namespace, networkName, subnet string) error {
	const pluginType = "ptp"
	ipam := map[string]string{"type": "host-local", "subnet": subnet}
	netAttachDef := libnet.NewNetAttachDef(
		networkName,
		libnet.NewNetConfig("mynet", libnet.NewNetPluginConfig(pluginType, map[string]interface{}{"ipam": ipam})),
	)
	_, err := libnet.CreateNetAttachDef(context.Background(), namespace, netAttachDef)
	return err
}
