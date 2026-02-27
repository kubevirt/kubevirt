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
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Premigration Hook Server", func() {
	Context("vGPU Dedicated Hook", func() {
		It("should update vGPU mdev uuid according to target node config", func() {
			By("creating a VMI and libvirt domain with a single vGPU")
			targetUUID := "05b59010-d19c-47d2-9477-33b4579edc90"

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "kubevirt",
					Namespace:   "testns",
					Annotations: map[string]string{},
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
							"bb4a98d8-60c1-40c6-b39b-866b1e82bd8c",
							"ua-gpu-gpu1",
							0x00,
						),
					},
				},
			}

			By("annotating the vmi with the target mdev uuid")
			vmi.Annotations[TargetMdevUUIDAnnotation] = targetUUID

			Expect(VGPUDedicatedHook(vmi, &domain)).NotTo(HaveOccurred(), "failed to modify domain")

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
			targetUUID := "3ad93aea-4e81-49a9-b1af-9d020373a357"

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "kubevirt",
					Namespace:   "testns",
					Annotations: map[string]string{},
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
							"bb4a98d8-60c1-40c6-b39b-866b1e82bd8c",
							"ua-gpu-gpu1",
							0x00,
						),
						newMdevHostdev(
							"19dcdf19-0ef0-496b-be3b-c591109ca572",
							"ua-gpu-gpu2",
							0x01,
						),
					},
				},
			}

			By("annotating the vmi with the target mdev uuid")
			vmi.Annotations[TargetMdevUUIDAnnotation] = targetUUID

			Expect(VGPUDedicatedHook(vmi, &domain)).To(MatchError("the migrating vmi can only have one vGPU"))
		})
	})
})

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
