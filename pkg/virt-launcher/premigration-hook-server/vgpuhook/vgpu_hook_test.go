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
 */

package vgpuhook

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

const (
	sourceUUID  = "bb4a98d8-60c1-40c6-b39b-866b1e82bd8c"
	sourceUUID2 = "19dcdf19-0ef0-496b-be3b-c591109ca572"
	targetUUID  = "05b59010-d19c-47d2-9477-33b4579edc90"
)

var _ = Describe("Premigration Hook Server", func() {
	Context("vGPU Dedicated Hook", func() {
		It("should update vGPU mdev uuid according to target node config", func() {
			By("creating a VMI and libvirt domain with a single vGPU")

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "testns",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								DeviceName: "nvidia.com/test",
								Name:       "gpu1",
							}},
						},
					},
				},
			}
			domain := libvirtxml.Domain{
				Type: "kvm",
				Name: "kubevirt",
				Devices: &libvirtxml.DomainDeviceList{
					Hostdevs: []libvirtxml.DomainHostdev{
						newMdevHostdev(
							sourceUUID,
							"ua-gpu-gpu1",
							0x00,
						),
					},
				},
			}
			c := &convertertypes.ConverterContext{
				GPUHostDevices: []api.HostDevice{
					newAPIHostDeviceMDev(targetUUID, "gpu1"),
				},
			}

			Expect(VGPULiveMigration(c, vmi, &domain)).NotTo(HaveOccurred(), "failed to modify domain")

			expectedDomain := libvirtxml.Domain{
				Type: "kvm",
				Name: "kubevirt",
				Devices: &libvirtxml.DomainDeviceList{
					Hostdevs: []libvirtxml.DomainHostdev{
						newMdevHostdev(
							targetUUID,
							"ua-gpu-gpu1",
							0x00,
						),
					},
				},
			}

			Expect(domain).To(Equal(expectedDomain))
		})

		It("should fail if there is more than 1 vGPU", func() {
			By("creating a VMI and libvirt domain with 2 vGPUs")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "testns",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{
									DeviceName: "nvidia.com/test",
									Name:       "gpu1",
								},
								{
									DeviceName: "nvidia.com/test",
									Name:       "gpu2",
								},
							},
						},
					},
				},
			}
			domain := libvirtxml.Domain{
				Type: "kvm",
				Name: "kubevirt",
				Devices: &libvirtxml.DomainDeviceList{
					Hostdevs: []libvirtxml.DomainHostdev{
						newMdevHostdev(
							sourceUUID,
							"ua-gpu-gpu1",
							0x00,
						),
						newMdevHostdev(
							sourceUUID2,
							"ua-gpu-gpu2",
							0x01,
						),
					},
				},
			}
			c := &convertertypes.ConverterContext{
				GPUHostDevices: []api.HostDevice{
					newAPIHostDeviceMDev(sourceUUID, "gpu1"),
					newAPIHostDeviceMDev(sourceUUID2, "gpu2"),
				},
			}

			Expect(VGPULiveMigration(c, vmi, &domain)).To(MatchError("the migrating vmi should only have one vGPU"))
		})

		It("should fail if GPU is not an mdev vGPU", func() {
			By("creating a VMI and libvirt domain with 1 PCI GPU (no mdev)")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "testns",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{
									DeviceName: "nvidia.com/test",
									Name:       "gpu1",
								},
							},
						},
					},
				},
			}
			domain := libvirtxml.Domain{
				Type: "kvm",
				Name: "kubevirt",
				Devices: &libvirtxml.DomainDeviceList{
					Hostdevs: []libvirtxml.DomainHostdev{
						newPCIHostdev(
							"ua-gpu-gpu1",
							0x00,
						),
					},
				},
			}
			c := &convertertypes.ConverterContext{
				GPUHostDevices: []api.HostDevice{
					newAPIHostDevicePCI("gpu1"),
				},
			}

			Expect(VGPULiveMigration(c, vmi, &domain)).To(MatchError("unsupporting gpu type for migration: " + api.AddressPCI))
		})
	})
})

func newAPIHostDeviceMDev(uuid, name string) api.HostDevice {
	return api.HostDevice{
		Type: api.HostDeviceMDev,
		Source: api.HostDeviceSource{
			Address: &api.Address{
				UUID: uuid,
				Type: api.HostDeviceMDev,
			},
		},
		Alias: api.NewUserDefinedAlias("gpu-" + name),
	}
}

func newAPIHostDevicePCI(name string) api.HostDevice {
	return api.HostDevice{
		Type: api.HostDevicePCI,
		Source: api.HostDeviceSource{
			Address: &api.Address{
				Domain:   "0x0000",
				Bus:      "0x01",
				Slot:     "0x00",
				Function: "0x0",
				Type:     api.AddressPCI,
			},
		},
		Alias: api.NewUserDefinedAlias("gpu-" + name),
	}
}

func newPCIHostdev(alias string, slot uint) libvirtxml.DomainHostdev {
	domain := uint(0x0000)
	bus := uint(0x09)
	function := uint(0x0)
	sourceDomain := uint(0x0000)
	sourceBus := uint(0x01)
	sourceSlot := uint(0x00)
	sourceFunction := uint(0x0)

	return libvirtxml.DomainHostdev{
		Managed: "yes",
		SubsysPCI: &libvirtxml.DomainHostdevSubsysPCI{
			Source: &libvirtxml.DomainHostdevSubsysPCISource{
				Address: &libvirtxml.DomainAddressPCI{
					Domain:   &sourceDomain,
					Bus:      &sourceBus,
					Slot:     &sourceSlot,
					Function: &sourceFunction,
				},
			},
		},
		Alias: &libvirtxml.DomainAlias{
			Name: alias,
		},
		Address: &libvirtxml.DomainAddress{
			PCI: &libvirtxml.DomainAddressPCI{
				Domain:   &domain,
				Bus:      &bus,
				Slot:     &slot,
				Function: &function,
			},
		},
	}
}

func newMdevHostdev(uuid, alias string, slot uint) libvirtxml.DomainHostdev {
	domain := uint(0x0000)
	bus := uint(0x09)
	function := uint(0x0)

	return libvirtxml.DomainHostdev{
		Managed: "no",
		SubsysMDev: &libvirtxml.DomainHostdevSubsysMDev{
			Model: "vfio-pci",
			Source: &libvirtxml.DomainHostdevSubsysMDevSource{
				Address: &libvirtxml.DomainAddressMDev{
					UUID: uuid,
				},
			},
		},
		Alias: &libvirtxml.DomainAlias{
			Name: alias,
		},
		Address: &libvirtxml.DomainAddress{
			PCI: &libvirtxml.DomainAddressPCI{
				Domain:   &domain,
				Bus:      &bus,
				Slot:     &slot,
				Function: &function,
			},
		},
	}
}
