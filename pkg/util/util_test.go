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
	})
})
