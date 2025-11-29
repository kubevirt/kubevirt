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

package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Volume type test", func() {

	Context("IsUtilityVolume", func() {
		It("returns true for utility volume", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					UtilityVolumes: []v1.UtilityVolume{
						{
							Name: "utility-vol",
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
			}

			Expect(IsUtilityVolume(vmi, "utility-vol")).To(BeTrue())
		})

		It("returns false for non-utility volumes", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{
						{
							Name: "regular-volume",
						},
					},
					UtilityVolumes: []v1.UtilityVolume{
						{
							Name: "utility-vol",
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
			}

			Expect(IsUtilityVolume(vmi, "regular-volume")).To(BeFalse())
		})
	})

	Context("GetHotplugVolumes", func() {
		DescribeTable("should not return the new volume", func(volume v1.Volume) {

			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{volume},
				},
			}
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Volumes: []k8sv1.Volume{{Name: "existing"}},
				},
			}
			Expect(GetHotplugVolumes(vmi, pod)).To(BeEmpty())
		},
			Entry("if it already exist", v1.Volume{Name: "existing"}),
			Entry("with HostDisk", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{HostDisk: &v1.HostDisk{}}}),
			Entry("with CloudInitNoCloud", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}}),
			Entry("with CloudInitConfigDrive", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{}}}),
			Entry("with Sysprep", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{Sysprep: &v1.SysprepSource{}}}),
			Entry("with ContainerDisk", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{}}}),
			Entry("with Ephemeral", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}}),
			Entry("with EmptyDisk", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}}),
			Entry("with ConfigMap", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{}}}),
			Entry("with Secret", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{}}}),
			Entry("with DownwardAPI", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{DownwardAPI: &v1.DownwardAPIVolumeSource{}}}),
			Entry("with ServiceAccount", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{ServiceAccount: &v1.ServiceAccountVolumeSource{}}}),
			Entry("with DownwardMetrics", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{DownwardMetrics: &v1.DownwardMetricsVolumeSource{}}}),
		)

		DescribeTable("should return the new volume", func(volume *v1.Volume) {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{*volume},
				},
			}
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Volumes: []k8sv1.Volume{{Name: "existing"}},
				},
			}
			Expect(GetHotplugVolumes(vmi, pod)).To(ContainElement(volume))
		},
			Entry("with DataVolume", &v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{DataVolume: &v1.DataVolumeSource{}}}),
			Entry("with PersistentVolumeClaim", &v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{}}}),
			Entry("with MemoryDump", &v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{MemoryDump: &v1.MemoryDumpVolumeSource{}}}),
		)
	})
})
