package dra

/*
These unit tests verify the behaviour of CreateDRAGPUHostDevices,
which converts the DRA-related information stored on a VMI into
libvirt HostDevice definitions that virt-launcher will add to the
libvirt domain.
*/

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CreateDRAGPUHostDevices", func() {
	Context("when the VMI has no GPUs with DRA", func() {
		It("should return an empty slice without error", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(BeEmpty())
		})
	})

	Context("when the VMI has a physical GPU (PCI) allocated through DRA", func() {
		It("should create exactly one PCI host device", func() {
			pciAddr := "0000:02:00.0"
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name: "gpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							}},
						},
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					DeviceStatus: &v1.DeviceStatus{
						GPUStatuses: []v1.DeviceStatusInfo{{
							Name: "gpu1",
							DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
								ResourceClaimName: ptr.To("claim1"),
								Name:              ptr.To("device1"),
								Attributes: &v1.DeviceAttribute{
									PCIAddress: &pciAddr,
								},
							},
						}},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(1))

			dev := hostDevices[0]
			Expect(dev.Type).To(Equal(api.HostDevicePCI))
			Expect(dev.Managed).To(Equal("no"))
			Expect(dev.Alias).ToNot(BeNil())
			Expect(dev.Alias.GetName()).To(Equal(AliasPrefix + "gpu1"))
			Expect(dev.Source.Address).ToNot(BeNil())
			Expect(dev.Source.Address.Type).To(Equal(api.AddressPCI))
		})
	})

	Context("when the VMI has a virtual GPU (mdev) allocated through DRA", func() {
		It("should create exactly one mdev host device with display enabled", func() {
			uuid := "123e4567-e89b-12d3-a456-426614174000"
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name: "vgpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							}},
						},
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					DeviceStatus: &v1.DeviceStatus{
						GPUStatuses: []v1.DeviceStatusInfo{{
							Name: "vgpu1",
							DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
								ResourceClaimName: ptr.To("claim1"),
								Name:              ptr.To("device1"),
								Attributes: &v1.DeviceAttribute{
									MDevUUID: &uuid,
								},
							},
						}},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(1))

			dev := hostDevices[0]
			Expect(dev.Type).To(Equal(api.HostDeviceMDev))
			Expect(dev.Display).To(Equal("on"))
			Expect(dev.RamFB).To(Equal("on"))
			Expect(dev.Alias.GetName()).To(Equal(AliasPrefix + "vgpu1"))
			Expect(dev.Source.Address).ToNot(BeNil())
			Expect(dev.Source.Address.UUID).To(Equal(uuid))
		})
	})

	Context("validation errors", func() {
		It("should return an error when the number of host devices does not match DRA GPUs in the spec", func() {
			pciAddr := "0000:02:00.0"
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{
									Name: "gpu1",
									ClaimRequest: &v1.ClaimRequest{
										ClaimName:   ptr.To("claim1"),
										RequestName: ptr.To("req1"),
									},
								},
								{
									Name: "gpu2",
									ClaimRequest: &v1.ClaimRequest{
										ClaimName:   ptr.To("claim2"),
										RequestName: ptr.To("req2"),
									},
								},
							},
						},
					},
				},
				Status: v1.VirtualMachineInstanceStatus{
					DeviceStatus: &v1.DeviceStatus{
						GPUStatuses: []v1.DeviceStatusInfo{{
							Name: "gpu1",
							DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
								ResourceClaimName: ptr.To("claim1"),
								Name:              ptr.To("device1"),
								Attributes: &v1.DeviceAttribute{
									PCIAddress: &pciAddr,
								},
							},
						}},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).To(HaveOccurred())
			Expect(hostDevices).To(BeNil())
		})
	})
})
