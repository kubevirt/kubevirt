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
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	expect "github.com/google/goexpect"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/net/dns"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	netcloudinit "kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// DRA-based SR-IOV Tests
// These tests mirror the Multus-based SR-IOV tests above, but use DRA (Dynamic Resource Allocation)
// for device management instead of NetworkAttachmentDefinition + device plugin approach.
//
// DRA SR-IOV uses BOTH NetworkAttachmentDefinition (for CNI config) AND ResourceClaimTemplate
// (for device allocation with VfConfig parameters).
var _ = Describe(SIG("DRA-SRIOV", decorators.DRANetwork, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("VMI connected to single DRA SR-IOV network", func() {
		var (
			claimName    = "dra-sriov-claim"
			networkName  = "dra-sriov-net"
			templateName = "single-vf-dra-sriov-net"
			driverName   = "sriovnetwork.k8snetworkplumbingwg.io"
		)

		BeforeEach(func() {
			err := libnet.CreateSRIOVNetworkWithDRA(
				context.Background(),
				testsuite.NamespaceTestDefault,
				networkName,
				driverName,
				defaultVLAN,
			)
			Expect(err).NotTo(HaveOccurred(), "should create NAD and ResourceClaimTemplate")

			DeferCleanup(func() {
				err := libnet.DeleteResourceClaimTemplate(context.Background(), testsuite.NamespaceTestDefault, "single-vf-"+networkName)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should have cloud-init meta_data with tagged interface and aligned cpus to DRA sriov interface numa node for VMIs with dedicatedCPUs", decorators.RequiresNodeWithCPUManager, func() {
			vmi := newDRASRIOVVmi([]string{claimName}, templateName, libvmi.WithCloudInitConfigDrive(libvmici.WithConfigDriveNetworkData(defaultCloudInitNetworkData())))
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 4,
				DedicatedCPUPlacement: true,
			}

			for idx, iface := range vmi.Spec.Domain.Devices.Interfaces {
				if iface.Name == claimName {
					iface.Tag = "specialNet"
					vmi.Spec.Domain.Devices.Interfaces[idx] = iface
				}
			}
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			nic := domSpec.Devices.HostDevices[0]
			// find the SRIOV interface
			for _, iface := range domSpec.Devices.HostDevices {
				if iface.Alias.GetName() == claimName {
					nic = iface
				}
			}
			address := nic.Address
			pciAddrStr := fmt.Sprintf("%s:%s:%s.%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
			srcAddr := nic.Source.Address
			sourcePCIAddress := fmt.Sprintf("%s:%s:%s.%s", srcAddr.Domain[2:], srcAddr.Bus[2:], srcAddr.Slot[2:], srcAddr.Function[2:])
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			alignedCPUsInt := lookupDeviceVCPUAffinityOnPod(vmiPod, sourcePCIAddress, domSpec)
			Expect(alignedCPUsInt).ToNot(BeEmpty(), "expected aligned CPUs for SR-IOV device")
			deviceData := []cloudinit.DeviceData{
				{
					Type:        cloudinit.NICMetadataType,
					Bus:         nic.Address.Type,
					Address:     pciAddrStr,
					Tags:        []string{"specialNet"},
					AlignedCPUs: alignedCPUsInt,
				},
			}
			vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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

		It("should create a virtual machine with DRA sriov interface", func() {
			vmi := newDRASRIOVVmi([]string{claimName}, templateName, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})
		})

		It("should create a virtual machine with DRA sriov interface with all pci devices on the root bus", func() {
			vmi := newDRASRIOVVmi([]string{claimName}, templateName, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
			vmi.Annotations = map[string]string{
				v1.PlacePCIDevicesOnRootComplex: "true",
			}
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

			By("checking virtual machine instance has two interfaces")
			expectedInterfaces := []string{"eth0", "eth1"}
			checkInterfacesInGuest(vmi, expectedInterfaces)

			for _, iface := range expectedInterfaces {
				Expect(isInterfaceOnRootPCIComplex(vmi, iface)).To(BeTrue(), fmt.Sprintf("Expected interface %s on PCI root complex", iface))
			}
		})

		It("should create a virtual machine with DRA sriov interface and dedicatedCPUs", decorators.RequiresNodeWithCPUManager, func() {
			vmi := newDRASRIOVVmi([]string{claimName}, templateName, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				DedicatedCPUPlacement: true,
			}
			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

			By("checking virtual machine instance has two interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})
		})

		It("should create a virtual machine with DRA sriov interface with custom MAC address", func() {
			macAddr, err := libnet.GenerateRandomMac()
			Expect(err).NotTo(HaveOccurred())
			mac := macAddr.String()
			vmi := newDRASRIOVVmi([]string{claimName}, templateName, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
			vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

			vmi, err = createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			By("checking virtual machine instance has an interface with the requested MAC address")
			ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 140*time.Second)
			Expect(err).NotTo(HaveOccurred())
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(libnet.CheckMacAddress(vmi, ifaceName, mac)).To(Succeed())

			By("checking virtual machine instance reports the expected network name and info source")
			Eventually(func() error {
				updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}

				if networkName := getInterfaceNetworkNameByMAC(updatedVMI, mac); networkName != claimName {
					return fmt.Errorf("expected network name %q for MAC %q, got %q", claimName, mac, networkName)
				}

				networkInterface := vmispec.LookupInterfaceStatusByMac(updatedVMI.Status.Interfaces, mac)
				if networkInterface == nil {
					return fmt.Errorf("interface not found")
				}

				if !vmispec.ContainsInfoSource(networkInterface.InfoSource, vmispec.InfoSourceDomain) ||
					!vmispec.ContainsInfoSource(networkInterface.InfoSource, vmispec.InfoSourceGuestAgent) {
					return fmt.Errorf("interface status for %q does not contain all info sources %q", networkInterface.Name, networkInterface.InfoSource)
				}

				return nil
			}).WithTimeout(5 * time.Minute).WithPolling(time.Second).Should(Succeed())
		})

		Context("migration", decorators.RequiresTwoSchedulableNodes, func() {
			var vmi *v1.VirtualMachineInstance
			var mac string

			BeforeEach(func() {
				macAddr, err := libnet.GenerateRandomMac()
				Expect(err).NotTo(HaveOccurred())
				mac = macAddr.String()

				vmi = newDRASRIOVVmi([]string{claimName}, templateName, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())))
				vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

				vmi, err = createVMIAndWait(vmi)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vmi)

				ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 30*time.Second)
				Expect(err).NotTo(HaveOccurred())
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
				var sriovIfaceName string
				Eventually(func() error {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					if err != nil {
						return err
					}

					ifaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, claimName)
					if ifaceStatus == nil {
						return fmt.Errorf("interface status for %q was not found", claimName)
					}

					if !vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceDomain) ||
						!vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceGuestAgent) {
						return fmt.Errorf("interface status for %q does not contain all info sources %q", ifaceStatus.Name, ifaceStatus.InfoSource)
					}

					sriovIfaceName = ifaceStatus.InterfaceName

					return nil
				}).WithTimeout(5 * time.Minute).WithPolling(time.Second).Should(Succeed())
				Expect(sriovIfaceName).NotTo(BeEmpty())

				updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(libnet.CheckMacAddress(updatedVMI, sriovIfaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest after migration")
			})
		})

		Context("memory hotplug", Serial, decorators.RequiresTwoSchedulableNodes, func() {
			BeforeEach(func() {
				updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
					WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
				}
				rolloutStrategy := pointer.P(v1.VMRolloutStrategyLiveUpdate)
				err := config.RegisterKubevirtConfigChange(
					config.WithWorkloadUpdateStrategy(updateStrategy),
					config.WithVMRolloutStrategy(rolloutStrategy),
				)
				Expect(err).ToNot(HaveOccurred())

				currentKv := libkubevirt.GetCurrentKv(virtClient)
				config.WaitForConfigToBePropagatedToComponent(
					"kubevirt.io=virt-controller",
					currentKv.ResourceVersion,
					config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
					time.Minute)
			})

			It("Should successfully reattach host-device", func() {
				const (
					initialGuestMemory = "1Gi"
					updatedGuestMemory = "3Gi"
					mac                = "de:ad:00:00:be:af"
				)
				vmi := libvmifact.NewAlpineWithTestTooling(
					libvmi.WithGuestMemory(initialGuestMemory),
					libvmi.WithInterface(libvmi.InterfaceWithMac(libvmi.InterfaceDeviceWithSRIOVBinding(claimName), mac)),
					libvmi.WithNetwork(libvmi.DRANetwork(claimName, claimName, "vf")),
					libvmi.WithResourceClaimTemplate(claimName, templateName),
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

				_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring the VMI has more available guest memory")
				initialGuestMemoryQuantity := resource.MustParse(initialGuestMemory)
				Eventually(func() int64 {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return vmi.Status.Memory.GuestCurrent.Value()
				}).
					WithTimeout(5 * time.Minute).
					WithPolling(5 * time.Second).
					Should(BeNumerically(">", initialGuestMemoryQuantity.Value()))

				By("Ensuring SR-IOV device was hotplugged to the VMI")
				Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).
						Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return vmi.Status.Interfaces
				}).
					WithTimeout(1 * time.Minute).
					WithPolling(5 * time.Second).
					Should(ContainElement(
						MatchFields(IgnoreExtras, Fields{
							"Name":       Equal(claimName),
							"InfoSource": ContainSubstring(vmispec.InfoSourceDomain),
							"MAC":        Equal(mac),
						}),
					))
			})
		})

	})

	Context("VMI connected to two DRA SR-IOV networks", func() {
		var (
			claim1       = "dra-sriov-claim-1"
			claim2       = "dra-sriov-claim-2"
			networkName1 = "dra-sriov-net-1"
			networkName2 = "dra-sriov-net-2"
			template1    = "single-vf-dra-sriov-net-1"
			template2    = "single-vf-dra-sriov-net-2"
			driverName   = "sriovnetwork.k8snetworkplumbingwg.io"
		)

		BeforeEach(func() {
			for _, netName := range []string{networkName1, networkName2} {
				err := libnet.CreateSRIOVNetworkWithDRA(
					context.Background(),
					testsuite.NamespaceTestDefault,
					netName,
					driverName,
					defaultVLAN,
				)
				Expect(err).NotTo(HaveOccurred(), "should create NAD and ResourceClaimTemplate")
			}

			DeferCleanup(func() {
				for _, tmpl := range []string{template1, template2} {
					err := libnet.DeleteResourceClaimTemplate(context.Background(), testsuite.NamespaceTestDefault, tmpl)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		It("should create a virtual machine with two DRA sriov interfaces referring different claims", func() {
			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(claim1)),
				libvmi.WithNetwork(libvmi.DRANetwork(claim1, claim1, "vf")),
				libvmi.WithResourceClaimTemplate(claim1, template1),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(claim2)),
				libvmi.WithNetwork(libvmi.DRANetwork(claim2, claim2, "vf")),
				libvmi.WithResourceClaimTemplate(claim2, template2),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())),
			)
			vmi.Spec.Domain.Devices.Interfaces[1].PciAddress = "0000:06:00.0"
			vmi.Spec.Domain.Devices.Interfaces[2].PciAddress = "0000:07:00.0"

			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi)

			Expect(checkDefaultInterfaceInPod(vmi)).To(Succeed())

			By("checking virtual machine instance has three interfaces")
			checkInterfacesInGuest(vmi, []string{"eth0", "eth1", "eth2"})

			Expect(pciAddressExistsInGuest(vmi, vmi.Spec.Domain.Devices.Interfaces[1].PciAddress)).To(Succeed())
			Expect(pciAddressExistsInGuest(vmi, vmi.Spec.Domain.Devices.Interfaces[2].PciAddress)).To(Succeed())
		})
	})

	Context("Connected to multiple DRA SR-IOV networks", func() {
		var (
			driverName = "sriovnetwork.k8snetworkplumbingwg.io"
			claims     = []string{"dra-claim-1", "dra-claim-2", "dra-claim-3", "dra-claim-4"}
			networks   = []string{"dra-net-1", "dra-net-2", "dra-net-3", "dra-net-4"}
			templates  = []string{"single-vf-dra-net-1", "single-vf-dra-net-2", "single-vf-dra-net-3", "single-vf-dra-net-4"}
		)

		BeforeEach(func() {
			for _, netName := range networks {
				err := libnet.CreateSRIOVNetworkWithDRA(
					context.Background(),
					testsuite.NamespaceTestDefault,
					netName,
					driverName,
					defaultVLAN,
				)
				Expect(err).NotTo(HaveOccurred(), "should create NAD and ResourceClaimTemplate")
			}

			DeferCleanup(func() {
				for _, tmpl := range templates {
					err := libnet.DeleteResourceClaimTemplate(context.Background(), testsuite.NamespaceTestDefault, tmpl)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		It("should correctly plug all the DRA interfaces based on the specified MAC and (guest) PCI addresses", func() {
			pciAddressTemplate := "0000:2%d:00.0"

			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(defaultCloudInitNetworkData())),
			)
			for i, claimName := range claims {
				libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(claimName))(vmi)
				libvmi.WithNetwork(libvmi.DRANetwork(claimName, claimName, "vf"))(vmi)
				libvmi.WithResourceClaimTemplate(claimName, templates[i])(vmi)
			}
			for i := range claims {
				secondaryInterfaceIdx := i + 1
				mac, err := libnet.GenerateRandomMac()
				Expect(err).NotTo(HaveOccurred())
				vmi.Spec.Domain.Devices.Interfaces[secondaryInterfaceIdx].MacAddress = mac.String()
				vmi.Spec.Domain.Devices.Interfaces[secondaryInterfaceIdx].PciAddress = fmt.Sprintf(pciAddressTemplate, secondaryInterfaceIdx)
			}

			vmi, err := createVMIAndWait(vmi)
			Expect(err).ToNot(HaveOccurred())

			for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
				if iface.SRIOV == nil {
					continue
				}
				guestInterfaceName, err := findIfaceByMAC(virtClient, vmi, iface.MacAddress, 5*time.Minute)
				Expect(err).ToNot(HaveOccurred())
				Expect(pciAddressExistsInGuestInterface(vmi, iface.PciAddress, guestInterfaceName)).To(Succeed())
			}
		})
	})

	Context("VMI connected to link-enabled DRA SR-IOV network", func() {
		var (
			networkNameLinked  = "dra-sriov-net-linked"
			templateNameLinked = "single-vf-dra-sriov-net-linked"
			driverName         = "sriovnetwork.k8snetworkplumbingwg.io"
			sriovNode          string
		)

		BeforeEach(func() {
			sriovNode = draSRIOVNodeName()
			Expect(sriovNode).NotTo(BeEmpty(), "could not find a schedulable node with sriov_capable=true label")

			err := libnet.CreateSRIOVNetworkWithDRA(
				context.Background(),
				testsuite.NamespaceTestDefault,
				networkNameLinked,
				driverName,
				defaultVLAN,
				withLinkState(),
			)
			Expect(err).NotTo(HaveOccurred(), "should create NAD and ResourceClaimTemplate")

			DeferCleanup(func() {
				err := libnet.DeleteResourceClaimTemplate(context.Background(), testsuite.NamespaceTestDefault, templateNameLinked)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should connect to another machine with DRA sriov interface over IP", func() {
			cidrA := "192.168.1.1/24"
			cidrB := "192.168.1.2/24"
			ipA, err := libnet.CidrToIP(cidrA)
			Expect(err).ToNot(HaveOccurred())
			ipB, err := libnet.CidrToIP(cidrB)
			Expect(err).ToNot(HaveOccurred())

			vmi1, err := createDRASRIOVVmiOnNode(sriovNode, "dra-claim-linked-1", templateNameLinked, cidrA)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(deleteVMI, vmi1)
			vmi2, err := createDRASRIOVVmiOnNode(sriovNode, "dra-claim-linked-2", templateNameLinked, cidrB)
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
				cidrVlaned1         = "192.168.0.1/24"
				networkVlanned1     = "dra-net-vlan-1"
				networkVlanned2     = "dra-net-vlan-2"
				networkNonVlanned   = "dra-net-no-vlan"
				templateVlanned1    = "single-vf-dra-net-vlan-1"
				templateVlanned2    = "single-vf-dra-net-vlan-2"
				templateNonVlanned  = "single-vf-dra-net-no-vlan"
				claimNameVlanned1   = "dra-claim-vlan-1"
				claimNameVlanned2   = "dra-claim-vlan-2"
				claimNameNonVlanned = "dra-claim-no-vlan"
				driverName          = "sriovnetwork.k8snetworkplumbingwg.io"
			)
			var ipVlaned1 string

			BeforeEach(func() {
				var err error
				ipVlaned1, err = libnet.CidrToIP(cidrVlaned1)
				Expect(err).ToNot(HaveOccurred())

				for _, netName := range []string{networkVlanned1, networkVlanned2, networkNonVlanned} {
					vlanID := specificVLAN
					if netName == networkNonVlanned {
						vlanID = defaultVLAN
					}
					err := libnet.CreateSRIOVNetworkWithDRA(
						context.Background(),
						testsuite.NamespaceTestDefault,
						netName,
						driverName,
						vlanID,
						withLinkState(),
					)
					Expect(err).NotTo(HaveOccurred(), "should create NAD and ResourceClaimTemplate")
				}

				DeferCleanup(func() {
					for _, tmpl := range []string{templateVlanned1, templateVlanned2, templateNonVlanned} {
						err := libnet.DeleteResourceClaimTemplate(context.Background(), testsuite.NamespaceTestDefault, tmpl)
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})

			It("should be able to ping between two VMIs with the same VLAN over DRA SRIOV network", func() {
				vlanedVMI1, err := createDRASRIOVVmiOnNode(sriovNode, claimNameVlanned1, templateVlanned1, cidrVlaned1)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vlanedVMI1)
				vlanedVMI2, err := createDRASRIOVVmiOnNode(sriovNode, claimNameVlanned2, templateVlanned2, "192.168.0.2/24")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vlanedVMI2)

				_, err = waitVMI(vlanedVMI1)
				Expect(err).NotTo(HaveOccurred())
				vlanedVMI2, err = waitVMI(vlanedVMI2)
				Expect(err).NotTo(HaveOccurred())

				By("pinging from vlanedVMI2 to vlanedVMI1 over vlan")
				Eventually(func() error {
					return libnet.PingFromVMConsole(vlanedVMI2, ipVlaned1)
				}, 15*time.Second, time.Second).ShouldNot(HaveOccurred())
			})

			It("should NOT be able to ping between Vlaned VMI and a non Vlaned VMI using DRA", func() {
				vlanedVMI, err := createDRASRIOVVmiOnNode(sriovNode, claimNameVlanned1, templateVlanned1, cidrVlaned1)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, vlanedVMI)
				nonVlanedVMI, err := createDRASRIOVVmiOnNode(sriovNode, claimNameNonVlanned, templateNonVlanned, "192.168.0.3/24")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(deleteVMI, nonVlanedVMI)

				_, err = waitVMI(vlanedVMI)
				Expect(err).NotTo(HaveOccurred())
				nonVlanedVMI, err = waitVMI(nonVlanedVMI)
				Expect(err).NotTo(HaveOccurred())

				By("pinging between nonVlanedVMI and the vlaned vmi")
				Eventually(func() error {
					return libnet.PingFromVMConsole(nonVlanedVMI, ipVlaned1)
				}, 15*time.Second, time.Second).Should(HaveOccurred())
			})
		})
	})
}))

// newDRASRIOVVmi creates a VMI with DRA-based SR-IOV networks
// claimNames: list of claim names to use
// templateName: the ResourceClaimTemplate name to reference (can be empty for pre-created claims)
func newDRASRIOVVmi(claimNames []string, templateName string, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	options := []libvmi.Option{
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	}

	for _, claimName := range claimNames {
		options = append(options,
			libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(claimName)),
			libvmi.WithNetwork(libvmi.DRANetwork(claimName, claimName, "vf")),
		)
		if templateName != "" {
			options = append(options, libvmi.WithResourceClaimTemplate(claimName, templateName))
		}
	}
	opts = append(options, opts...)
	return libvmifact.NewFedora(opts...)
}

// createDRASRIOVVmiOnNode creates a VMI on the specified node, connected to the specified DRA SR-IOV network.
func createDRASRIOVVmiOnNode(nodeName, claimName, templateName, cidr string) (*v1.VirtualMachineInstance, error) {
	mac, err := libnet.GenerateRandomMac()
	if err != nil {
		return nil, err
	}

	networkData := netcloudinit.CreateNetworkDataWithStaticIPsByMac("sriovnet1", mac.String(), cidr)
	vmi := newDRASRIOVVmi([]string{claimName}, templateName, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)))
	libvmi.WithNodeAffinityFor(nodeName)(vmi)
	const secondaryInterfaceIndex = 1
	vmi.Spec.Domain.Devices.Interfaces[secondaryInterfaceIndex].MacAddress = mac.String()

	virtCli := kubevirt.Client()
	vmi, err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return vmi, nil
}

func draSRIOVNodeName() string {
	nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
	for _, node := range nodes.Items {
		if val, ok := node.Labels["sriov_capable"]; ok && val == "true" {
			return node.Name
		}
	}
	return ""
}
