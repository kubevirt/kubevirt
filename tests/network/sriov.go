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
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	expect "github.com/google/goexpect"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/net/dns"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	netcloudinit "kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	defaultVLAN  = 0
	specificVLAN = 200
)

const (
	sriovnet1           = "sriov"
	sriovnet2           = "sriov2"
	sriovnet3           = "sriov3"
	sriovnet4           = "sriov4"
	sriovnetLinkEnabled = "sriov-linked"
)

var pciAddressRegex = regexp.MustCompile(hardware.PCI_ADDRESS_PATTERN)

var _ = Describe("SRIOV", Serial, decorators.SRIOV, func() {
	var virtClient kubecli.KubevirtClient

	sriovResourceName := readSRIOVResourceName()

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		// Check if the hardware supports SRIOV
		if err := validateSRIOVSetup(virtClient, sriovResourceName, 1); err != nil {
			Skip("Sriov is not enabled in this environment. Skip these tests using - export FUNC_TEST_ARGS='--skip=SRIOV'")
		}
	})

	Context("VMI connected to single SRIOV network", func() {
		BeforeEach(func() {
			netAttachDef := libnet.NewSriovNetAttachDef(sriovnet1, defaultVLAN)
			netAttachDef.Annotations = map[string]string{libnet.ResourceNameAnnotation: sriovResourceName}
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.NamespaceTestDefault, netAttachDef)
			Expect(err).NotTo(HaveOccurred(), shouldCreateNetwork)
		})

		It("should have cloud-init meta_data with tagged interface and aligned cpus to sriov interface numa node for VMIs with dedicatedCPUs", decorators.RequiresNodeWithCPUManager, func() {
			vmi := newSRIOVVmi([]string{sriovnet1}, libvmi.WithCloudInitConfigDrive(libvmici.WithConfigDriveNetworkData(defaultCloudInitNetworkData())))
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

			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

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
			vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
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
			Expect(mountGuestDevice(vmi, "config-2")).To(Succeed())

			By("checking cloudinit meta-data")
			const consoleCmd = `cat /mnt/openstack/latest/meta_data.json; printf "@@"`
			res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
				&expect.BSnd{S: consoleCmd + console.CRLF},
				&expect.BExp{R: `(.*)@@`},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
			rawOutput := res[len(res)-1].Output
			Expect(trimRawString2JSON(rawOutput)).To(MatchJSON(buf))
		})

		It("[test_id:1754]should create a virtual machine with sriov interface", func() {
			vmi := newSRIOVVmi([]string{sriovnet1}, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
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
			vmi := newSRIOVVmi([]string{sriovnet1}, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
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
			expectedInterfaces := []string{"eth0", "eth1"}
			checkInterfacesInGuest(vmi, expectedInterfaces)

			for _, iface := range expectedInterfaces {
				Expect(isInterfaceOnRootPCIComplex(vmi, iface)).To(BeTrue(), fmt.Sprintf("Expected interface %s on PCI root complex", iface))
			}
		})

		It("[test_id:3959]should create a virtual machine with sriov interface and dedicatedCPUs", decorators.RequiresNodeWithCPUManager, func() {
			// In addition to verifying that we can start a VMI with CPU pinning
			// this also tests if we've correctly calculated the overhead for VFIO devices.
			vmi := newSRIOVVmi([]string{sriovnet1}, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
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
			vmi := newSRIOVVmi([]string{sriovnet1}, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
			vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			By("checking virtual machine instance has an interface with the requested MAC address")
			ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 140*time.Second)
			Expect(err).NotTo(HaveOccurred())
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(libnet.CheckMacAddress(vmi, ifaceName, mac)).To(Succeed())

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
				vmi = newSRIOVVmi([]string{sriovnet1}, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
				vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

				var err error
				vmi, err = createVMIAndWait(vmi)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vmi)

				ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 30*time.Second)
				Expect(err).NotTo(HaveOccurred())
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(libnet.CheckMacAddress(vmi, ifaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest")
			})

			It("should be successful with a running VMI on the target", func() {
				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				// It may take some time for the VMI interface status to be updated with the information reported by
				// the guest-agent.
				ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 2*time.Minute+10*time.Second)
				Expect(err).NotTo(HaveOccurred())
				updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(libnet.CheckMacAddress(updatedVMI, ifaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest after migration")
			})
		})

		Context("memory hotplug", Serial, decorators.RequiresTwoSchedulableNodes, func() {
			BeforeEach(func() {
				virtClient := kubevirt.Client()
				updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
					WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
				}
				rolloutStrategy := pointer.P(v1.VMRolloutStrategyLiveUpdate)
				originalKv := libkubevirt.GetCurrentKv(virtClient)
				patchWorkloadUpdateMethodAndRolloutStrategy(originalKv.Name, virtClient, updateStrategy, rolloutStrategy)

				currentKv := libkubevirt.GetCurrentKv(virtClient)
				config.WaitForConfigToBePropagatedToComponent(
					"kubevirt.io=virt-controller",
					currentKv.ResourceVersion,
					config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
					time.Minute)
			})

			It("Should successfully reattach host-device", func() {
				const (
					initialGuestMemory      = "1Gi"
					updatedGuestMemory      = "3Gi"
					sriovNetworkLogicalName = "sriov-network"
				)
				vmi := libvmifact.NewAlpineWithTestTooling(
					libvmi.WithGuestMemory(initialGuestMemory),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(sriovNetworkLogicalName)),
					libvmi.WithNetwork(libvmi.MultusNetwork(sriovNetworkLogicalName, sriovnet1)),
				)
				vmi.Spec.Domain.Resources.Requests = nil

				vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

				vm, err := kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				Eventually(matcher.ThisVM(vm)).WithTimeout(6 * time.Minute).WithPolling(3 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Hotplugging additional memory")
				patchData, err := patch.GenerateTestReplacePatch(
					"/spec/template/spec/domain/memory/guest",
					initialGuestMemory,
					updatedGuestMemory,
				)
				Expect(err).NotTo(HaveOccurred())

				_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring the VMI has more available guest memory")
				initialGuestMemoryQuantity := resource.MustParse(initialGuestMemory)
				Eventually(func() int64 {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return vmi.Status.Memory.GuestCurrent.Value()
				}).
					WithTimeout(5 * time.Minute).
					WithPolling(5 * time.Second).
					Should(BeNumerically(">", initialGuestMemoryQuantity.Value()))

				By("Ensuring SR-IOV device was hotplugged to the VMI")
				Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).
						Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return vmi.Status.Interfaces
				}).
					WithTimeout(5 * time.Minute).
					WithPolling(5 * time.Second).
					Should(ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Name":       Equal(sriovNetworkLogicalName),
							"InfoSource": ContainSubstring(vmispec.InfoSourceDomain),
						}),
					))
			})
		})
	})

	Context("VMI connected to two SRIOV networks", func() {
		BeforeEach(func() {
			netAttachDef1 := libnet.NewSriovNetAttachDef(sriovnet1, defaultVLAN)
			netAttachDef1.Annotations = map[string]string{libnet.ResourceNameAnnotation: sriovResourceName}
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.NamespaceTestDefault, netAttachDef1)
			Expect(err).NotTo(HaveOccurred(), shouldCreateNetwork)

			netAttachDef2 := libnet.NewSriovNetAttachDef(sriovnet2, defaultVLAN)
			netAttachDef2.Annotations = map[string]string{libnet.ResourceNameAnnotation: sriovResourceName}
			_, err = libnet.CreateNetAttachDef(context.Background(), testsuite.NamespaceTestDefault, netAttachDef2)
			Expect(err).NotTo(HaveOccurred(), shouldCreateNetwork)
		})

		It("[test_id:1755]should create a virtual machine with two sriov interfaces referring the same resource", func() {
			sriovNetworks := []string{sriovnet1, sriovnet2}
			vmi := newSRIOVVmi(sriovNetworks, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
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
				netAttachDef := libnet.NewSriovNetAttachDef(sriovNetwork, defaultVLAN)
				netAttachDef.Annotations = map[string]string{libnet.ResourceNameAnnotation: sriovResourceName}
				_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.NamespaceTestDefault, netAttachDef)
				Expect(err).NotTo(HaveOccurred(), shouldCreateNetwork)
			}
		})

		It("should correctly plug all the interfaces based on the specified MAC and (guest) PCI addresses", func() {
			macAddressTemplate := "de:ad:00:be:ef:%02d"
			pciAddressTemplate := "0000:2%d:00.0"
			vmi := newSRIOVVmi(sriovNetworks, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
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
			netAttachDef := libnet.NewSriovNetAttachDef(sriovnetLinkEnabled, defaultVLAN, libnet.WithLinkState())
			netAttachDef.Annotations = map[string]string{libnet.ResourceNameAnnotation: sriovResourceName}
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.NamespaceTestDefault, netAttachDef)
			Expect(err).NotTo(HaveOccurred(), shouldCreateNetwork)
		})

		It("[test_id:3956]should connect to another machine with sriov interface over IP", func() {
			cidrA := "192.168.1.1/24"
			cidrB := "192.168.1.2/24"
			ipA, err := libnet.CidrToIP(cidrA)
			Expect(err).ToNot(HaveOccurred())
			ipB, err := libnet.CidrToIP(cidrB)
			Expect(err).ToNot(HaveOccurred())

			// create two vms on the same sriov network
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

				netAttachDef := libnet.NewSriovNetAttachDef(sriovnetVlanned, specificVLAN, libnet.WithLinkState())
				netAttachDef.Annotations = map[string]string{libnet.ResourceNameAnnotation: sriovResourceName}
				_, err = libnet.CreateNetAttachDef(context.Background(), testsuite.NamespaceTestDefault, netAttachDef)
				Expect(err).NotTo(HaveOccurred(), shouldCreateNetwork)
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
	sriovNodes := getNodesWithAllocatedResource(sriovResourceName)
	if len(sriovNodes) < minRequiredNodes {
		return fmt.Errorf("not enough compute nodes with SR-IOV support detected")
	}
	return nil
}

func getNodesWithAllocatedResource(resourceName string) []k8sv1.Node {
	nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
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
	pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).NotTo(HaveOccurred())

	out, err := exec.ExecuteCommandOnPod(
		pod,
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
	networkData := netcloudinit.CreateDefaultCloudInitNetworkData()
	return networkData
}

func findIfaceByMAC(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, mac string, timeout time.Duration) (string, error) {
	var ifaceName string
	err := virtwait.PollImmediately(5*time.Second, timeout, func(ctx context.Context) (done bool, err error) {
		vmi, err := virtClient.VirtualMachineInstance(vmi.GetNamespace()).Get(ctx, vmi.GetName(), k8smetav1.GetOptions{})
		if err != nil {
			return false, err
		}

		ifaceName, err = getInterfaceNameByMAC(vmi, mac)
		if err != nil {
			return false, nil
		}
		if ifaceName == "" {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("could not find interface with MAC %q on VMI %q: %v", mac, vmi.Name, err)
	}
	return ifaceName, nil
}

func newSRIOVVmi(networks []string, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	options := []libvmi.Option{
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	}

	for _, name := range networks {
		options = append(options,
			libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(name)),
			libvmi.WithNetwork(libvmi.MultusNetwork(name, name)),
		)
	}
	opts = append(options, opts...)
	return libvmifact.NewFedora(opts...)
}

func checkInterfacesInGuest(vmi *v1.VirtualMachineInstance, interfaces []string) {
	for _, iface := range interfaces {
		Expect(libnet.InterfaceExists(vmi, iface)).To(Succeed())
	}
}

// isInterfaceOnRootPCIComplex checks whether device is on root complex
// Follow the sysfs path of the interface stating from bus 0
// If the interface is on root complex, we expect it to have a PCI address in
// the format of 0000:00:??.0, because we hard code everything to 0 during assignment
// except for the slot which we allocate dynamically.
// In addition, we expect only one PCI address along the path, otherwise it is an indication
// that another device is bridging between the interface and the root-complex, i.e.
// a pcie-root-port allocated by libvirt.
//
// Examples:
// on root       /sys/devices/pci0000:00/0000:00:03.0/net/eth1
// not on root   /sys/devices/pci0000:00/0000:00:02.7/0000:08:00.0/net/eth1
// Another device (virtio) may look like this:
// on root 		/sys/devices/pci0000:00/0000:00:02.0/virtio0/net/eth0
// not on root 	/sys/devices/pci0000:00/0000:00:02.0/0000:01:00.0/virtio0/net/eth0
func isInterfaceOnRootPCIComplex(vmi *v1.VirtualMachineInstance, iface string) (bool, error) {
	const bus0Path = "/sys/devices/pci0000:00/"
	ifacePath, err := console.RunCommandAndStoreOutput(vmi,
		fmt.Sprintf("find %s -name %s", bus0Path, iface), 15*time.Second)
	if err != nil {
		return false, err
	}
	if ifacePath == "" {
		return false, fmt.Errorf("interface %s not found under pci0000:00", iface)
	}

	ifacePath, _ = strings.CutPrefix(ifacePath, bus0Path)

	ifacePathSlice := strings.Split(ifacePath, "/")
	// expecting [0000:00:??.0 net eth?] or [0000:00:??.0 virtio0 net eth?]
	if len(ifacePathSlice) < 3 {
		return false, fmt.Errorf("interface path %s not as expected", ifacePath)
	}
	if !strings.HasPrefix(ifacePathSlice[0], "0000:00:") || !strings.HasSuffix(ifacePathSlice[0], ".0") {
		// if we assigned the device to root complex it must be on 0000:00, function 0, only the slot is dynamic
		return false, nil
	}

	if pciAddressRegex.MatchString(ifacePathSlice[0]) && pciAddressRegex.MatchString(ifacePathSlice[1]) {
		// we have two PCI addresses in the path=> a device is bridging between the interface and the root complex
		return false, nil
	}
	return true, nil
}

// createVMIAndWait creates the received VMI and waits for the guest to load the guest-agent
func createVMIAndWait(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error) {
	virtClient := kubevirt.Client()

	var err error
	vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return waitVMI(vmi)
}

func waitVMI(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error) {
	warningsIgnoreList := []string{
		// Running multi sriov jobs with Kind, DinD is resource extensive, causing DeadlineExceeded transient warning
		// Kubevirt re-enqueue the request once it happens, so its safe to ignore this warning.
		// see https://github.com/kubevirt/kubevirt/issues/5027
		"unknown error encountered sending command SyncVMI: rpc error: code = DeadlineExceeded desc = context deadline exceeded",

		// SR-IOV tests run on Kind cluster in a docker-in-docker CI environment which is resource extensive,
		// causing general slowness in the cluster.
		// It manifests having warning event raised due to SR-IOV VMs virt-launcher pod's SR-IOV network-pci-map
		// downward API mount not become ready on time.
		// Kubevirt re-enqueues such requests and eventually reconcile this state.
		// Tacking issue: https://github.com/kubevirt/kubevirt/issues/12691
		"failed to create SR-IOV hostdevices: failed to create PCI address pool with network status from file: context deadline exceeded: file is not populated with network-info",
	}

	libwait.WaitUntilVMIReady(vmi, console.LoginToFedora, libwait.WithWarningsIgnoreList(warningsIgnoreList))

	virtClient := kubevirt.Client()

	Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

	return virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
}

// deleteVMI deletes the specified VMI and waits for its absence.
// Waiting for the VMI removal is placed intentionally for VMI/s with SR-IOV networks in order
// to assure resources (VF/s) are fully released before reused again on a new VMI.
// Ref: https://github.com/k8snetworkplumbingwg/sriov-cni/issues/219
func deleteVMI(vmi *v1.VirtualMachineInstance) error {
	virtClient := kubevirt.Client()

	GinkgoWriter.Println("sriov:", vmi.Name, "deletion started")
	if err := virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, k8smetav1.DeleteOptions{}); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete VMI: %v", err)
	}

	const timeout = 30 * time.Second
	return virtwait.PollImmediately(1*time.Second, timeout, func(ctx context.Context) (done bool, err error) {
		_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(ctx, vmi.Name, k8smetav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func checkDefaultInterfaceInPod(vmi *v1.VirtualMachineInstance) error {
	vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).NotTo(HaveOccurred())

	By("checking default interface is present")
	_, err = exec.ExecuteCommandOnPod(
		vmiPod,
		"compute",
		[]string{"ip", "address", "show", "eth0"},
	)
	if err != nil {
		return err
	}

	By("checking default interface is attached to VMI")
	_, err = exec.ExecuteCommandOnPod(
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
	mac, err := libnet.GenerateRandomMac()
	if err != nil {
		return nil, err
	}

	// manually configure IP/link on sriov interfaces because there is
	// no DHCP server to serve the address to the guest
	networkData := netcloudinit.CreateNetworkDataWithStaticIPsByMac(networkName, mac.String(), cidr)
	vmi := newSRIOVVmi([]string{networkName}, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)))
	libvmi.WithNodeAffinityFor(nodeName)(vmi)
	const secondaryInterfaceIndex = 1
	vmi.Spec.Domain.Devices.Interfaces[secondaryInterfaceIndex].MacAddress = mac.String()

	virtCli := kubevirt.Client()
	vmi, err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return vmi, nil
}

func sriovNodeName(sriovResourceName string) (string, error) {
	sriovNodes := getNodesWithAllocatedResource(sriovResourceName)
	if len(sriovNodes) == 0 {
		return "", fmt.Errorf("failed to detect nodes with allocatable resources (%s)", sriovResourceName)
	}
	return sriovNodes[0].Name, nil
}

func mountGuestDevice(vmi *v1.VirtualMachineInstance, devName string) error {
	cmdCheck := fmt.Sprintf("mount $(blkid  -L %s) /mnt/\n", devName)
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "sudo su -\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
}

// trimRawString2JSON remove string left of first { and right of last }
// e.g. xxx { yyy } zzzz => { yyy }
func trimRawString2JSON(input string) string {
	startIdx := strings.Index(input, "{")
	endIdx := strings.LastIndex(input, "}")
	if startIdx == -1 || endIdx == -1 {
		return ""
	}
	return input[startIdx : endIdx+1]
}

func patchWorkloadUpdateMethodAndRolloutStrategy(kvName string, virtClient kubecli.KubevirtClient, updateStrategy *v1.KubeVirtWorkloadUpdateStrategy, rolloutStrategy *v1.VMRolloutStrategy) {
	methodData, err := json.Marshal(updateStrategy)
	ExpectWithOffset(1, err).To(Not(HaveOccurred()))
	rolloutData, err := json.Marshal(rolloutStrategy)
	ExpectWithOffset(1, err).To(Not(HaveOccurred()))

	data1 := fmt.Sprintf(`{"op": "replace", "path": "/spec/workloadUpdateStrategy", "value": %s}`, string(methodData))
	data2 := fmt.Sprintf(`{"op": "replace", "path": "/spec/configuration/vmRolloutStrategy", "value": %s}`, string(rolloutData))
	data := []byte(fmt.Sprintf(`[%s, %s]`, data1, data2))

	EventuallyWithOffset(1, func() error {
		_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kvName, types.JSONPatchType, data, metav1.PatchOptions{})
		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}
