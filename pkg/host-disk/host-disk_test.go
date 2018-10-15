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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package hostdisk

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

var _ = Describe("HostDisk", func() {
	var tempDir string

	addHostDisk := func(vmi *v1.VirtualMachineInstance, volumeName string, hostDiskType v1.HostDiskType, capacity string) {
		var quantity resource.Quantity

		err := os.Mkdir(path.Join(tempDir, volumeName), 0755)
		if !os.IsExist(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		if capacity != "" {
			quantity, err = resource.ParseQuantity(capacity)
			Expect(err).NotTo(HaveOccurred())
		}

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				HostDisk: &v1.HostDisk{
					Path:     path.Join(tempDir, volumeName, "disk.img"),
					Type:     hostDiskType,
					Capacity: quantity,
				},
			},
		})
	}

	createTempDiskImg := func(volumeName string) os.FileInfo {
		imgPath := path.Join(tempDir, volumeName, "disk.img")

		err := os.Mkdir(path.Join(tempDir, volumeName), 0755)
		Expect(err).NotTo(HaveOccurred())

		// 67108864 = 64Mi
		err = createSparseRaw(imgPath, 67108864)
		Expect(err).NotTo(HaveOccurred())

		file, err := os.Stat(imgPath)
		Expect(err).NotTo(HaveOccurred())
		return file
	}

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "host-disk-images")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("HostDisk with 'Disk' type", func() {
		It("Should not create a disk.img when it exists", func() {
			By("Creating a disk.img before adding a HostDisk volume")
			tmpDiskImg := createTempDiskImg("volume1")

			By("Creating a new minimal vmi")
			vmi := v1.NewMinimalVMI("fake-vmi")

			By("Adding a HostDisk volume for existing disk.img")
			addHostDisk(vmi, "volume1", v1.HostDiskExists, "")

			By("Executing CreateHostDisks which should not create a disk.img")
			err := CreateHostDisks(vmi)
			Expect(err).NotTo(HaveOccurred())

			// check if disk.img has the same modification time
			// which means that CreateHostDisks function did not create a new disk.img
			hostDiskImg, _ := os.Stat(vmi.Spec.Volumes[0].HostDisk.Path)
			Expect(tmpDiskImg.ModTime()).To(Equal(hostDiskImg.ModTime()))
		})
		It("Should not create a disk.img when it does not exist", func() {
			By("Creating a new minimal vmi")
			vmi := v1.NewMinimalVMI("fake-vmi")

			By("Adding a HostDisk volume")
			addHostDisk(vmi, "volume1", v1.HostDiskExists, "")

			By("Executing CreateHostDisks which should not create disk.img")
			err := CreateHostDisks(vmi)
			Expect(err).NotTo(HaveOccurred())

			// disk.img should not exist
			_, err = os.Stat(vmi.Spec.Volumes[0].HostDisk.Path)
			Expect(true).To(Equal(os.IsNotExist(err)))
		})
	})

	Describe("HostDisk with 'DiskOrCreate' type", func() {
		Context("With multiple HostDisk volumes", func() {
			Context("With non existing disk.img", func() {
				It("Should create disk.img if there is enough space", func() {
					By("Creating a new minimal vmi")
					vmi := v1.NewMinimalVMI("fake-vmi")

					By("Adding a HostDisk volumes")
					addHostDisk(vmi, "volume1", v1.HostDiskExistsOrCreate, "64Mi")
					addHostDisk(vmi, "volume2", v1.HostDiskExistsOrCreate, "128Mi")
					addHostDisk(vmi, "volume3", v1.HostDiskExistsOrCreate, "80Mi")

					By("Executing CreateHostDisks which should create disk.img")
					err := CreateHostDisks(vmi)
					Expect(err).NotTo(HaveOccurred())

					// check if images exist and the size is adequate to requirements
					img1, err := os.Stat(vmi.Spec.Volumes[0].HostDisk.Path)
					Expect(err).NotTo(HaveOccurred())
					Expect(img1.Size()).To(Equal(int64(67108864))) // 64Mi

					img2, err := os.Stat(vmi.Spec.Volumes[1].HostDisk.Path)
					Expect(err).NotTo(HaveOccurred())
					Expect(img2.Size()).To(Equal(int64(134217728))) // 128Mi

					img3, err := os.Stat(vmi.Spec.Volumes[2].HostDisk.Path)
					Expect(err).NotTo(HaveOccurred())
					Expect(img3.Size()).To(Equal(int64(83886080))) // 80Mi
				})
				It("Should stop creating disk images if there is not enough space and should return err", func() {
					By("Creating a new minimal vmi")
					vmi := v1.NewMinimalVMI("fake-vmi")

					By("Adding a HostDisk volumes")
					addHostDisk(vmi, "volume1", v1.HostDiskExistsOrCreate, "64Mi")
					addHostDisk(vmi, "volume2", v1.HostDiskExistsOrCreate, "1E")
					addHostDisk(vmi, "volume3", v1.HostDiskExistsOrCreate, "128Mi")

					By("Executing CreateHostDisks func which should not create a disk.img")
					err := CreateHostDisks(vmi)
					Expect(err).To(HaveOccurred())

					// only first disk.img should be created
					// when there is not enough space anymore
					// function should return err and stop creating images
					img1, err := os.Stat(vmi.Spec.Volumes[0].HostDisk.Path)
					Expect(err).NotTo(HaveOccurred())
					Expect(img1.Size()).To(Equal(int64(67108864))) // 64Mi

					_, err = os.Stat(vmi.Spec.Volumes[1].HostDisk.Path)
					Expect(true).To(Equal(os.IsNotExist(err)))

					_, err = os.Stat(vmi.Spec.Volumes[2].HostDisk.Path)
					Expect(true).To(Equal(os.IsNotExist(err)))
				})
			})
		})
		Context("With existing disk.img", func() {
			It("Should not re-create disk.img", func() {
				By("Creating a disk.img before adding a HostDisk volume")
				tmpDiskImg := createTempDiskImg("volume1")

				By("Creating a new minimal vmi")
				vmi := v1.NewMinimalVMI("fake-vmi")

				By("Adding a HostDisk volume")
				addHostDisk(vmi, "volume1", v1.HostDiskExistsOrCreate, "128Mi")

				By("Executing CreateHostDisks which should not create a disk.img")
				err := CreateHostDisks(vmi)
				Expect(err).NotTo(HaveOccurred())

				// check if disk.img has the same modification time
				// which means that CreateHostDisks function did not create a new disk.img
				capacity := vmi.Spec.Volumes[0].HostDisk.Capacity
				specSize, _ := capacity.AsInt64()
				hostDiskImg, _ := os.Stat(vmi.Spec.Volumes[0].HostDisk.Path)
				Expect(tmpDiskImg.ModTime()).To(Equal(hostDiskImg.ModTime()))
				// check if img has the same size as before
				Expect(tmpDiskImg.Size()).NotTo(Equal(specSize))
				Expect(tmpDiskImg.Size()).To(Equal(int64(67108864)))
			})
		})
	})

	Describe("HostDisk with unkown type", func() {
		It("Should not create a disk.img", func() {
			By("Creating a new minimal vmi")
			vmi := v1.NewMinimalVMI("fake-vmi")

			By("Adding a HostDisk volume with unknown type")
			addHostDisk(vmi, "volume1", "UnknownType", "")

			By("Executing CreateHostDisks which should not create a disk.img")
			err := CreateHostDisks(vmi)
			Expect(err).NotTo(HaveOccurred())

			// disk.img should not exist
			_, err = os.Stat(vmi.Spec.Volumes[0].HostDisk.Path)
			Expect(true).To(Equal(os.IsNotExist(err)))
		})
	})

	Describe("VMI with PVC volume", func() {

		var virtClient *kubecli.MockKubevirtClient

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			kubeClient := fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		})

		table.DescribeTable("PVC in", func(mode k8sv1.PersistentVolumeMode) {

			By("Creating the PVC")
			namespace := "testns"
			pvcName := "pvcDevice"
			pvc := &k8sv1.PersistentVolumeClaim{
				TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: pvcName},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					VolumeMode: &mode,
				},
			}

			virtClient.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)

			By("Creating a VMI with PVC volume")
			volumeName := "pvc-volume"
			volumes := []v1.Volume{
				{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
					},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: namespace, UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
			}

			By("Replacing PVCs with hostdisks")
			ReplacePVCByHostDisk(vmi, virtClient)

			Expect(len(vmi.Spec.Volumes)).To(Equal(1), "There should still be 1 volume")

			if mode == k8sv1.PersistentVolumeFilesystem {
				Expect(vmi.Spec.Volumes[0].HostDisk).NotTo(BeNil(), "There should be a hostdisk volume")
				Expect(vmi.Spec.Volumes[0].HostDisk.Type).To(Equal(v1.HostDiskExistsOrCreate), "Correct hostdisk type")
				Expect(vmi.Spec.Volumes[0].HostDisk.Path).NotTo(BeNil(), "Hostdisk path is filled")
				Expect(vmi.Spec.Volumes[0].HostDisk.Capacity).NotTo(BeNil(), "Hostdisk capacity is filled")

				Expect(vmi.Spec.Volumes[0].PersistentVolumeClaim).To(BeNil(), "There shouldn't be a PVC volume anymore")
			} else if mode == k8sv1.PersistentVolumeBlock {
				Expect(vmi.Spec.Volumes[0].HostDisk).To(BeNil(), "There should be no hostdisk volume")
				Expect(vmi.Spec.Volumes[0].PersistentVolumeClaim).ToNot(BeNil(), "There should still be a PVC volume")
				Expect(vmi.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "There should still be the correct PVC volume")
			} else {
				Fail("unknown PVC mode!")
			}

		},
			table.Entry("filemode", k8sv1.PersistentVolumeFilesystem),
			table.Entry("blockmode", k8sv1.PersistentVolumeBlock),
		)
	})

})
