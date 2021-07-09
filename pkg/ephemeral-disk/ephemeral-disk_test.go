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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package ephemeraldisk

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("ContainerDisk", func() {
	var imageTempDirPath string
	var backingTempDirPath string
	var pvcBaseTempDirPath string
	var creator *ephemeralDiskCreator

	createBackingImageForPVC := func(volumeName string) {
		os.Mkdir(filepath.Join(pvcBaseTempDirPath, volumeName), 0755)
		f, _ := os.Create(creator.getBackingFilePath(volumeName))
		f.Close()
	}

	AppendEphemeralPVC := func(vmi *v1.VirtualMachineInstance, diskName string, claimName string) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: diskName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{},
			},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: diskName,
			VolumeSource: v1.VolumeSource{
				Ephemeral: &v1.EphemeralVolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
					},
				},
			},
		})

		By("Creating a backing image for the PVC")
		createBackingImageForPVC(diskName)

		// Test the test infra itself: make sure that the backing file has been created.
		_, err := os.Stat(filepath.Join(pvcBaseTempDirPath, diskName, "disk.img"))
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		var err error

		backingTempDirPath, err = ioutil.TempDir("", "ephemeraldisk-backing")
		Expect(err).NotTo(HaveOccurred())

		imageTempDirPath, err = ioutil.TempDir("", "ephemeraldisk-image")
		Expect(err).NotTo(HaveOccurred())

		pvcBaseTempDirPath, err = ioutil.TempDir("", "pvc-base-dir-path")
		Expect(err).NotTo(HaveOccurred())

		creator = &ephemeralDiskCreator{
			mountBaseDir:   imageTempDirPath,
			pvcBaseDir:     pvcBaseTempDirPath,
			discCreateFunc: fakeCreateBackingDisk,
		}
	})

	AfterEach(func() {
		os.RemoveAll(imageTempDirPath)
		os.RemoveAll(backingTempDirPath)
		os.RemoveAll(pvcBaseTempDirPath)
	})

	Describe("ephemeral-backed PVC", func() {
		Context("With single ephemeral volume", func() {
			It("Should create VirtualMachineInstance's ephemeral image", func() {
				By("Creating a minimal VirtualMachineInstance object")
				vmi := v1.NewMinimalVMI("fake-vmi")

				By("Adding a single ephemeral-backed PVC to the VirtualMachineInstance")
				AppendEphemeralPVC(vmi, "fake-disk", "fake-pvc")

				By("Creating VirtualMachineInstance disk image that corresponds to the VMIs PVC")
				err := creator.CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())

				// Now we can test the behavior - the COW image must exist.
				_, err = os.Stat(filepath.Join(creator.mountBaseDir, "fake-disk", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("With multiple ephemeral volumes", func() {
			It("Should create VirtualMachineInstance's ephemeral images", func() {
				By("Creating a minimal VirtualMachineInstance object")
				vmi := v1.NewMinimalVMI("fake-vmi")

				By("Adding multiple ephemeral-backed PVCs to the VirtualMachineInstance")
				AppendEphemeralPVC(vmi, "fake-disk1", "fake-pvc1")
				AppendEphemeralPVC(vmi, "fake-disk2", "fake-pvc2")
				AppendEphemeralPVC(vmi, "fake-disk3", "fake-pvc3")

				By("Creating VirtualMachineInstance disk image that corresponds to the VMIs PVC")
				err := creator.CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())

				// Now we can test the behavior - the COW image must exist.
				_, err = os.Stat(filepath.Join(creator.mountBaseDir, "fake-disk1", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat(filepath.Join(creator.mountBaseDir, "fake-disk2", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat(filepath.Join(creator.mountBaseDir, "fake-disk3", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create ephemeral images in an idempotent way", func() {
				vmi := v1.NewMinimalVMI("fake-vmi")
				AppendEphemeralPVC(vmi, "fake-disk1", "fake-pvc1")
				err := creator.CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())
				err = creator.CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func fakeCreateBackingDisk(backingFile string, imagePath string) ([]byte, error) {
	_, err := os.Stat(backingFile)
	if os.IsNotExist(err) {
		return nil, err
	}
	f, _ := os.Create(imagePath)
	f.Close()
	return nil, nil
}
