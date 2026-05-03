/*
 * This file is part of the KubeVirt project
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

package sriov_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"kubevirt.io/kubevirt/pkg/dra/metadata"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"

	netsriov "kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
)

const (
	netname1 = "net1"
	netname2 = "net2"

	sriovDRADriverName       = "sriovnetwork.k8snetworkplumbingwg.io"
	sriovDRAMetadataFileName = sriovDRADriverName + "-metadata.json"
)

var _ = Describe("SRIOV HostDevice", func() {
	Context("creation", func() {
		It("creates no device given no interfaces", func() {
			vmi := &v1.VirtualMachineInstance{}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("creates no device given no SRIOV interfaces", func() {
			iface := v1.Interface{}
			iface.Masquerade = &v1.InterfaceMasquerade{}
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("creates no device given SRIOV interface that has no status", func() {
			iface := newSRIOVInterface("test")
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("creates no device given SRIOV interface without multus info source", func() {
			iface := newSRIOVInterface("test")
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}
			vmi.Status = v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
					Name: "test",
				}},
			}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("fails to create device given no available host PCI", func() {
			iface := newSRIOVInterface("test")
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}
			vmi.Status = v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
					Name:       "test",
					InfoSource: vmispec.InfoSourceMultusStatus,
				}},
			}

			_, err := sriov.CreateHostDevices(vmi)

			Expect(err).To(HaveOccurred())
		})

		It("fails to create a device given bad host PCI address", func() {
			ifaces := []v1.Interface{newSRIOVInterface("net1")}
			pool := newPCIAddressPoolStub("0bad0pci0address0")

			_, err := sriov.CreateHostDevicesFromIfacesAndPool(ifaces, pool)

			Expect(err).To(HaveOccurred())
		})

		It("fails to create a device given bad guest PCI address", func() {
			iface := newSRIOVInterface("net1")
			iface.PciAddress = "0bad0pci0address0"
			pool := newPCIAddressPoolStub("0000:81:01.0")

			_, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			Expect(err).To(HaveOccurred())
		})

		It("fails to create a device given two interfaces but only one host PCI", func() {
			iface1 := newSRIOVInterface(netname1)
			iface2 := newSRIOVInterface(netname1)
			pool := newPCIAddressPoolStub("0000:81:01.0")

			_, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

			Expect(err).To(HaveOccurred())
		})

		It("creates 2 devices that are connected to the same network", func() {
			iface1 := newSRIOVInterface(netname1)
			iface2 := newSRIOVInterface(netname1)
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:01.1")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x1"}
			expectHostDevice2 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("creates 2 devices that are connected to different networks", func() {
			iface1 := newSRIOVInterface(netname1)
			iface2 := newSRIOVInterface(netname2)
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x02", Function: "0x0"}
			expectHostDevice2 := api.HostDevice{
				Alias:   newSRIOVAlias(netname2),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("creates 1 device that includes guest PCI addresses", func() {
			iface := newSRIOVInterface(netname1)
			iface.PciAddress = "0000:01:01.0"
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			guestPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x01", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    api.HostDevicePCI,
				Managed: "no",
				Address: &guestPCIAddress1,
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})

		DescribeTable("create two devices with custom guest PCI address",
			func(iface1, iface2 v1.Interface) {
				var expectedGuestPCIAddress1 *api.Address
				var expectedGuestPCIAddress2 *api.Address

				var err error
				if iface1.PciAddress != "" {
					expectedGuestPCIAddress1, err = device.NewPciAddressField(iface1.PciAddress)
					Expect(err).NotTo(HaveOccurred())
				}

				if iface2.PciAddress != "" {
					expectedGuestPCIAddress2, err = device.NewPciAddressField(iface2.PciAddress)
					Expect(err).NotTo(HaveOccurred())
				}

				pool := newPCIAddressPoolStub("0000:81:00.0", "0000:81:01.0")
				hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x00", Function: "0x0"}
				hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}

				devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)
				Expect(err).NotTo(HaveOccurred())

				expectHostDevice1 := api.HostDevice{
					Alias:   newSRIOVAlias(netname1),
					Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
					Address: expectedGuestPCIAddress1,
					Type:    api.HostDevicePCI,
					Managed: "no",
				}

				expectHostDevice2 := api.HostDevice{
					Alias:   newSRIOVAlias(netname2),
					Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
					Address: expectedGuestPCIAddress2,
					Type:    api.HostDevicePCI,
					Managed: "no",
				}

				Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
			},
			Entry("both interfaces have a custom guest PCI address",
				newSRIOVInterfaceWithPCIAddress(netname1, "0000:20:00.0"),
				newSRIOVInterfaceWithPCIAddress(netname2, "0000:20:01.0"),
			),
			Entry("only the first interface has a custom guest PCI address",
				newSRIOVInterfaceWithPCIAddress(netname1, "0000:20:00.0"),
				newSRIOVInterface(netname2),
			),
			Entry("only the second interface has a custom guest PCI address",
				newSRIOVInterface(netname1),
				newSRIOVInterfaceWithPCIAddress(netname2, "0000:20:01.0"),
			),
		)

		It("creates 1 device that includes boot-order", func() {
			iface := newSRIOVInterface(netname1)
			val := uint(1)
			iface.BootOrder = &val
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:     newSRIOVAlias(netname1),
				Source:    api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:      api.HostDevicePCI,
				Managed:   "no",
				BootOrder: &api.BootOrder{Order: *iface.BootOrder},
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})

		DescribeTable("create two devices with custom boot-order",
			func(iface1, iface2 v1.Interface) {
				var expectedBootOrder1 *api.BootOrder
				var expectedBootOrder2 *api.BootOrder

				if iface1.BootOrder != nil {
					expectedBootOrder1 = &api.BootOrder{Order: *iface1.BootOrder}
				}

				if iface2.BootOrder != nil {
					expectedBootOrder2 = &api.BootOrder{Order: *iface2.BootOrder}
				}

				pool := newPCIAddressPoolStub("0000:81:00.0", "0000:81:01.0")
				hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x00", Function: "0x0"}
				hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}

				devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)
				Expect(err).NotTo(HaveOccurred())

				expectHostDevice1 := api.HostDevice{
					Alias:     newSRIOVAlias(netname1),
					Source:    api.HostDeviceSource{Address: &hostPCIAddress1},
					Type:      api.HostDevicePCI,
					Managed:   "no",
					BootOrder: expectedBootOrder1,
				}

				expectHostDevice2 := api.HostDevice{
					Alias:     newSRIOVAlias(netname2),
					Source:    api.HostDeviceSource{Address: &hostPCIAddress2},
					Type:      api.HostDevicePCI,
					Managed:   "no",
					BootOrder: expectedBootOrder2,
				}

				Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
			},
			Entry("both interfaces have a custom bootOrder",
				newSRIOVInterfaceWithBootOrder(netname1, 1),
				newSRIOVInterfaceWithBootOrder(netname2, 2),
			),
			Entry("only the first interface has a custom bootOrder",
				newSRIOVInterfaceWithBootOrder(netname1, 1),
				newSRIOVInterface(netname2),
			),
			Entry("only the second interface has a custom bootOrder",
				newSRIOVInterface(netname1),
				newSRIOVInterfaceWithBootOrder(netname2, 2),
			),
		)
	})

	Context("DRA SR-IOV creation", func() {
		newDRAVMI := func(claimRefName, requestName string) *v1.VirtualMachineInstance {
			iface := newSRIOVInterface(netname1)
			iface.PciAddress = "0000:20:00.0"

			return &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Interfaces: []v1.Interface{iface},
						},
					},
					Networks: []v1.Network{{
						Name: netname1,
						NetworkSource: v1.NetworkSource{
							ResourceClaim: &v1.ClaimRequest{
								ClaimName:   ptr.To(claimRefName),
								RequestName: ptr.To(requestName),
							},
						},
					}},
				},
			}
		}

		It("uses metadata file path for direct ResourceClaim", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("claim-ref", "vf")
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name:              "claim-ref",
				ResourceClaimName: ptr.To("manual-vf-claim"),
			}}
			Expect(writeMetadataFile(tempDir, "resourceclaims", "manual-vf-claim", "vf", "0000:65:0a.3")).To(Succeed())

			devices, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(devices).To(HaveLen(1))

			expectedHostAddr, err := device.NewPciAddressField("0000:65:0a.3")
			Expect(err).ToNot(HaveOccurred())
			expectedGuestAddr, err := device.NewPciAddressField("0000:20:00.0")
			Expect(err).ToNot(HaveOccurred())
			expectedHostDevice := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: expectedHostAddr},
				Type:    api.HostDevicePCI,
				Managed: "no",
				Address: expectedGuestAddr,
			}
			Expect(devices).To(Equal([]api.HostDevice{expectedHostDevice}))
		})

		It("uses metadata file path for ResourceClaimTemplate-backed pod claim", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("generated-claim-ref", "vf")
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name: "generated-claim-ref",
			}}
			Expect(writeMetadataFile(tempDir, "resourceclaimtemplates", "generated-claim-ref", "vf", "0000:65:0a.4")).To(Succeed())

			devices, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(devices).To(HaveLen(1))

			expectedHostAddr, err := device.NewPciAddressField("0000:65:0a.4")
			Expect(err).ToNot(HaveOccurred())
			Expect(devices[0].Source.Address).To(Equal(expectedHostAddr))
		})

		It("propagates boot order from DRA SR-IOV interface", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("claim-ref", "vf")
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name:              "claim-ref",
				ResourceClaimName: ptr.To("manual-vf-claim"),
			}}
			bootOrder := uint(1)
			vmi.Spec.Domain.Devices.Interfaces[0].BootOrder = &bootOrder
			Expect(writeMetadataFile(tempDir, "resourceclaims", "manual-vf-claim", "vf", "0000:65:0a.5")).To(Succeed())

			devices, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(devices).To(HaveLen(1))
			Expect(devices[0].BootOrder).To(Equal(&api.BootOrder{Order: bootOrder}))
		})

		It("does not set guest PCI address when DRA SR-IOV interface has no guest PCI address", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("claim-ref", "vf")
			vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = ""
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name:              "claim-ref",
				ResourceClaimName: ptr.To("manual-vf-claim"),
			}}
			Expect(writeMetadataFile(tempDir, "resourceclaims", "manual-vf-claim", "vf", "0000:65:0a.6")).To(Succeed())

			devices, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(devices).To(HaveLen(1))
			Expect(devices[0].Address).To(BeNil())
		})

		It("fails when DRA SR-IOV interface has malformed guest PCI address", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("claim-ref", "vf")
			vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = "not-a-pci-address"
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name:              "claim-ref",
				ResourceClaimName: ptr.To("manual-vf-claim"),
			}}
			Expect(writeMetadataFile(tempDir, "resourceclaims", "manual-vf-claim", "vf", "0000:65:0a.7")).To(Succeed())

			_, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to interpret the guest PCI address for interface net1"))
		})

		It("fails when metadata file is missing for direct ResourceClaim", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("claim-ref", "vf")
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name:              "claim-ref",
				ResourceClaimName: ptr.To("manual-vf-claim"),
			}}

			_, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve PCI address for SR-IOV DRA interface net1"))
			Expect(err.Error()).To(ContainSubstring("failed to read metadata for claim \"manual-vf-claim\" request \"vf\""))
		})

		It("fails when metadata file contains malformed JSON", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("claim-ref", "vf")
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name:              "claim-ref",
				ResourceClaimName: ptr.To("manual-vf-claim"),
			}}
			Expect(writeRawMetadataFile(
				tempDir,
				"resourceclaims",
				"manual-vf-claim",
				"vf",
				`{"apiVersion":"metadata.resource.k8s.io/v1alpha1","kind":"DeviceMetadata","metadata":{"name":"manual-vf-claim"},"requests":[`,
			)).To(Succeed())

			_, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve PCI address for SR-IOV DRA interface net1"))
			Expect(err.Error()).To(ContainSubstring("read object from metadata stream"))
		})

		It("fails when metadata has invalid host pciBusID", func() {
			tempDir := GinkgoT().TempDir()

			vmi := newDRAVMI("claim-ref", "vf")
			vmi.Spec.ResourceClaims = []k8sv1.PodResourceClaim{{
				Name:              "claim-ref",
				ResourceClaimName: ptr.To("manual-vf-claim"),
			}}
			Expect(writeMetadataFile(tempDir, "resourceclaims", "manual-vf-claim", "vf", "not-a-pci-address")).To(Succeed())

			_, err := sriov.CreateDRAHostDevices(vmi, tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create PCI address for SR-IOV DRA interface net1"))
		})

	})

	Context("safe detachment", func() {
		hostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias(netsriov.SRIOVAliasPrefix + "net1")}

		It("ignores an empty list of devices", func() {
			domainSpec := newDomainSpec()

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(Succeed())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails to register a callback", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(true, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails to detach device", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{fail: true}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails on timeout due to no detach event", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
		})

		It("fails due to a missing event from a sriov device", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			c.sendEvent("non-sriov")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 10*time.Millisecond)).To(HaveOccurred())
			Expect(c.EventChannel()).To(BeEmpty())
		})

		// Failure to deregister the callback only emits a logging error.
		It("succeeds to wait for a detached device and fails to deregister a callback", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, true)
			c.sendEvent(api.UserAliasPrefix + hostDevice.Alias.GetName())
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 10*time.Millisecond)).To(Succeed())
		})

		It("succeeds detaching 2 sriov devices", func() {
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias(netsriov.SRIOVAliasPrefix + "net2")}
			domainSpec := newDomainSpec(hostDevice, hostDevice2)

			c := newCallbackerStub(false, false)
			c.sendEvent(api.UserAliasPrefix + hostDevice.Alias.GetName())
			c.sendEvent(api.UserAliasPrefix + hostDevice2.Alias.GetName())
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 10*time.Millisecond)).To(Succeed())
		})
	})
})

func newDomainSpec(hostDevices ...api.HostDevice) *api.DomainSpec {
	domainSpec := &api.DomainSpec{}
	domainSpec.Devices.HostDevices = append(domainSpec.Devices.HostDevices, hostDevices...)
	return domainSpec
}

func newSRIOVAlias(netName string) *api.Alias {
	return api.NewUserDefinedAlias(netsriov.SRIOVAliasPrefix + netName)
}

func newSRIOVInterfaceWithPCIAddress(name, customPCIAddress string) v1.Interface {
	iface := newSRIOVInterface(name)
	iface.PciAddress = customPCIAddress

	return iface
}

func newSRIOVInterfaceWithBootOrder(name string, bootOrder uint) v1.Interface {
	iface := newSRIOVInterface(name)
	iface.BootOrder = &bootOrder

	return iface
}

func writeRawMetadataFile(basePath, claimSubdir, claimName, requestName, metadataJSON string) error {
	dir := filepath.Join(basePath, claimSubdir, claimName, requestName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, sriovDRAMetadataFileName), []byte(metadataJSON), 0644)
}

func writeMetadataFile(basePath, claimSubdir, claimName, requestName, pciAddress string) error {
	md := metadata.DeviceMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metadata.APIVersionV1Alpha1,
			Kind:       "DeviceMetadata",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: claimName,
		},
		Requests: []metadata.DeviceMetadataRequest{{
			Name: requestName,
			Devices: []metadata.Device{{
				Driver: sriovDRADriverName,
				Pool:   "pool0",
				Name:   "dev0",
				Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
					metadata.PCIBusIDAttribute: {StringValue: ptr.To(pciAddress)},
				},
			}},
		}},
	}
	data, err := json.Marshal(md)
	if err != nil {
		return err
	}
	return writeRawMetadataFile(basePath, claimSubdir, claimName, requestName, string(data))
}

type stubPCIAddressPool struct {
	pciAddresses []string
}

func newPCIAddressPoolStub(PCIAddresses ...string) *stubPCIAddressPool {
	return &stubPCIAddressPool{PCIAddresses}
}

func (p *stubPCIAddressPool) Pop(_ string) (string, error) {
	if len(p.pciAddresses) == 0 {
		return "", fmt.Errorf("pool is empty")
	}

	address := p.pciAddresses[0]
	p.pciAddresses = p.pciAddresses[1:]

	return address, nil
}

type deviceDetacherStub struct {
	fail bool
}

func (d deviceDetacherStub) DetachDeviceFlags(data string, flags libvirt.DomainDeviceModifyFlags) error {
	if d.fail {
		return fmt.Errorf("detach device error")
	}
	return nil
}

func newCallbackerStub(failRegister, failDeregister bool) *callbackerStub {
	return &callbackerStub{
		failRegister:   failRegister,
		failDeregister: failDeregister,
		eventChan:      make(chan interface{}, hostdevice.MaxConcurrentHotPlugDevicesEvents),
	}
}

type callbackerStub struct {
	failRegister   bool
	failDeregister bool
	eventChan      chan interface{}
}

func (c *callbackerStub) Register() error {
	if c.failRegister {
		return fmt.Errorf("register error")
	}
	return nil
}

func (c *callbackerStub) Deregister() error {
	if c.failDeregister {
		return fmt.Errorf("deregister error")
	}
	return nil
}

func (c *callbackerStub) EventChannel() <-chan interface{} {
	return c.eventChan
}

func (c *callbackerStub) sendEvent(data string) {
	c.eventChan <- data
}
