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

package diskimg_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/storage/diskimg"
	"kubevirt.io/kubevirt/pkg/storage/volumepath"
)

var _ = Describe("ImageBackedPVCImages", func() {
	const volumeName = "disk0"

	storage := func(q string) k8sv1.ResourceList {
		if q == "" {
			return k8sv1.ResourceList{}
		}
		return k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse(q)}
	}

	pvcStatus := func(mode k8sv1.PersistentVolumeMode, capacity, requests string) v1.VolumeStatus {
		return v1.VolumeStatus{
			Name: volumeName,
			PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
				ClaimName:  "claim",
				VolumeMode: &mode,
				Capacity:   storage(capacity),
				Requests:   storage(requests),
			},
		}
	}

	withHotplug := func(status v1.VolumeStatus) v1.VolumeStatus {
		status.HotplugVolume = &v1.HotplugVolumeStatus{}
		return status
	}

	image := func(size string) diskimg.Image {
		q := resource.MustParse(size)
		return diskimg.Image{
			Dir:  volumepath.FilesystemDir(volumeName),
			Name: "disk.img",
			Size: q.Value(),
		}
	}

	DescribeTable("should size the image from the PVC", func(capacity, requests, expected string) {
		vmi := libvmi.New(
			libvmi.WithPersistentVolumeClaim(volumeName, "claim"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithVolumeStatus(pvcStatus(k8sv1.PersistentVolumeFilesystem, capacity, requests)),
			)),
		)
		Expect(diskimg.ImageBackedPVCImages(vmi)).To(ConsistOf(image(expected)))
	},
		Entry("with equal capacity and requests", "2Gi", "2Gi", "2Gi"),
		Entry("with requests smaller than capacity", "2Gi", "1Gi", "1Gi"),
		Entry("without capacity", "", "1Gi", "1Gi"),
		Entry("without requests", "2Gi", "", "2Gi"),
		Entry("aligned down to 1MiB", "", "1537Ki", "1Mi"),
	)

	It("should create an image for a filesystem DataVolume", func() {
		vmi := libvmi.New(
			libvmi.WithDataVolume(volumeName, "dv"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithVolumeStatus(pvcStatus(k8sv1.PersistentVolumeFilesystem, "2Gi", "2Gi")),
			)),
		)
		Expect(diskimg.ImageBackedPVCImages(vmi)).To(ConsistOf(image("2Gi")))
	})

	DescribeTable("should not create an image", func(vmi *v1.VirtualMachineInstance) {
		Expect(diskimg.ImageBackedPVCImages(vmi)).To(BeEmpty())
	},
		Entry("for a block PVC", libvmi.New(
			libvmi.WithPersistentVolumeClaim(volumeName, "claim"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithVolumeStatus(pvcStatus(k8sv1.PersistentVolumeBlock, "2Gi", "2Gi")),
			)),
		)),
		Entry("for a PVC without PVC info", libvmi.New(
			libvmi.WithPersistentVolumeClaim(volumeName, "claim"),
		)),
		Entry("for a hotplugged PVC", libvmi.New(
			libvmi.WithHotplugPersistentVolumeClaim(volumeName, "claim"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithVolumeStatus(withHotplug(pvcStatus(k8sv1.PersistentVolumeFilesystem, "2Gi", "2Gi"))),
			)),
		)),
		Entry("for a PVC shared through virtiofs", libvmi.New(
			libvmi.WithFilesystemPVC(volumeName),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithVolumeStatus(pvcStatus(k8sv1.PersistentVolumeFilesystem, "2Gi", "2Gi")),
			)),
		)),
		Entry("for a HostDisk", libvmi.New(
			libvmi.WithHostDiskAndCapacity(volumeName, "/data/disk.img", v1.HostDiskExistsOrCreate, "1Gi"),
		)),
	)

	It("should fail when the PVC provides neither capacity nor requests", func() {
		vmi := libvmi.New(
			libvmi.WithPersistentVolumeClaim(volumeName, "claim"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithVolumeStatus(pvcStatus(k8sv1.PersistentVolumeFilesystem, "", "")),
			)),
		)
		_, err := diskimg.ImageBackedPVCImages(vmi)
		Expect(err).To(HaveOccurred())
	})

	It("should fail when the PVC is smaller than 1MiB", func() {
		vmi := libvmi.New(
			libvmi.WithPersistentVolumeClaim(volumeName, "claim"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithVolumeStatus(pvcStatus(k8sv1.PersistentVolumeFilesystem, "", "512Ki")),
			)),
		)
		_, err := diskimg.ImageBackedPVCImages(vmi)
		Expect(err).To(HaveOccurred())
	})
})
