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
	"os"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	sriovLinkEnableConfNAD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"vlan\": 0, \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	sriovVlanConfNAD       = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"vlan\": 200, \"ipam\":{}}"}}`
	sriovConfNAD           = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"vlan\": 0, \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
)

const (
	sriovnet1           = "sriov"
	sriovnet2           = "sriov2"
	sriovnet3           = "sriov3"
	sriovnet4           = "sriov4"
	sriovnetLinkEnabled = "sriov-linked"
)

var _ = Describe("[Serial]SRIOV", Serial, decorators.SRIOV, func() {

	var virtClient kubecli.KubevirtClient

	sriovResourceName := readSRIOVResourceName()

	createSriovNetworkAttachmentDefinition := func(networkName string, namespace string, networkAttachmentDefinition string) error {
		sriovNad := fmt.Sprintf(networkAttachmentDefinition, networkName, namespace, sriovResourceName)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, sriovNad)
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		// Check if the hardware supports SRIOV
		if err := validateSRIOVSetup(virtClient, sriovResourceName, 1); err != nil {
			Skip("Sriov is not enabled in this environment. Skip these tests using - export FUNC_TEST_ARGS='--skip=SRIOV'")
		}
	})

	Context("VMI connected to single SRIOV network", func() {
		BeforeEach(func() {
			Expect(createSriovNetworkAttachmentDefinition(sriovnet1, util.NamespaceTestDefault, sriovConfNAD)).
				To(Succeed(), shouldCreateNetwork)
		})

		It("should have cloud-init meta_data with aligned cpus to sriov interface numa node for VMIs with dedicatedCPUs", func() {
			checks.SkipTestIfNoCPUManager()
			noCloudInitNetworkData := ""
			vmi := newSRIOVVmi([]string{sriovnet1}, noCloudInitNetworkData)
			tests.AddCloudInitConfigDriveData(vmi, "disk1", "", defaultCloudInitNetworkData(), false)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 4,
				DedicatedCPUPlacement: true,
			}

			for idx, iface := range vmi.Spec.Domain.Devices.Interfaces {
				if iface.Name == sriovnet1 {
					iface.Tag = "specialNet"
					vmi.Spec.Domain.Devices.Interfaces[idx] = iface
				}
			}
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

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
			pciAddrStr := fmt.Sprintf("%s:%s:%s.%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
			srcAddr := nic.Source.Address
			sourcePCIAddress := fmt.Sprintf("%s:%s:%s.%s", srcAddr.Domain[2:], srcAddr.Bus[2:], srcAddr.Slot[2:], srcAddr.Function[2:])
			alignedCPUsInt, err := hardware.LookupDeviceVCPUAffinity(sourcePCIAddress, domSpec)
			Expect(err).ToNot(HaveOccurred())
			deviceData := []cloudinit.DeviceData{
				{
					Type:        cloudinit.NICMetadataType,
					Bus:         nic.Address.Type,
					Address:     pciAddrStr,
					Tags:        []string{"specialNet"},
					AlignedCPUs: alignedCPUsInt,
				},
			}
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			metadataStruct := cloudinit.ConfigDriveMetadata{
				InstanceID: fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
				Hostname:   dns.SanitizeHostname(vmi),
				UUID:       string(vmi.Spec.Domain.Firmware.UUID),
				Devices:    &deviceData,
			}

			buf, err := json.Marshal(metadataStruct)
			Expect(err).ToNot(HaveOccurred())
			By("mouting cloudinit iso")
			mountCloudInitConfigDrive := tests.MountCloudInitFunc("config-2")
			mountCloudInitConfigDrive(vmi)

			By("checking cloudinit meta-data")
			tests.CheckCloudInitMetaData(vmi, "openstack/latest/meta_data.json", string(buf))

		})

		It("should have cloud-init meta_data with tagged sriov nics", func() {
			noCloudInitNetworkData := ""
			vmi := newSRIOVVmi([]string{sriovnet1}, noCloudInitNetworkData)
			testInstancetype := "testInstancetype"
			if vmi.Annotations == nil {
				vmi.Annotations = make(map[string]string)
			}
			vmi.Annotations[v1.InstancetypeAnnotation] = testInstancetype
			tests.AddCloudInitConfigDriveData(vmi, "disk1", "", defaultCloudInitNetworkData(), false)

			for idx, iface := range vmi.Spec.Domain.Devices.Interfaces {
				if iface.Name == sriovnet1 {
					iface.Tag = "specialNet"
					vmi.Spec.Domain.Devices.Interfaces[idx] = iface
				}
			}
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

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
			pciAddrStr := fmt.Sprintf("%s:%s:%s.%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
			deviceData := []cloudinit.DeviceData{
				{
					Type:    cloudinit.NICMetadataType,
					Bus:     nic.Address.Type,
					Address: pciAddrStr,
					Tags:    []string{"specialNet"},
				},
			}
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			metadataStruct := cloudinit.ConfigDriveMetadata{
				InstanceID:   fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
				InstanceType: testInstancetype,
				Hostname:     dns.SanitizeHostname(vmi),
				UUID:         string(vmi.Spec.Domain.Firmware.UUID),
				Devices:      &deviceData,
			}

			buf, err := json.Marshal(metadataStruct)
			Expect(err).ToNot(HaveOccurred())
			By("mouting cloudinit iso")
			mountCloudInitConfigDrive := tests.MountCloudInitFunc("config-2")
			mountCloudInitConfigDrive(vmi)

			By("checking cloudinit meta-data")
			tests.CheckCloudInitMetaData(vmi, "openstack/latest/meta_data.json", string(buf))
		})

		It("[test_id:1754]should create a virtual machine with sriov interface", func() {
			vmi := newSRIOVVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})

			// there is little we can do beyond just checking two devices are present: PCI slots are different inside
			// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
			// it's hard to match them.
		})

		It("[test_id:1754]should create a virtual machine with sriov interface with all pci devices on the root bus", func() {
			vmi := newSRIOVVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi.Annotations = map[string]string{
				v1.PlacePCIDevicesOnRootComplex: "true",
			}
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

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
			Expect(rootPortController).To(BeEmpty(), "libvirt should not add additional buses to the root one")
		})

		It("[test_id:3959]should create a virtual machine with sriov interface and dedicatedCPUs", func() {
			checks.SkipTestIfNoCPUManager()
			// In addition to verifying that we can start a VMI with CPU pinning
			// this also tests if we've correctly calculated the overhead for VFIO devices.
			vmi := newSRIOVVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				DedicatedCPUPlacement: true,
			}
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
			Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})
		})
		It("[test_id:3985]should create a virtual machine with sriov interface with custom MAC address", func() {
			const mac = "de:ad:00:00:be:ef"
			vmi := newSRIOVVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
			vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			By("checking virtual machine instance has an interface with the requested MAC address")
			ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 140*time.Second)
			Expect(err).NotTo(HaveOccurred())
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(checkMacAddress(vmi, ifaceName, mac)).To(Succeed())

			By("checking virtual machine instance reports the expected network name")
			Expect(getInterfaceNetworkNameByMAC(vmi, mac)).To(Equal(sriovnet1))
			By("checking virtual machine instance reports the expected info source")
			networkInterface := vmispec.LookupInterfaceStatusByMac(vmi.Status.Interfaces, mac)
			Expect(networkInterface).NotTo(BeNil(), "interface not found")
			Expect(networkInterface.InfoSource).To(Equal(vmispec.NewInfoSource(
				vmispec.InfoSourceDomain, vmispec.InfoSourceGuestAgent, vmispec.InfoSourceMultusStatus)))
		})

		Context("migration", func() {

			BeforeEach(func() {
				if err := validateSRIOVSetup(virtClient, sriovResourceName, 2); err != nil {
					Skip("Migration tests require at least 2 nodes: " + err.Error())
				}
			})

			var vmi *v1.VirtualMachineInstance

			const mac = "de:ad:00:00:be:ef"

			BeforeEach(func() {
				// The SR-IOV VF MAC should be preserved on migration, therefore explicitly specify it.
				vmi = newSRIOVVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

				var err error
				vmi, err = createVMIAndWait(vmi)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vmi)

				ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 30*time.Second)
				Expect(err).NotTo(HaveOccurred())
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(checkMacAddress(vmi, ifaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest")
			})

			It("should be successful with a running VMI on the target", func() {
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				// It may take some time for the VMI interface status to be updated with the information reported by
				// the guest-agent.
				ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 30*time.Second)
				Expect(err).NotTo(HaveOccurred())
				updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(checkMacAddress(updatedVMI, ifaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest after migration")
			})
		})
	})

	Context("VMI connected to two SRIOV networks", func() {
		BeforeEach(func() {
			Expect(createSriovNetworkAttachmentDefinition(sriovnet1, util.NamespaceTestDefault, sriovConfNAD)).To(Succeed(), shouldCreateNetwork)
			Expect(createSriovNetworkAttachmentDefinition(sriovnet2, util.NamespaceTestDefault, sriovConfNAD)).To(Succeed(), shouldCreateNetwork)
		})

		It("[test_id:1755]should create a virtual machine with two sriov interfaces referring the same resource", func() {
			sriovNetworks := []string{sriovnet1, sriovnet2}
			vmi := newSRIOVVmi(sriovNetworks, defaultCloudInitNetworkData())
			vmi.Spec.Domain.Devices.Interfaces[1].PciAddress = "0000:06:00.0"
			vmi.Spec.Domain.Devices.Interfaces[2].PciAddress = "0000:07:00.0"

			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variables are defined in pod")
			for _, name := range sriovNetworks {
				Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, name, sriovResourceName)).To(Succeed())
			}

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

			By("checking virtual machine instance has three interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1", "eth2"})

			Expect(pciAddressExistsInGuest(vmi, vmi.Spec.Domain.Devices.Interfaces[1].PciAddress)).To(Succeed())
			Expect(pciAddressExistsInGuest(vmi, vmi.Spec.Domain.Devices.Interfaces[2].PciAddress)).To(Succeed())
		})
	})

	Context("Connected to multiple SRIOV networks", func() {
		sriovNetworks := []string{sriovnet1, sriovnet2, sriovnet3, sriovnet4}
		BeforeEach(func() {
			for _, sriovNetwork := range sriovNetworks {
				Expect(createSriovNetworkAttachmentDefinition(sriovNetwork, util.NamespaceTestDefault, sriovConfNAD)).To(Succeed(), shouldCreateNetwork)
			}
		})

		It("should correctly plug all the interfaces based on the specified MAC and (guest) PCI addresses", func() {
			macAddressTemplate := "de:ad:00:be:ef:%02d"
			pciAddressTemplate := "0000:2%d:00.0"
			vmi := newSRIOVVmi(sriovNetworks, defaultCloudInitNetworkData())
			for i := range sriovNetworks {
				secondaryInterfaceIdx := i + 1
				vmi.Spec.Domain.Devices.Interfaces[secondaryInterfaceIdx].MacAddress = fmt.Sprintf(macAddressTemplate, secondaryInterfaceIdx)
				vmi.Spec.Domain.Devices.Interfaces[secondaryInterfaceIdx].PciAddress = fmt.Sprintf(pciAddressTemplate, secondaryInterfaceIdx)
			}

			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())

			for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
				if iface.SRIOV == nil {
					continue
				}
				guestInterfaceName, err := findIfaceByMAC(virtClient, vmi, iface.MacAddress, 30*time.Second)
				Expect(err).ToNot(HaveOccurred())
				Expect(pciAddressExistsInGuestInterface(vmi, iface.PciAddress, guestInterfaceName)).To(Succeed())
			}
		})
	})

	Context("VMI connected to link-enabled SRIOV network", func() {
		var sriovNode string

		BeforeEach(func() {
			var err error
			sriovNode, err = sriovNodeName(sriovResourceName)
			Expect(err).ToNot(HaveOccurred())
		})

		BeforeEach(func() {
			Expect(createSriovNetworkAttachmentDefinition(sriovnetLinkEnabled, util.NamespaceTestDefault, sriovLinkEnableConfNAD)).
				To(Succeed(), shouldCreateNetwork)
		})

		It("[test_id:3956]should connect to another machine with sriov interface over IPv4", func() {
			cidrA := "192.168.1.1/24"
			cidrB := "192.168.1.2/24"
			ipA, err := libnet.CidrToIP(cidrA)
			Expect(err).ToNot(HaveOccurred())
			ipB, err := libnet.CidrToIP(cidrB)
			Expect(err).ToNot(HaveOccurred())

			//create two vms on the same sriov network
			vmi1, err := createSRIOVVmiOnNode(sriovNode, sriovnetLinkEnabled, cidrA)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi1)
			vmi2, err := createSRIOVVmiOnNode(sriovNode, sriovnetLinkEnabled, cidrB)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi2)

			vmi1, err = waitVMI(vmi1)
			Expect(err).NotTo(HaveOccurred())
			vmi2, err = waitVMI(vmi2)
			Expect(err).NotTo(HaveOccurred())

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
			vmi1IP, err := libnet.CidrToIP(vmi1CIDR)
			Expect(err).ToNot(HaveOccurred())
			vmi2IP, err := libnet.CidrToIP(vmi2CIDR)
			Expect(err).ToNot(HaveOccurred())

			//create two vms on the same sriov network
			vmi1, err := createSRIOVVmiOnNode(sriovNode, sriovnetLinkEnabled, vmi1CIDR)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi1)
			vmi2, err := createSRIOVVmiOnNode(sriovNode, sriovnetLinkEnabled, vmi2CIDR)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi2)

			vmi1, err = waitVMI(vmi1)
			Expect(err).NotTo(HaveOccurred())
			vmi2, err = waitVMI(vmi2)
			Expect(err).NotTo(HaveOccurred())

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
				ipVlaned1, err = libnet.CidrToIP(cidrVlaned1)
				Expect(err).ToNot(HaveOccurred())
				Expect(createSriovNetworkAttachmentDefinition(sriovnetVlanned, util.NamespaceTestDefault, sriovVlanConfNAD)).To(Succeed())
			})

			It("should be able to ping between two VMIs with the same VLAN over SRIOV network", func() {
				vlanedVMI1, err := createSRIOVVmiOnNode(sriovNode, sriovnetVlanned, cidrVlaned1)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vlanedVMI1)
				vlanedVMI2, err := createSRIOVVmiOnNode(sriovNode, sriovnetVlanned, "192.168.0.2/24")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vlanedVMI2)

				_, err = waitVMI(vlanedVMI1)
				Expect(err).NotTo(HaveOccurred())
				vlanedVMI2, err = waitVMI(vlanedVMI2)
				Expect(err).NotTo(HaveOccurred())

				By("pinging from vlanedVMI2 and the anonymous vmi over vlan")
				Eventually(func() error {
					return libnet.PingFromVMConsole(vlanedVMI2, ipVlaned1)
				}, 15*time.Second, time.Second).ShouldNot(HaveOccurred())
			})

			It("should NOT be able to ping between Vlaned VMI and a non Vlaned VMI", func() {
				vlanedVMI, err := createSRIOVVmiOnNode(sriovNode, sriovnetVlanned, cidrVlaned1)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vlanedVMI)
				nonVlanedVMI, err := createSRIOVVmiOnNode(sriovNode, sriovnetLinkEnabled, "192.168.0.3/24")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, nonVlanedVMI)

				_, err = waitVMI(vlanedVMI)
				Expect(err).NotTo(HaveOccurred())
				nonVlanedVMI, err = waitVMI(nonVlanedVMI)
				Expect(err).NotTo(HaveOccurred())

				By("pinging between nonVlanedVMIand the anonymous vmi")
				Eventually(func() error {
					return libnet.PingFromVMConsole(nonVlanedVMI, ipVlaned1)
				}, 15*time.Second, time.Second).Should(HaveOccurred())
			})
		})
	})
})

func readSRIOVResourceName() string {
	sriovResourceName := os.Getenv("SRIOV_RESOURCE_NAME")
	if sriovResourceName == "" {
		const defaultTestSRIOVResourceName = "kubevirt.io/sriov_net"
		sriovResourceName = defaultTestSRIOVResourceName
	}
	return sriovResourceName
}

func pciAddressExistsInGuest(vmi *v1.VirtualMachineInstance, pciAddress string) error {
	command := fmt.Sprintf("grep -q %s /sys/class/net/*/device/uevent\n", pciAddress)
	return console.RunCommand(vmi, command, 15*time.Second)
}

func pciAddressExistsInGuestInterface(vmi *v1.VirtualMachineInstance, pciAddress, interfaceName string) error {
	command := fmt.Sprintf("grep -q PCI_SLOT_NAME=%s /sys/class/net/%s/device/uevent\n", pciAddress, interfaceName)
	return console.RunCommand(vmi, command, 15*time.Second)
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
	nodes := libnode.GetAllSchedulableNodes(virtClient)
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
	out, err := exec.ExecuteCommandOnPod(
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
	networkData := libnet.CreateDefaultCloudInitNetworkData()
	return networkData
}

func findIfaceByMAC(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, mac string, timeout time.Duration) (string, error) {
	var ifaceName string
	err := wait.Poll(timeout, 5*time.Second, func() (done bool, err error) {
		vmi, err := virtClient.VirtualMachineInstance(vmi.GetNamespace()).Get(context.Background(), vmi.GetName(), &k8smetav1.GetOptions{})
		if err != nil {
			return false, err
		}

		ifaceName, err = getInterfaceNameByMAC(vmi, mac)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("could not find interface with MAC %q on VMI %q: %v", mac, vmi.Name, err)
	}
	return ifaceName, nil
}

func newSRIOVVmi(networks []string, cloudInitNetworkData string) *v1.VirtualMachineInstance {
	options := []libvmi.Option{
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	}
	if cloudInitNetworkData != "" {
		cloudinitOption := libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkData)
		options = append(options, cloudinitOption)
	}

	for _, name := range networks {
		options = append(options,
			libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(name)),
			libvmi.WithNetwork(libvmi.MultusNetwork(name, name)),
		)
	}
	return libvmi.NewFedora(options...)
}

func checkInterfacesInGuest(vmi *v1.VirtualMachineInstance, interfaces []string) {
	for _, iface := range interfaces {
		Expect(libnet.InterfaceExists(vmi, iface)).To(Succeed())
	}
}

// createVMIAndWait creates the received VMI and waits for the guest to load the guest-agent
func createVMIAndWait(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error) {
	virtClient := kubevirt.Client()

	var err error
	vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
	if err != nil {
		return nil, err
	}

	return waitVMI(vmi)
}

func waitVMI(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error) {
	// Running multi sriov jobs with Kind, DinD is resource extensive, causing DeadlineExceeded transient warning
	// Kubevirt re-enqueue the request once it happens, so its safe to ignore this warning.
	// see https://github.com/kubevirt/kubevirt/issues/5027
	warningsIgnoreList := []string{"unknown error encountered sending command SyncVMI: rpc error: code = DeadlineExceeded desc = context deadline exceeded"}
	libwait.WaitUntilVMIReady(vmi, console.LoginToFedora, libwait.WithWarningsIgnoreList(warningsIgnoreList))

	virtClient := kubevirt.Client()

	Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

	return virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
}

// deleteVMI deletes the specified VMI and waits for its absence.
// Waiting for the VMI removal is placed intentionally for VMI/s with SR-IOV networks in order
// to assure resources (VF/s) are fully released before reused again on a new VMI.
// Ref: https://github.com/k8snetworkplumbingwg/sriov-cni/issues/219
func deleteVMI(vmi *v1.VirtualMachineInstance) error {
	virtClient := kubevirt.Client()

	GinkgoWriter.Println("sriov:", vmi.Name, "deletion started")
	if err := virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &k8smetav1.DeleteOptions{}); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete VMI: %v", err)
	}

	const timeout = 30 * time.Second
	return wait.PollImmediate(1*time.Second, timeout, func() (done bool, err error) {
		_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func checkDefaultInterfaceInPod(vmi *v1.VirtualMachineInstance) error {
	vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

	virtClient := kubevirt.Client()

	By("checking default interface is present")
	_, err := exec.ExecuteCommandOnPod(
		virtClient,
		vmiPod,
		"compute",
		[]string{"ip", "address", "show", "eth0"},
	)
	if err != nil {
		return err
	}

	By("checking default interface is attached to VMI")
	_, err = exec.ExecuteCommandOnPod(
		virtClient,
		vmiPod,
		"compute",
		[]string{"ip", "address", "show", "k6t-eth0"},
	)
	if err != nil {
		return err
	}

	return nil
}

// createSRIOVVmiOnNode creates a VMI on the specified node, connected to the specified SR-IOV network.
func createSRIOVVmiOnNode(nodeName, networkName, cidr string) (*v1.VirtualMachineInstance, error) {
	// Explicitly choose different random mac addresses instead of relying on kubemacpool to do it:
	// 1) we don't at the moment deploy kubemacpool in kind providers
	// 2) even if we would do, it's probably a good idea to have the suite not depend on this fact
	//
	// This step is needed to guarantee that no VFs on the PF carry a duplicate MAC address that may affect
	// ability of VMIs to send and receive ICMP packets on their ports.
	mac, err := GenerateRandomMac()
	if err != nil {
		return nil, err
	}

	// manually configure IP/link on sriov interfaces because there is
	// no DHCP server to serve the address to the guest
	vmi := newSRIOVVmi([]string{networkName}, cloudInitNetworkDataWithStaticIPsByMac(networkName, mac.String(), cidr))
	const secondaryInterfaceIndex = 1
	vmi.Spec.Domain.Devices.Interfaces[secondaryInterfaceIndex].MacAddress = mac.String()

	vmi = tests.CreateVmiOnNode(vmi, nodeName)

	return vmi, nil
}

func sriovNodeName(sriovResourceName string) (string, error) {
	virtClient := kubevirt.Client()

	sriovNodes := getNodesWithAllocatedResource(virtClient, sriovResourceName)
	if len(sriovNodes) == 0 {
		return "", fmt.Errorf("failed to detect nodes with allocatable resources (%s)", sriovResourceName)
	}
	return sriovNodes[0].Name, nil
}
