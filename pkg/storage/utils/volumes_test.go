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

package utils

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("GetVolumes", func() {

	const backendVolume = "persistent-state-for-"

	createVMI := func(hasEFI, hasTPM bool, name string) *v1.VirtualMachineInstance {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{
					{Name: "rootdisk"},
				},
				Domain: v1.DomainSpec{
					Firmware: &v1.Firmware{
						Bootloader: &v1.Bootloader{
							EFI: nil,
						},
					},
					Devices: v1.Devices{
						TPM: nil,
					},
				},
			},
			Status: v1.VirtualMachineInstanceStatus{
				VolumeStatus: []v1.VolumeStatus{
					{
						Name: backendVolume + name,
						PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
							ClaimName: backendVolume,
						},
					},
				},
			},
		}

		if hasEFI {
			vmi.Spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{
				Persistent: pointer.P(true),
			}
		}

		if hasTPM {
			vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
				Persistent: pointer.P(true),
			}
		}

		return vmi
	}

	DescribeTable("should handle volume exclusions based on flags",
		func(hasEFI, hasTPM bool, expectedVolumes []string, opts ...VolumeOption) {
			vmi := createVMI(hasEFI, hasTPM, "")
			client := kubecli.NewMockKubevirtClient(nil) // Mock client for testing
			volumes, _ := GetVolumes(vmi, client, opts...)

			Expect(volumes).To(HaveLen(len(expectedVolumes)))
			for _, expectedVolumeName := range expectedVolumes {
				Expect(volumes).To(ContainElement(HaveField("Name", Equal(expectedVolumeName))))
			}
		},
		Entry("when no options are provided",
			true,
			true,
			[]string{"rootdisk"},
		),
		Entry("when WithBackendVolume is provided and both EFI and TPM are set",
			true,
			true,
			[]string{backendVolume},
			WithBackendVolume,
		),
		Entry("when WithRegularVolumes is provided and only EFI is set",
			true,
			false,
			[]string{"rootdisk"},
			WithRegularVolumes,
		),
		Entry("when no backend volumes and WithBackendVolume is set",
			false,
			false,
			[]string{},
			WithBackendVolume,
		),
		Entry("when WithAllVolumes is provided and both EFI and TPM are set",
			true,
			true,
			[]string{"rootdisk", backendVolume},
			WithAllVolumes,
		),
		Entry("when WithAllVolumes is provided and no EFI or TPM is set",
			false,
			false,
			[]string{"rootdisk"},
			WithAllVolumes,
		),
	)

	It("should trim backend volume name", func() {
		vmi := createVMI(true, true, strings.Repeat("a", 63))
		client := kubecli.NewMockKubevirtClient(nil)
		volumes, err := GetVolumes(vmi, client, WithBackendVolume)
		Expect(err).ToNot(HaveOccurred())
		Expect(volumes).To(HaveLen(1))
		Expect(len(volumes[0].Name)).To(BeNumerically("<", 63))
	})
})
