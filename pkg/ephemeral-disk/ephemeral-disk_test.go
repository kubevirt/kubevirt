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
	"os/exec"
	"os/user"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("RegistryDisk", func() {
	var imageTempDirPath string
	var backingTempDirPath string

	owner, err := user.Current()
	if err != nil {
		panic(err)
	}

	createBackingImageForPVC := func(volumeName string) {
		// Note that this is within our tmpdir -- cleanup is done after tests.
		os.Mkdir(filepath.Join(pvcBaseDir, volumeName), 0755)

		var args []string

		args = append(args, "create")
		args = append(args, "-f")
		args = append(args, "raw")
		args = append(args, getBackingFilePath(volumeName))
		args = append(args, "1K")

		// Requires qemu-img binary to be present.
		cmd := exec.Command("qemu-img", args...)
		err := cmd.Run()
		Expect(err).NotTo(HaveOccurred())
	}

	AppendEphemeralPVC := func(vmi *v1.VirtualMachineInstance, diskName string, volumeName string, claimName string) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name:       diskName,
			VolumeName: volumeName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{},
			},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				Ephemeral: &v1.EphemeralVolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
					},
				},
			},
		})

		By("Creating a backing image for the PVC")
		createBackingImageForPVC(volumeName)

		// Test the test infra itself: make sure that the backing file has been created.
		_, err := os.Stat(filepath.Join(pvcBaseDir, volumeName, "disk.img"))
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		SetLocalDataOwner(owner.Username)

		backingTempDirPath, err := ioutil.TempDir("", "ephemeraldisk-backing")
		Expect(err).NotTo(HaveOccurred())
		err = setBackingDirectory(backingTempDirPath)
		Expect(err).NotTo(HaveOccurred())

		imageTempDirPath, err = ioutil.TempDir("", "ephemeraldisk-image")
		Expect(err).NotTo(HaveOccurred())
		err = SetLocalDirectory(imageTempDirPath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(imageTempDirPath)
		os.RemoveAll(backingTempDirPath)
	})

	Describe("ephemeral-backed PVC", func() {
		Context("With single ephemeral volume", func() {
			It("Should create VirtualMachineInstance's ephemeral image", func() {
				By("Creating a minimal VirtualMachineInstance object")
				vmi := v1.NewMinimalVMI("fake-vmi")

				By("Adding a single ephemeral-backed PVC to the VirtualMachineInstance")
				AppendEphemeralPVC(vmi, "fake-disk", "fake-volume", "fake-pvc")

				By("Creating VirtualMachineInstance disk image that corresponds to the VMIs PVC")
				err := CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())

				// Now we can test the behavior - the COW image must exist.
				_, err = os.Stat(filepath.Join(mountBaseDir, "fake-volume", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("With multiple ephemeral volumes", func() {
			It("Should create VirtualMachineInstance's ephemeral images", func() {
				By("Creating a minimal VirtualMachineInstance object")
				vmi := v1.NewMinimalVMI("fake-vmi")

				By("Adding multiple ephemeral-backed PVCs to the VirtualMachineInstance")
				AppendEphemeralPVC(vmi, "fake-disk1", "fake-volume1", "fake-pvc1")
				AppendEphemeralPVC(vmi, "fake-disk2", "fake-volume2", "fake-pvc2")
				AppendEphemeralPVC(vmi, "fake-disk3", "fake-volume3", "fake-pvc3")

				By("Creating VirtualMachineInstance disk image that corresponds to the VMIs PVC")
				err := CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())

				// Now we can test the behavior - the COW image must exist.
				_, err = os.Stat(filepath.Join(mountBaseDir, "fake-volume1", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat(filepath.Join(mountBaseDir, "fake-volume2", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat(filepath.Join(mountBaseDir, "fake-volume3", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create ephemeral images in an idempotent way", func() {
				vmi := v1.NewMinimalVMI("fake-vmi")
				AppendEphemeralPVC(vmi, "fake-disk1", "fake-volume1", "fake-pvc1")
				err := CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())
				err = CreateEphemeralImages(vmi)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
