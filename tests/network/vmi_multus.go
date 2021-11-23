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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"regexp"
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
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/tests/framework/checks"

	"kubevirt.io/kubevirt/tests/util"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const (
	postUrl                = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
	linuxBridgeConfNAD     = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"%s\", \"bridge\": \"%s\", \"vlan\": %d, \"ipam\": {%s}, \"macspoofchk\": %t, \"mtu\": 1400},{\"type\": \"tuning\"}]}"}}`
	ptpConfNAD             = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"ptp\", \"ipam\": { \"type\": \"host-local\", \"subnet\": \"%s\" }},{\"type\": \"tuning\"}]}"}}`
	macvtapNetworkConfNAD  = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s", "annotations": {"k8s.v1.cni.cncf.io/resourceName": "macvtap.network.kubevirt.io/%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"%s\", \"type\": \"macvtap\"}"}}`
	sriovConfNAD           = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"vlan\": 0, \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	sriovLinkEnableConfNAD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"vlan\": 0, \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	sriovVlanConfNAD       = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"vlan\": 200, \"ipam\":{}}"}}`
)

const (
	sriovnet1           = "sriov"
	sriovnet2           = "sriov2"
	sriovnetLinkEnabled = "sriov-link-enabled"
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

const (
	helloWorldCloudInitData = "#!/bin/bash\necho 'hello'\n"
)

var _ = SIGDescribe("[Serial]Multus", func() {

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
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()

		nodes = util.GetAllSchedulableNodes(virtClient)
		Expect(len(nodes.Items) > 0).To(BeTrue())

		const vlanID100 = 100
		Expect(createBridgeNetworkAttachmentDefinition(util.NamespaceTestDefault, linuxBridgeVlan100Network, bridge10CNIType, bridge10Name, vlanID100, "", bridge10MacSpoofCheck)).To(Succeed())

		// Create ptp crds with tuning plugin enabled in two different namespaces
		Expect(createPtpNetworkAttachmentDefinition(util.NamespaceTestDefault, ptpConf1, ptpSubnet)).To(Succeed())
		Expect(createPtpNetworkAttachmentDefinition(tests.NamespaceTestAlternative, ptpConf2, ptpSubnet)).To(Succeed())

		// Multus tests need to ensure that old VMIs are gone
		Eventually(func() int {
			list1, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			list2, err := virtClient.VirtualMachineInstance(tests.NamespaceTestAlternative).List(&v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(list1.Items) + len(list2.Items)
		}, 6*time.Minute, 1*time.Second).Should(BeZero())
	})

	createVMIOnNode := func(interfaces []v1.Interface, networks []v1.Network) *v1.VirtualMachineInstance {
		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskAlpine), helloWorldCloudInitData)
		vmi.Spec.Domain.Devices.Interfaces = interfaces
		vmi.Spec.Networks = networks

		// Arbitrarily select one compute node in the cluster, on which it is possible to create a VMI
		// (i.e. a schedulable node).
		nodeName := nodes.Items[0].Name
		tests.CreateVmiOnNode(vmi, nodeName)

		return vmi
	}

	Describe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance using different types of interfaces.", func() {
		const ptpGateway = ptpSubnetIP1
		Context("VirtualMachineInstance with cni ptp plugin interface", func() {
			It("[test_id:1751]should create a virtual machine with one interface", func() {
				By("checking virtual machine instance can ping using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), helloWorldCloudInitData)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: ptpConf1},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, libnet.WithIPv6(console.LoginToCirros))

				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())
			})

			It("[test_id:1752]should create a virtual machine with one interface with network definition from different namespace", func() {
				tests.SkipIfOpenShift4("OpenShift 4 does not support usage of the network definition from the different namespace")
				By("checking virtual machine instance can ping using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), helloWorldCloudInitData)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: fmt.Sprintf("%s/%s", tests.NamespaceTestAlternative, ptpConf2)},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, libnet.WithIPv6(console.LoginToCirros))

				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())
			})

			It("[test_id:1753]should create a virtual machine with two interfaces", func() {
				By("checking virtual machine instance can ping using ptp cni plugin")
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), helloWorldCloudInitData)

				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					defaultNetwork,
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: ptpConf1},
					}},
				}

				_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(detachedVMI)
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

				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())
			})
		})

		Context("VirtualMachineInstance with multus network as default network", func() {
			It("[test_id:1751]should create a virtual machine with one interface with multus default network definition", func() {
				detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), helloWorldCloudInitData)
				detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				detachedVMI.Spec.Networks = []v1.Network{
					{Name: "ptp", NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: fmt.Sprintf("%s/%s", util.NamespaceTestDefault, ptpConf1),
							Default:     true,
						}}},
				}

				_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(detachedVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(detachedVMI, libnet.WithIPv6(console.LoginToCirros))

				By("checking virtual machine instance can ping using ptp cni plugin")
				Expect(libnet.PingFromVMConsole(detachedVMI, ptpGateway)).To(Succeed())

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
				tests.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)

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
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmiOne, util.NamespaceTestDefault)
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
			getIfaceIPByNetworkName := func(vmiName, networkName string) (string, error) {
				vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmiName, &metav1.GetOptions{})
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

			table.DescribeTable("should be able to ping between two vms", func(interfaces []v1.Interface, networks []v1.Network, ifaceName, staticIPVm1, staticIPVm2 string) {
				if staticIPVm2 == "" || staticIPVm1 == "" {
					ipam := generateIPAMConfig("host-local", ptpSubnet)
					Expect(createBridgeNetworkAttachmentDefinition(util.NamespaceTestDefault, linuxBridgeVlan100WithIPAMNetwork, bridge10CNIType, bridge10Name, 0, ipam, bridge10MacSpoofCheck)).To(Succeed())
				}

				vmiOne := createVMIOnNode(interfaces, networks)
				vmiTwo := createVMIOnNode(interfaces, networks)

				tests.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)
				tests.WaitUntilVMIReady(vmiTwo, console.LoginToAlpine)

				Expect(configureAlpineInterfaceIP(vmiOne, ifaceName, staticIPVm1)).To(Succeed())
				By(fmt.Sprintf("checking virtual machine interface %s state", ifaceName))
				Expect(checkInterface(vmiOne, ifaceName)).To(Succeed())

				Expect(configureAlpineInterfaceIP(vmiTwo, ifaceName, staticIPVm2)).To(Succeed())
				By(fmt.Sprintf("checking virtual machine interface %s state", ifaceName))
				Expect(checkInterface(vmiTwo, ifaceName)).To(Succeed())

				ipAddr := ""
				if staticIPVm2 != "" {
					ipAddr, err = cidrToIP(staticIPVm2)
				} else {
					const secondaryNetworkIndex = 1
					ipAddr, err = getIfaceIPByNetworkName(vmiTwo.Name, networks[secondaryNetworkIndex].Name)
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(ipAddr).ToNot(BeEmpty())

				By("ping between virtual machines")
				Expect(libnet.PingFromVMConsole(vmiOne, ipAddr)).To(Succeed())
			},
				table.Entry("[test_id:1577]with secondary network only", []v1.Interface{linuxBridgeInterface}, []v1.Network{linuxBridgeNetwork}, "eth0", ptpSubnetIP1+ptpSubnetMask, ptpSubnetIP2+ptpSubnetMask),
				table.Entry("[test_id:1578]with default network and secondary network", []v1.Interface{defaultInterface, linuxBridgeInterface}, []v1.Network{defaultNetwork, linuxBridgeNetwork}, "eth1", ptpSubnetIP1+ptpSubnetMask, ptpSubnetIP2+ptpSubnetMask),
				table.Entry("with default network and secondary network with IPAM", []v1.Interface{defaultInterface, linuxBridgeInterfaceWithIPAM}, []v1.Network{defaultNetwork, linuxBridgeWithIPAMNetwork}, "eth1", "", ""),
			)
		})

		Context("VirtualMachineInstance with Linux bridge CNI plugin interface and custom MAC address.", func() {
			customMacAddress := "50:00:00:00:90:0d"
			It("[test_id:676]should configure valid custom MAC address on Linux bridge CNI interface.", func() {
				By("Creating a VM with Linux bridge CNI network interface and default MAC address.")
				vmiTwo := libvmi.NewTestToolingFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterface),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByDevice("eth1", ptpSubnetIP2+ptpSubnetMask), false))
				vmiTwo = tests.CreateVmiOnNode(vmiTwo, nodes.Items[0].Name)

				By("Creating another VM with custom MAC address on its Linux bridge CNI interface.")
				linuxBridgeInterfaceWithCustomMac := linuxBridgeInterface
				linuxBridgeInterfaceWithCustomMac.MacAddress = customMacAddress
				vmiOne := libvmi.NewTestToolingFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(linuxBridgeInterfaceWithCustomMac),
					libvmi.WithNetwork(&linuxBridgeNetwork),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByMac(linuxBridgeInterfaceWithCustomMac.Name, customMacAddress, ptpSubnetIP1+ptpSubnetMask), false))
				vmiOne = tests.CreateVmiOnNode(vmiOne, nodes.Items[0].Name)

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

				tests.WaitUntilVMIReady(vmiOne, console.LoginToAlpine)

				updatedVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmiOne.Name, &metav1.GetOptions{})
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
				Expect(interfacesByName[masqueradeIfaceName].MAC).To(Not(Equal(interfacesByName[linuxBridgeIfaceName].MAC)))
				Expect(runSafeCommand(vmiOne, fmt.Sprintf("ip addr show eth0 | grep %s\n", interfacesByName["default"].MAC))).To(Succeed())
				Expect(runSafeCommand(vmiOne, fmt.Sprintf("ip addr show eth1 | grep %s\n", interfacesByName[linuxBridgeIfaceName].MAC))).To(Succeed())
			})

			It("should have the correct MTU on the secondary interface with no dhcp server", func() {
				getPodInterfaceMtu := func(vmi *v1.VirtualMachineInstance) string {
					vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
					output, err := tests.ExecuteCommandOnPod(
						virtClient,
						vmiPod,
						"compute",
						[]string{"cat", "/sys/class/net/net1/mtu"},
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

				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				vmi = tests.WaitUntilVMIReady(vmi, console.LoginToFedora)
				Expect(getPodInterfaceMtu(vmi)).To(Equal(getVmiInterfaceMtu(vmi)))
			})
		})

		Context("VirtualMachineInstance with invalid MAC address", func() {

			It("[test_id:1713]should failed to start with invalid MAC address", func() {
				By("Start VMI")
				linuxBridgeIfIdx := 1

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskAlpine), helloWorldCloudInitData)
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					defaultInterface,
					linuxBridgeInterface,
				}
				vmi.Spec.Domain.Devices.Interfaces[linuxBridgeIfIdx].MacAddress = "de:00c:00c:00:00:de:abc"

				vmi.Spec.Networks = []v1.Network{
					defaultNetwork,
					linuxBridgeNetwork,
				}

				_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
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

				Expect(createBridgeNetworkAttachmentDefinition(util.NamespaceTestDefault,
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

				initialMacAddress, err := tests.GenerateRandomMac()
				Expect(err).NotTo(HaveOccurred())
				initialMacAddressStr := initialMacAddress.String()

				spoofedMacAddress, err := tests.GenerateRandomMac()
				Expect(err).NotTo(HaveOccurred())
				spoofedMacAddressStr := spoofedMacAddress.String()

				linuxBridgeInterfaceWithMACSpoofCheck := libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeWithMACSpoofCheckNetwork)

				By("Creating a VM with custom MAC address on its Linux bridge CNI interface.")
				linuxBridgeInterfaceWithCustomMac := linuxBridgeInterfaceWithMACSpoofCheck
				libvmi.InterfaceWithMac(&linuxBridgeInterfaceWithCustomMac, initialMacAddressStr)

				vmiUnderTest := libvmi.NewTestToolingFedora(
					libvmi.WithInterface(linuxBridgeInterfaceWithCustomMac),
					libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeWithMACSpoofCheckNetwork)),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByMac(linuxBridgeInterfaceWithCustomMac.Name, linuxBridgeInterfaceWithCustomMac.MacAddress, vmUnderTestIPAddress+bridgeSubnetMask), false))
				vmiUnderTest = tests.CreateVmiOnNode(vmiUnderTest, nodes.Items[0].Name)

				By("Creating a target VM with Linux bridge CNI network interface and default MAC address.")
				targetVmi := libvmi.NewTestToolingFedora(
					libvmi.WithInterface(linuxBridgeInterfaceWithMACSpoofCheck),
					libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeWithMACSpoofCheckNetwork)),
					libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByDevice("eth0", targetVMIPAddress+bridgeSubnetMask), false))
				targetVmi = tests.CreateVmiOnNode(targetVmi, nodes.Items[0].Name)

				vmiUnderTest = tests.WaitUntilVMIReady(vmiUnderTest, console.LoginToFedora)
				tests.WaitUntilVMIReady(targetVmi, console.LoginToFedora)

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
				agentVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling), userdata)

				agentVMI.Spec.Domain.Devices.Interfaces = interfaces
				agentVMI.Spec.Networks = networks
				agentVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

				By("Starting a VirtualMachineInstance")
				agentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				tests.WaitForSuccessfulVMIStart(agentVMI)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, agentVMI)

				getOptions := &metav1.GetOptions{}
				Eventually(func() bool {
					updatedVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(agentVMI.Name, getOptions)
					if err != nil {
						return false
					}
					return len(updatedVmi.Status.Interfaces) == 4
				}, 420*time.Second, 4).Should(BeTrue(), "Should have interfaces in vmi status")

				updatedVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(agentVMI.Name, getOptions)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(updatedVmi.Status.Interfaces)).To(Equal(4))
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

var _ = Describe("[Serial]SRIOV", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	sriovResourceName := os.Getenv("SRIOV_RESOURCE_NAME")

	if sriovResourceName == "" {
		sriovResourceName = "kubevirt.io/sriov_net"
	}

	createSriovNetworkAttachmentDefinition := func(networkName string, namespace string, networkAttachmentDefinition string) error {
		sriovNad := fmt.Sprintf(networkAttachmentDefinition, networkName, namespace, sriovResourceName)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, sriovNad)
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.SkipIfNonRoot(virtClient, "SRIOV")

		// Check if the hardware supports SRIOV
		if err := validateSRIOVSetup(virtClient, sriovResourceName, 1); err != nil {
			Skip("Sriov is not enabled in this environment. Skip these tests using - export FUNC_TEST_ARGS='--ginkgo.skip=SRIOV'")
		}
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("VirtualMachineInstance with sriov plugin interface", func() {

		getSriovVmi := func(networks []string, cloudInitNetworkData string) *v1.VirtualMachineInstance {
			withVmiOptions := []libvmi.Option{
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			}
			if cloudInitNetworkData != "" {
				cloudinitOption := libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkData, false)
				withVmiOptions = append(withVmiOptions, cloudinitOption)
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
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			return vmi
		}

		waitVmi := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			// Need to wait for cloud init to finish and start the agent inside the vmi.
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Running multi sriov jobs with Kind, DinD is resource extensive, causing DeadlineExceeded transient warning
			// Kubevirt re-enqueue the request once it happens, so its safe to ignore this warning.
			// see https://github.com/kubevirt/kubevirt/issues/5027
			warningsIgnoreList := []string{"unknown error encountered sending command SyncVMI: rpc error: code = DeadlineExceeded desc = context deadline exceeded"}
			tests.WaitUntilVMIReadyIgnoreSelectedWarnings(vmi, console.LoginToFedora, warningsIgnoreList)
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

		// createSriovVMs instantiates two VMs on the same node connected through SR-IOV.
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

			// schedule both VM's on the same node to prevent test from being affected by how the SR-IOV card port's are connected
			sriovNodes := getNodesWithAllocatedResource(virtClient, sriovResourceName)
			Expect(sriovNodes).ToNot(BeEmpty())
			sriovNode := sriovNodes[0].Name
			vmi1 = tests.CreateVmiOnNode(vmi1, sriovNode)
			vmi2 = tests.CreateVmiOnNode(vmi2, sriovNode)

			vmi1 = waitVmi(vmi1)
			vmi2 = waitVmi(vmi2)

			vmi1, err = virtClient.VirtualMachineInstance(vmi1.Namespace).Get(vmi1.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi2, err = virtClient.VirtualMachineInstance(vmi2.Namespace).Get(vmi2.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			return vmi1, vmi2
		}

		Context("Connected to single SRIOV network", func() {
			BeforeEach(func() {
				Expect(createSriovNetworkAttachmentDefinition(sriovnet1, util.NamespaceTestDefault, sriovConfNAD)).
					To(Succeed(), "should successfully create the network")
			})

			It("should block migration for SR-IOV VMI's when LiveMigration feature-gate is on but SRIOVLiveMigration is off", func() {
				tests.EnableFeatureGate(virtconfig.LiveMigrationGate)
				defer tests.UpdateKubeVirtConfigValueAndWait(tests.KubeVirtDefaultConfig)

				vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				vmim := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				Eventually(func() error {
					_, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Create(vmim)
					return err
				}, 1*time.Minute, 20*time.Second).ShouldNot(Succeed())
			})

			It("should have cloud-init meta_data with tagged sriov nics", func() {
				noCloudInitNetworkData := ""
				vmi := getSriovVmi([]string{sriovnet1}, noCloudInitNetworkData)

				tests.AddCloudInitConfigDriveData(vmi, "disk1", "", defaultCloudInitNetworkData(), false)

				for idx, iface := range vmi.Spec.Domain.Devices.Interfaces {
					if iface.Name == sriovnet1 {
						iface.Tag = "specialNet"
						vmi.Spec.Domain.Devices.Interfaces[idx] = iface
					}
				}
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())

				domSpec := &api.DomainSpec{}
				Expect(xml.Unmarshal([]byte(domXml), domSpec)).To(Succeed())
				nic := domSpec.Devices.HostDevices[0]
				// find the SRIOV interface
				for _, iface := range domSpec.Devices.HostDevices {
					if iface.Alias.GetName() == sriovnet1 {
						nic = iface
					}
				}
				address := nic.Address
				pciAddrStr := fmt.Sprintf("%s:%s:%s:%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
				deviceData := []cloudinit.DeviceData{
					{
						Type:    cloudinit.NICMetadataType,
						Bus:     nic.Address.Type,
						Address: pciAddrStr,
						Tags:    []string{"specialNet"},
					},
				}
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				metadataStruct := cloudinit.ConfigDriveMetadata{
					InstanceID: fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
					Hostname:   dns.SanitizeHostname(vmi),
					UUID:       string(vmi.UID),
					Devices:    &deviceData,
				}

				buf, err := json.Marshal(metadataStruct)
				Expect(err).To(BeNil())
				By("mouting cloudinit iso")
				mountCloudInitConfigDrive := tests.MountCloudInitFunc("config-2")
				mountCloudInitConfigDrive(vmi)

				By("checking cloudinit meta-data")
				tests.CheckCloudInitMetaData(vmi, "openstack/latest/meta_data.json", string(buf))
			})

			It("[test_id:1754]should create a virtual machine with sriov interface", func() {
				vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
				Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

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
				Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

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
				checks.SkipTestIfNoCPUManager()
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
				Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

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

			Context("migration", func() {

				BeforeEach(func() {
					if err := validateSRIOVSetup(virtClient, sriovResourceName, 2); err != nil {
						Skip("Migration tests require at least 2 nodes: " + err.Error())
					}
				})

				BeforeEach(func() {
					tests.EnableFeatureGate(virtconfig.SRIOVLiveMigrationGate)
				})

				AfterEach(func() {
					tests.DisableFeatureGate(virtconfig.SRIOVLiveMigrationGate)
				})

				var vmi *v1.VirtualMachineInstance

				const mac = "de:ad:00:00:be:ef"

				BeforeEach(func() {
					// The SR-IOV VF MAC should be preserved on migration, therefore explicitly specify it.
					vmi = getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
					vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

					vmi = startVmi(vmi)
					vmi = waitVmi(vmi)

					var interfaceName string

					// It may take some time for the VMI interface status to be updated with the information reported by
					// the guest-agent.
					Eventually(func() error {
						var err error
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						interfaceName, err = getInterfaceNameByMAC(vmi, mac)
						return err
					}, 30*time.Second, 5*time.Second).Should(Succeed())

					Expect(checkMacAddress(vmi, interfaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest")
				})

				It("should be successful with a running VMI on the target", func() {
					By("starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)
					tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

					// It may take some time for the VMI interface status to be updated with the information reported by
					// the guest-agent.
					Eventually(func() error {
						updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						interfaceName, err := getInterfaceNameByMAC(updatedVMI, mac)
						if err != nil {
							return err
						}
						return checkMacAddress(updatedVMI, interfaceName, mac)
					}, 30*time.Second, 5*time.Second).Should(Succeed(),
						"SR-IOV VF is expected to exist in the guest after migration")
				})
			})
		})

		Context("Connected to two SRIOV networks", func() {
			BeforeEach(func() {
				Expect(createSriovNetworkAttachmentDefinition(sriovnet1, util.NamespaceTestDefault, sriovConfNAD)).To(Succeed(), "should successfully create the network")
				Expect(createSriovNetworkAttachmentDefinition(sriovnet2, util.NamespaceTestDefault, sriovConfNAD)).To(Succeed(), "should successfully create the network")
			})

			It("[test_id:1755]should create a virtual machine with two sriov interfaces referring the same resource", func() {
				sriovNetworks := []string{sriovnet1, sriovnet2}
				vmi := getSriovVmi(sriovNetworks, defaultCloudInitNetworkData())
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variables are defined in pod")
				for _, name := range sriovNetworks {
					Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, name, sriovResourceName)).To(Succeed())
				}

				checkDefaultInterfaceInPod(vmi)

				By("checking virtual machine instance has three interfaces")
				checkInterfacesInGuest(vmi, []string{"eth0", "eth1", "eth2"})

				// there is little we can do beyond just checking three devices are present: PCI slots are different inside
				// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
				// it's hard to match them.
			})
		})

		Context("Connected to link-enabled SRIOV network", func() {
			BeforeEach(func() {
				Expect(createSriovNetworkAttachmentDefinition(sriovnetLinkEnabled, util.NamespaceTestDefault, sriovLinkEnableConfNAD)).
					To(Succeed(), "should successfully create the network")
			})

			It("[test_id:3956]should connect to another machine with sriov interface over IPv4", func() {
				cidrA := "192.168.1.1/24"
				cidrB := "192.168.1.2/24"
				ipA, err := cidrToIP(cidrA)
				Expect(err).ToNot(HaveOccurred())
				ipB, err := cidrToIP(cidrB)
				Expect(err).ToNot(HaveOccurred())

				//create two vms on the same sriov network
				vmi1, vmi2 := createSriovVMs(sriovnetLinkEnabled, sriovnetLinkEnabled, cidrA, cidrB)

				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi1, ipB)
				}, 15*time.Second, time.Second).Should(Succeed())
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi2, ipA)
				}, 15*time.Second, time.Second).Should(Succeed())
			})

			It("[test_id:3957]should connect to another machine with sriov interface over IPv6", func() {
				vmi1CIDR := "fc00::1/64"
				vmi2CIDR := "fc00::2/64"
				vmi1IP, err := cidrToIP(vmi1CIDR)
				Expect(err).ToNot(HaveOccurred())
				vmi2IP, err := cidrToIP(vmi2CIDR)
				Expect(err).ToNot(HaveOccurred())

				//create two vms on the same sriov network
				vmi1, vmi2 := createSriovVMs(sriovnetLinkEnabled, sriovnetLinkEnabled, vmi1CIDR, vmi2CIDR)

				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi1, vmi2IP)
				}, 15*time.Second, time.Second).Should(Succeed())
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi2, vmi1IP)
				}, 15*time.Second, time.Second).Should(Succeed())
			})

			Context("With VLAN", func() {
				const (
					cidrVlaned1     = "192.168.0.1/24"
					sriovnetVlanned = "sriov-vlan"
				)
				var ipVlaned1 string

				BeforeEach(func() {
					var err error
					ipVlaned1, err = cidrToIP(cidrVlaned1)
					Expect(err).ToNot(HaveOccurred())
					Expect(createSriovNetworkAttachmentDefinition(sriovnetVlanned, util.NamespaceTestDefault, sriovVlanConfNAD)).To(Succeed())
				})

				It("should be able to ping between two VMIs with the same VLAN over SRIOV network", func() {
					_, vlanedVMI2 := createSriovVMs(sriovnetVlanned, sriovnetVlanned, cidrVlaned1, "192.168.0.2/24")

					By("pinging from vlanedVMI2 and the anonymous vmi over vlan")
					Eventually(func() error {
						return libnet.PingFromVMConsole(vlanedVMI2, ipVlaned1)
					}, 15*time.Second, time.Second).ShouldNot(HaveOccurred())
				})

				It("should NOT be able to ping between Vlaned VMI and a non Vlaned VMI", func() {
					_, nonVlanedVMI := createSriovVMs(sriovnetVlanned, sriovnetLinkEnabled, cidrVlaned1, "192.168.0.3/24")

					By("pinging between nonVlanedVMIand the anonymous vmi")
					Eventually(func() error {
						return libnet.PingFromVMConsole(nonVlanedVMI, ipVlaned1)
					}, 15*time.Second, time.Second).Should(HaveOccurred())
				})
			})
		})
	})
})

var _ = SIGDescribe("Macvtap", func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var macvtapLowerDevice string
	var macvtapNetworkName string

	createMacvtapNetworkAttachmentDefinition := func(namespace, networkName, macvtapLowerDevice string) error {
		macvtapNad := fmt.Sprintf(macvtapNetworkConfNAD, networkName, namespace, macvtapLowerDevice, networkName)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, macvtapNad)
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		macvtapLowerDevice = "eth0"
		macvtapNetworkName = "net1"

		// cleanup the environment
		tests.BeforeTestCleanup()
	})

	BeforeEach(func() {
		Expect(createMacvtapNetworkAttachmentDefinition(util.NamespaceTestDefault, macvtapNetworkName, macvtapLowerDevice)).
			To(Succeed(), "A macvtap network named %s should be provisioned", macvtapNetworkName)
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
		return libvmi.NewTestToolingFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithInterface(
				*libvmi.InterfaceWithMac(
					v1.DefaultMacvtapNetworkInterface(macvtapNetworkName), mac)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName)))
	}

	createCirrosVMIStaticIPOnNode := func(nodeName string, networkName string, ifaceName string, ipCIDR string, mac *string) *v1.VirtualMachineInstance {
		var vmi *v1.VirtualMachineInstance
		if mac != nil {
			vmi = newCirrosVMIWithExplicitMac(networkName, *mac)
		} else {
			vmi = newCirrosVMIWithMacvtapNetwork(networkName)
		}
		vmi = tests.WaitUntilVMIReady(
			tests.CreateVmiOnNode(vmi, nodeName),
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
		var nodeList *k8sv1.NodeList
		var nodeName string
		var serverIP string

		BeforeEach(func() {
			nodeList = util.GetAllSchedulableNodes(virtClient)
			Expect(nodeList.Items).NotTo(BeEmpty(), "schedulable kubernetes nodes must be present")
			nodeName = nodeList.Items[0].Name
			chosenMACHW, err := tests.GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			chosenMAC = chosenMACHW.String()
			serverCIDR := "192.0.2.102/24"

			serverIP, err = cidrToIP(serverCIDR)
			Expect(err).ToNot(HaveOccurred())

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
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())
			})
		})
	})

	Context("VMI migration", func() {
		var clientVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			tests.SkipIfMigrationIsNotPossible()
		})

		BeforeEach(func() {
			macAddressHW, err := tests.GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			macAddress := macAddressHW.String()
			clientVMI, err = createCirrosVMIRandomNode(macvtapNetworkName, macAddress)
			Expect(err).NotTo(HaveOccurred(), "must succeed creating a VMI on a random node")
		})

		It("should be successful when the VMI MAC address is defined in its spec", func() {
			By("starting the migration")
			migration := tests.NewRandomMigration(clientVMI.Name, clientVMI.Namespace)
			migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			// check VMI, confirm migration state
			tests.ConfirmVMIPostMigration(virtClient, clientVMI, migrationUID)
		})

		Context("with live traffic", func() {
			var serverVMI *v1.VirtualMachineInstance
			var serverVMIPodName string
			var serverIP string

			macvtapIfaceIPReportTimeout := 4 * time.Minute

			waitVMMacvtapIfaceIPReport := func(vmi *v1.VirtualMachineInstance, macAddress string, timeout time.Duration) (string, error) {
				var vmiIP string
				err := wait.PollImmediate(time.Second, timeout, func() (done bool, err error) {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
					if err != nil {
						return false, err
					}

					for _, iface := range vmi.Status.Interfaces {
						if iface.MAC == macAddress {
							if ip := iface.IP; ip != "" {
								vmiIP = ip
								return true, nil
							}
							return false, nil
						}
					}

					return false, nil
				})
				if err != nil {
					return "", err
				}

				return vmiIP, nil
			}

			waitForPodCompleted := func(podNamespace string, podName string) error {
				pod, err := virtClient.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
					return nil
				}
				return fmt.Errorf("pod hasn't completed, current Phase: %s", pod.Status.Phase)
			}

			BeforeEach(func() {
				macAddressHW, err := tests.GenerateRandomMac()
				Expect(err).ToNot(HaveOccurred())
				macAddress := macAddressHW.String()

				serverVMI, err = createFedoraVMIRandomNode(macvtapNetworkName, macAddress)
				Expect(err).NotTo(HaveOccurred(), "must have succeeded creating a fedora VMI on a random node")
				Expect(serverVMI.Status.Interfaces).NotTo(BeEmpty(), "a migrate-able VMI must have network interfaces")
				serverVMIPodName = tests.GetVmPodName(virtClient, serverVMI)

				serverIP, err = waitVMMacvtapIfaceIPReport(serverVMI, macAddress, macvtapIfaceIPReportTimeout)
				Expect(err).NotTo(HaveOccurred(), "should have managed to figure out the IP of the server VMI")
			})

			BeforeEach(func() {
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *before* migrating the VMI")
			})

			It("should keep connectivity after a migration", func() {
				migration := tests.NewRandomMigration(serverVMI.Name, serverVMI.GetNamespace())
				_ = tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)
				// In case of clientVMI and serverVMI running on the same node before migration, the serverVMI
				// will be reachable only when the original launcher pod terminates.
				Eventually(func() error {
					return waitForPodCompleted(serverVMI.Namespace, serverVMIPodName)
				}, tests.ContainerCompletionWaitTime, time.Second).Should(Succeed(), fmt.Sprintf("all containers should complete in source virt-launcher pod: %s", serverVMIPodName))
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *after* migrating the VMI")
			})
		})
	})
})

func changeInterfaceMACAddress(vmi *v1.VirtualMachineInstance, interfaceName string, newMACAddress string) error {
	const maxCommandTimeout = 5 * time.Second

	commands := []string{
		"ip link set dev " + interfaceName + " down",
		"ip link set dev " + interfaceName + " address " + newMACAddress,
		"ip link set dev " + interfaceName + " up",
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

func cidrToIP(cidr string) (string, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	return ip.String(), nil
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

func validateSRIOVSetup(virtClient kubecli.KubevirtClient, sriovResourceName string, minRequiredNodes int) error {
	sriovNodes := getNodesWithAllocatedResource(virtClient, sriovResourceName)
	if len(sriovNodes) < minRequiredNodes {
		return fmt.Errorf("not enough compute nodes with SR-IOV support detected")
	}
	return nil
}

func getNodesWithAllocatedResource(virtClient kubecli.KubevirtClient, resourceName string) []k8sv1.Node {
	nodes := util.GetAllSchedulableNodes(virtClient)
	filteredNodes := []k8sv1.Node{}
	for _, node := range nodes.Items {
		resourceList := node.Status.Allocatable
		for k, v := range resourceList {
			if string(k) == resourceName {
				if v.Value() > 0 {
					filteredNodes = append(filteredNodes, node)
					break
				}
			}
		}
	}

	return filteredNodes
}

func validatePodKubevirtResourceNameByVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, networkName, sriovResourceName string) error {
	out, err := tests.ExecuteCommandOnPod(
		virtClient,
		tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace),
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
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
}
