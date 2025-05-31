package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("GPU VMI Predicates", func() {
	It("should detect VMI with GPU device plugins", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name:       "gpu1",
								DeviceName: "nvidia.com/gpu",
							},
						},
					},
				},
			},
		}
		Expect(IsGPUVMI(vmi)).To(BeTrue())
		Expect(IsGPUVMIDevicePlugins(vmi)).To(BeTrue())
		Expect(IsGPUVMIDRA(vmi)).To(BeFalse())
	})

	It("should detect VMI with GPU DRA", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name: "gpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   pointer.P("gpu-claim"),
									RequestName: pointer.P("gpu-request"),
								},
							},
						},
					},
				},
			},
		}
		Expect(IsGPUVMI(vmi)).To(BeTrue())
		Expect(IsGPUVMIDevicePlugins(vmi)).To(BeFalse())
		Expect(IsGPUVMIDRA(vmi)).To(BeTrue())
	})

	It("should detect VMI with mixed GPU device plugins and DRA", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name:       "gpu1",
								DeviceName: "nvidia.com/gpu",
							},
							{
								Name: "gpu2",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   pointer.P("gpu-claim"),
									RequestName: pointer.P("gpu-request"),
								},
							},
						},
					},
				},
			},
		}
		Expect(IsGPUVMI(vmi)).To(BeTrue())
		Expect(IsGPUVMIDevicePlugins(vmi)).To(BeTrue())
		Expect(IsGPUVMIDRA(vmi)).To(BeTrue())
	})

	It("should not detect GPU VMI when no GPUs are specified", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{},
				},
			},
		}
		Expect(IsGPUVMI(vmi)).To(BeFalse())
		Expect(IsGPUVMIDevicePlugins(vmi)).To(BeFalse())
		Expect(IsGPUVMIDRA(vmi)).To(BeFalse())
	})
})

var _ = Describe("Host Device VMI Predicates", func() {
	It("should detect VMI with host device plugins", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name:       "hostdev1",
								DeviceName: "vendor.com/device",
							},
						},
					},
				},
			},
		}
		Expect(IsHostDevVMI(vmi)).To(BeTrue())
		Expect(IsHostDevVMIDevicePlugins(vmi)).To(BeTrue())
		Expect(IsHostDevVMIDRA(vmi)).To(BeFalse())
	})

	It("should detect VMI with host device DRA", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name: "hostdev1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   pointer.P("hostdev-claim"),
									RequestName: pointer.P("hostdev-request"),
								},
							},
						},
					},
				},
			},
		}
		Expect(IsHostDevVMI(vmi)).To(BeTrue())
		Expect(IsHostDevVMIDevicePlugins(vmi)).To(BeFalse())
		Expect(IsHostDevVMIDRA(vmi)).To(BeTrue())
	})

	It("should detect VMI with mixed host device plugins and DRA", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name:       "hostdev1",
								DeviceName: "vendor.com/device",
							},
							{
								Name: "hostdev2",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   pointer.P("hostdev-claim"),
									RequestName: pointer.P("hostdev-request"),
								},
							},
						},
					},
				},
			},
		}
		Expect(IsHostDevVMI(vmi)).To(BeTrue())
		Expect(IsHostDevVMIDevicePlugins(vmi)).To(BeTrue())
		Expect(IsHostDevVMIDRA(vmi)).To(BeTrue())
	})

	It("should not detect host device VMI when no host devices are specified", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{},
				},
			},
		}
		Expect(IsHostDevVMI(vmi)).To(BeFalse())
		Expect(IsHostDevVMIDevicePlugins(vmi)).To(BeFalse())
		Expect(IsHostDevVMIDRA(vmi)).To(BeFalse())
	})
})
