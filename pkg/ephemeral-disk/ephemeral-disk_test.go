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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("ContainerDisk", func() {
	var imageTempDirPath string
	var pvcBaseTempDirPath string
	var blockDevBaseDir string
	var creator *ephemeralDiskCreator

	createBackingImageForPVC := func(volumeName string, isBlock bool) error {
		if err := os.Mkdir(filepath.Join(pvcBaseTempDirPath, volumeName), 0o755); err != nil {
			return err
		}
		f, err := os.Create(creator.getBackingFilePath(volumeName, isBlock))
		if err != nil {
			return err
		}
		defer f.Close()
		// Test the test infra itself: make sure that the backing file has been created.
		if isBlock {
			if _, err := os.Stat(filepath.Join(blockDevBaseDir, volumeName)); err != nil {
				return err
			}
		} else {
			if _, err := os.Stat(filepath.Join(pvcBaseTempDirPath, volumeName, "disk.img")); err != nil {
				return err
			}
		}
		return nil
	}

	BeforeEach(func() {
		imageTempDirPath = GinkgoT().TempDir()
		pvcBaseTempDirPath = GinkgoT().TempDir()
		blockDevBaseDir = GinkgoT().TempDir()

		creator = &ephemeralDiskCreator{
			mountBaseDir:    imageTempDirPath,
			pvcBaseDir:      pvcBaseTempDirPath,
			blockDevBaseDir: blockDevBaseDir,
			discCreateFunc:  fakeCreateBackingDisk,
		}
	})

	Describe("ephemeral-backed PVC", func() {
		Context("With single ephemeral volume", func() {
			It("Should create VirtualMachineInstance's ephemeral image", func() {
				By("Creating a minimal VirtualMachineInstance object with single ephemeral-backed PVC")
				vmi := libvmi.New(
					libvmi.WithEphemeralPersistentVolumeClaim("fake-disk", "fake-pvc"),
				)

				By("Creating a backing image for the PVC")
				Expect(createBackingImageForPVC("fake-disk", false)).To(Succeed())

				By("Creating VirtualMachineInstance disk image that corresponds to the VMIs PVC")
				err := creator.CreateEphemeralImages(vmi, &api.Domain{})
				Expect(err).NotTo(HaveOccurred())

				// Now we can test the behavior - the COW image must exist.
				_, err = os.Stat(filepath.Join(creator.mountBaseDir, "fake-disk", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("With multiple ephemeral volumes", func() {
			It("Should create VirtualMachineInstance's ephemeral images", func() {
				By("Creating a minimal VirtualMachineInstance object with multiple ephemeral-backed PVC")
				vmi := libvmi.New(
					libvmi.WithEphemeralPersistentVolumeClaim("fake-disk1", "fake-pvc1"),
					libvmi.WithEphemeralPersistentVolumeClaim("fake-disk2", "fake-pvc2"),
					libvmi.WithEphemeralPersistentVolumeClaim("fake-disk3", "fake-pvc3"),
				)

				By("Creating a backing images for the PVC")
				Expect(createBackingImageForPVC("fake-disk1", false)).To(Succeed())
				Expect(createBackingImageForPVC("fake-disk2", false)).To(Succeed())
				Expect(createBackingImageForPVC("fake-disk3", false)).To(Succeed())

				By("Creating VirtualMachineInstance disk image that corresponds to the VMIs PVC")
				err := creator.CreateEphemeralImages(vmi, &api.Domain{})
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
				By("Creating a minimal VirtualMachineInstance object with single ephemeral-backed PVC")
				vmi := libvmi.New(
					libvmi.WithEphemeralPersistentVolumeClaim("fake-disk", "fake-pvc"),
				)

				By("Creating a backing image for the PVC")
				Expect(createBackingImageForPVC("fake-disk", false)).To(Succeed())

				err := creator.CreateEphemeralImages(vmi, &api.Domain{})
				Expect(err).NotTo(HaveOccurred())
				err = creator.CreateEphemeralImages(vmi, &api.Domain{})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("With a block pvc backed ephemeral volume", func() {
			It("Should create VirtualMachineInstance's ephemeral image", func() {
				By("Creating a minimal VirtualMachineInstance object with single ephemeral-backed PVC")
				vmi := libvmi.New(
					libvmi.WithEphemeralPersistentVolumeClaim("fake-disk", "fake-pvc"),
				)

				By("Creating a backing images for the PVC")
				Expect(createBackingImageForPVC("fake-disk", true)).To(Succeed())

				By("Creating VirtualMachineInstance disk image that corresponds to the VMIs PVC")
				err := creator.CreateEphemeralImages(vmi, &api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Disks: []api.Disk{
								{
									BackingStore: &api.BackingStore{
										Type: "block",
										Source: &api.DiskSource{
											Dev:  filepath.Join(creator.blockDevBaseDir, "fake-disk"),
											Name: "fake-disk",
										},
									},
								},
							},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				// Now we can test the behavior - the COW image must exist.
				_, err = os.Stat(filepath.Join(creator.mountBaseDir, "fake-disk", "disk.qcow2"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func fakeCreateBackingDisk(backingFile string, backingFormat string, imagePath string) ([]byte, error) {
	if backingFormat != "raw" {
		return nil, fmt.Errorf("wrong backing format")
	}
	_, err := os.Stat(backingFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.Create(imagePath)
	if err != nil {
		return nil, err
	}
	err = f.Close()
	return nil, err
}
