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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	api2 "kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("ContainerDisk", func() {
	var imageTempDirPath string
	var backingTempDirPath string
	var pvcBaseTempDirPath string
	var blockDevBaseDir string
	var creator *ephemeralDiskCreator

	createBackingImageForPVC := func(volumeName string, isBlock bool) {
		os.Mkdir(filepath.Join(pvcBaseTempDirPath, volumeName), 0755)
		f, _ := os.Create(creator.getBackingFilePath(volumeName, isBlock))
		f.Close()
	}

	AppendEphemeralPVC := func(vmi *v1.VirtualMachineInstance, diskName string, claimName string, backingDiskIsblock bool) {
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
		createBackingImageForPVC(diskName, backingDiskIsblock)

		// Test the test infra itself: make sure that the backing file has been created.
		var err error
		if backingDiskIsblock {
			_, err = os.Stat(filepath.Join(blockDevBaseDir, diskName))
		} else {
			_, err = os.Stat(filepath.Join(pvcBaseTempDirPath, diskName, "disk.img"))
		}
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

		blockDevBaseDir, err = ioutil.TempDir("", "block-dev-base-dir-path")
		Expect(err).NotTo(HaveOccurred())

		creator = &ephemeralDiskCreator{
			mountBaseDir:    imageTempDirPath,
			pvcBaseDir:      pvcBaseTempDirPath,
			blockDevBaseDir: blockDevBaseDir,
			discCreateFunc:  fakeCreateBackingDisk,
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
				vmi := api2.NewMinimalVMI("fake-vmi")

				By("Adding a single ephemeral-backed PVC to the VirtualMachineInstance")
				AppendEphemeralPVC(vmi, "fake-disk", "fake-pvc", false)

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
				By("Creating a minimal VirtualMachineInstance object")
				vmi := api2.NewMinimalVMI("fake-vmi")

				By("Adding multiple ephemeral-backed PVCs to the VirtualMachineInstance")
				AppendEphemeralPVC(vmi, "fake-disk1", "fake-pvc1", false)
				AppendEphemeralPVC(vmi, "fake-disk2", "fake-pvc2", false)
				AppendEphemeralPVC(vmi, "fake-disk3", "fake-pvc3", false)

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
				vmi := api2.NewMinimalVMI("fake-vmi")
				AppendEphemeralPVC(vmi, "fake-disk1", "fake-pvc1", false)
				err := creator.CreateEphemeralImages(vmi, &api.Domain{})
				Expect(err).NotTo(HaveOccurred())
				err = creator.CreateEphemeralImages(vmi, &api.Domain{})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("With a block pvc backed ephemeral volume", func() {
			It("Should create VirtualMachineInstance's ephemeral image", func() {
				By("Creating a minimal VirtualMachineInstance object")
				vmi := api2.NewMinimalVMI("fake-vmi")

				By("Adding a single ephemeral-backed PVC to the VirtualMachineInstance")
				AppendEphemeralPVC(vmi, "fake-disk", "fake-pvc", true)

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
	if os.IsNotExist(err) {
		return nil, err
	}
	f, _ := os.Create(imagePath)
	f.Close()
	return nil, nil
}
