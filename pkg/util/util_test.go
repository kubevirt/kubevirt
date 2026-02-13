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
	"os"
	"path"

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

var _ = Describe("Secure Guest Capacity", func() {
	var (
		originalSecureGuestCapacityPath string
		tempDir                         string
	)

	BeforeEach(func() {
		originalSecureGuestCapacityPath = secureGuestCapacityPath
		tempDir, err := os.MkdirTemp("", "cgroup")
		Expect(err).ToNot(HaveOccurred())
		secureGuestCapacityPath = path.Join(tempDir, "misc.capacity")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
		secureGuestCapacityPath = originalSecureGuestCapacityPath
	})

	Context("when reading secure guest capacity from misc.capacity", func() {
		It("should successfully parse TDX capacity", func() {
			Expect(os.WriteFile(secureGuestCapacityPath, []byte("tdx 15\n"), 0644)).To(Succeed())
			caps, err := GetSecureGuestCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["tdx"]).To(Equal(15))
		})

		It("should successfully parse SEV-SNP capacity", func() {
			Expect(os.WriteFile(secureGuestCapacityPath, []byte("sev 99\nsev_es 2\n"), 0644)).To(Succeed())
			caps, err := GetSecureGuestCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(2))
			Expect(caps["sev"]).To(Equal(99))
			Expect(caps["sev_es"]).To(Equal(2))
		})

		It("should successfully handle empty file", func() {
			Expect(os.WriteFile(secureGuestCapacityPath, []byte(""), 0644)).To(Succeed())
			caps, err := GetSecureGuestCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(BeEmpty())
		})

		It("should return error when file does not exist", func() {
			secureGuestCapacityPath = "/nonexisted_path/misc.capacity"
			caps, err := GetSecureGuestCapacity()
			Expect(err).To(HaveOccurred())
			Expect(caps).To(BeNil())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
