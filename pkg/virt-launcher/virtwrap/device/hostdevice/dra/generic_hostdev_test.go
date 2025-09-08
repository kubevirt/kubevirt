package dra

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CreateDRAHostDevices", func() {
	Context("when the VMI has no host devices with DRA", func() {
		It("should return an empty slice without error", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
				Spec:       v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}},
			}
			hostDevs, err := CreateDRAHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevs).To(BeEmpty())
		})
	})

	Context("when the VMI has a PCI host device allocated through DRA", func() {
		It("should create a PCI HostDevice with correct attributes", func() {
			pci := "0000:03:00.1"
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							HostDevices: []v1.HostDevice{{
								Name:         "hd1",
								ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim1"), RequestName: ptr.To("req1")},
							}},
						},
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					DeviceStatus: &v1.DeviceStatus{
						HostDeviceStatuses: []v1.DeviceStatusInfo{{
							Name: "hd1",
							DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
								ResourceClaimName: ptr.To("claim1"),
								Name:              ptr.To("device1"),
								Attributes:        &v1.DeviceAttribute{PCIAddress: &pci},
							},
						}},
					},
				},
			}

			hostDevs, err := CreateDRAHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevs).To(HaveLen(1))

			dev := hostDevs[0]
			Expect(dev.Type).To(Equal(api.HostDevicePCI))
			Expect(dev.Managed).To(Equal("no"))
			Expect(dev.Alias.GetName()).To(Equal(DRAHostDeviceAliasPrefix + "hd1"))
			Expect(dev.Source.Address.Type).To(Equal(api.AddressPCI))
		})
	})

	Context("when the VMI has an MDEV host device allocated through DRA", func() {
		It("should create an MDEV HostDevice", func() {
			uuid := "abcd1234-1111-2222-3333-444455556666"
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							HostDevices: []v1.HostDevice{{
								Name:         "vhd1",
								ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim1"), RequestName: ptr.To("req1")},
							}},
						},
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					DeviceStatus: &v1.DeviceStatus{
						HostDeviceStatuses: []v1.DeviceStatusInfo{{
							Name: "vhd1",
							DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
								ResourceClaimName: ptr.To("claim1"),
								Name:              ptr.To("device1"),
								Attributes:        &v1.DeviceAttribute{MDevUUID: &uuid},
							},
						}},
					},
				},
			}

			hostDevs, err := CreateDRAHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevs).To(HaveLen(1))
			dev := hostDevs[0]
			Expect(dev.Type).To(Equal(api.HostDeviceMDev))
			Expect(dev.Alias.GetName()).To(Equal(DRAHostDeviceAliasPrefix + "vhd1"))
			Expect(dev.Source.Address.UUID).To(Equal(uuid))
		})
	})

	Context("validation mismatch", func() {
		It("should error when counts differ between spec and status", func() {
			pci := "0000:03:00.1"
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							HostDevices: []v1.HostDevice{{
								Name:         "hd1",
								ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim1"), RequestName: ptr.To("req1")},
							}, {
								Name:         "hd2",
								ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim2"), RequestName: ptr.To("req2")},
							}},
						},
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					DeviceStatus: &v1.DeviceStatus{
						HostDeviceStatuses: []v1.DeviceStatusInfo{{
							Name: "hd1",
							DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
								ResourceClaimName: ptr.To("claim1"),
								Name:              ptr.To("device1"),
								Attributes:        &v1.DeviceAttribute{PCIAddress: &pci},
							},
						}},
					},
				},
			}
			hostDevs, err := CreateDRAHostDevices(vmi)
			Expect(err).To(HaveOccurred())
			Expect(hostDevs).To(BeNil())
		})
	})
})
