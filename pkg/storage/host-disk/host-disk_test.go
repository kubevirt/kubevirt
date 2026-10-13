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

package hostdisk

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/fake"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/storage/diskimg"
)

var _ = Describe("HostDisk", func() {
	Describe("Images", func() {
		It("should only return images for DiskOrCreate HostDisks", func() {
			vmi := libvmi.New(
				libvmi.WithHostDisk("exists", "/data/exists/disk.img", v1.HostDiskExists),
				libvmi.WithHostDiskAndCapacity("create", "/data/create/my.img", v1.HostDiskExistsOrCreate, "64Mi"),
				libvmi.WithHostDisk("unknown", "/data/unknown/disk.img", "UnknownType"),
				libvmi.WithPersistentVolumeClaim("pvc", "claim"),
			)

			Expect(Images(vmi)).To(Equal([]diskimg.Image{{
				Dir:  GetMountedHostDiskDir("create"),
				Name: "my.img",
				Size: 64 << 20,
			}}))
		})
	})

	Describe("VMI with PVC volume", func() {

		var (
			virtClient *kubecli.MockKubevirtClient
			vmi        *v1.VirtualMachineInstance
		)

		const (
			pvcName    = "madeup"
			volumeName = "pvc-volume"
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			kubeClient := fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			By("Creating a VMI with PVC volume")
			vmi = libvmi.New(
				libvmi.WithPersistentVolumeClaim(volumeName, pvcName),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithVolumeStatus(v1.VolumeStatus{
						Name:                      volumeName,
						PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{},
					}),
				)),
			)
		})

		assertHostDiskWithSize := func(expectedSize resource.Quantity) {
			volume := vmi.Spec.Volumes[0]
			Expect(volume.HostDisk).NotTo(BeNil(), "There should be a hostdisk volume")
			Expect(volume.HostDisk.Type).To(Equal(v1.HostDiskExistsOrCreate), "Correct hostdisk type")
			Expect(volume.HostDisk.Path).NotTo(BeNil(), "Hostdisk path is filled")
			Expect(volume.PersistentVolumeClaim).To(BeNil(), "There shouldn't be a PVC volume anymore")
			Expect(volume.HostDisk.Capacity.Value()).To(Equal(expectedSize.Value()), "Hostdisk capacity is filled")
		}

		assertNoHostDisk := func() {
			volume := vmi.Spec.Volumes[0]
			Expect(volume.HostDisk).To(BeNil(), "There should be no hostdisk volume")
			Expect(volume.PersistentVolumeClaim).ToNot(BeNil(), "There should still be a PVC volume")
			Expect(volume.PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "There should still be the correct PVC volume")
		}

		DescribeTable("in", func(mode k8sv1.PersistentVolumeMode, devices v1.Devices, capacity, requests k8sv1.ResourceList, validateFunc func()) {
			vmi.Spec.Domain.Devices = devices
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.VolumeMode = &mode
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.Capacity = capacity
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.Requests = requests

			By("Replacing PVCs with hostdisks")
			Expect(ReplacePVCByHostDisk(vmi)).To(Succeed())
			Expect(vmi.Spec.Volumes).To(HaveLen(1), "There should always be 1 volume")
			validateFunc()
		},

			Entry("filemode",
				k8sv1.PersistentVolumeFilesystem,
				v1.Devices{},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				func() {
					assertHostDiskWithSize(resource.MustParse("2Gi"))
				},
			),
			Entry("filemode with smaller requested PVC size",
				k8sv1.PersistentVolumeFilesystem,
				v1.Devices{},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("1Gi")},
				func() {
					assertHostDiskWithSize(resource.MustParse("1Gi"))
				},
			),
			Entry("filemode without capacity PVC size",
				k8sv1.PersistentVolumeFilesystem,
				v1.Devices{},
				k8sv1.ResourceList{},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("1Gi")},
				func() {
					assertHostDiskWithSize(resource.MustParse("1Gi"))
				},
			),
			Entry("filemode without requested PVC size",
				k8sv1.PersistentVolumeFilesystem,
				v1.Devices{},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				k8sv1.ResourceList{},
				func() {
					assertHostDiskWithSize(resource.MustParse("2Gi"))
				},
			),
			Entry("blockmode",
				k8sv1.PersistentVolumeBlock,
				v1.Devices{},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				assertNoHostDisk,
			),
			Entry("filesystem passthrough",
				k8sv1.PersistentVolumeFilesystem,
				v1.Devices{
					Filesystems: []v1.Filesystem{{
						Name:     volumeName,
						Virtiofs: &v1.FilesystemVirtiofs{},
					}},
				},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("2Gi")},
				assertNoHostDisk,
			),
		)

		It("in filemode without capacity or requested PVC size should fail", func() {
			mode := k8sv1.PersistentVolumeFilesystem
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.VolumeMode = &mode
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.Capacity = k8sv1.ResourceList{}
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.Requests = k8sv1.ResourceList{}
			Expect(ReplacePVCByHostDisk(vmi)).ToNot(Succeed())
		})
	})
})
