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
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("PVC utils test", func() {

	namespace := "testns"
	file1Name := "file1"
	file2Name := "file2"
	blockName := "block"

	filePvc1 := kubev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: file1Name},
	}

	modeFile := kubev1.PersistentVolumeFilesystem
	filePvc2 := kubev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: file2Name},
		Spec: kubev1.PersistentVolumeClaimSpec{
			VolumeMode: &modeFile,
		},
	}

	modeBlock := kubev1.PersistentVolumeBlock
	blockPvc := kubev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: blockName},
		Spec: kubev1.PersistentVolumeClaimSpec{
			VolumeMode:  &modeBlock,
			AccessModes: []kubev1.PersistentVolumeAccessMode{kubev1.ReadWriteMany},
		},
	}

	Context("PVC block device test with store", func() {

		pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		pvcCache.Add(&filePvc1)
		pvcCache.Add(&filePvc2)
		pvcCache.Add(&blockPvc)

		It("should handle non existing PVC", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromStore(pvcCache, namespace, "doesNotExist")
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).To(BeNil(), "PVC is nil")
			Expect(exists).To(BeFalse(), "PVC was not found")
			Expect(isBlock).To(BeFalse(), "Is filesystem PVC")
		})

		It("should detect filesystem device for empty VolumeMode", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromStore(pvcCache, namespace, file1Name)
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).ToNot(BeNil(), "PVC isn't nil")
			Expect(exists).To(BeTrue(), "PVC was found")
			Expect(isBlock).To(BeFalse(), "Is filesystem PVC")
		})

		It("should detect filesystem device for filesystem VolumeMode", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromStore(pvcCache, namespace, file2Name)
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).ToNot(BeNil(), "PVC isn't nil")
			Expect(exists).To(BeTrue(), "PVC was found")
			Expect(isBlock).To(BeFalse(), "Is filesystem PVC")
		})

		It("should detect block device for block VolumeMode", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromStore(pvcCache, namespace, blockName)
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).ToNot(BeNil(), "PVC isn't nil")
			Expect(exists).To(BeTrue(), "PVC was found")
			Expect(isBlock).To(BeTrue(), "Is blockdevice PVC")
		})
	})

	Context("GetTotalSizeMigratedVolumes", func() {
		It("should return 0 when no migrated volumes", func() {
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{},
				},
			}
			result := GetTotalSizeMigratedVolumes(vmi)
			Expect(result.Value()).To(Equal(int64(0)))
		})

		It("should return 0 when SourcePVCInfo is nil", func() {
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName:    "test-vol",
							SourcePVCInfo: nil,
						},
					},
				},
			}
			result := GetTotalSizeMigratedVolumes(vmi)
			Expect(result.Value()).To(Equal(int64(0)))
		})

		It("should calculate size correctly for 10Gi volume using SourcePVCInfo", func() {
			tenGi := resource.MustParse("10Gi")
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName: "test-vol",
							SourcePVCInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName: "source-pvc",
								Capacity: k8sv1.ResourceList{
									k8sv1.ResourceStorage: tenGi,
								},
								Requests: k8sv1.ResourceList{
									k8sv1.ResourceStorage: tenGi,
								},
							},
							DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName: "dest-pvc",
							},
						},
					},
				},
			}
			result := GetTotalSizeMigratedVolumes(vmi)
			Expect(result.Value()).To(Equal(tenGi.Value()))
		})

		It("should sum multiple migrated volumes", func() {
			tenGi := resource.MustParse("10Gi")
			fiveGi := resource.MustParse("5Gi")
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName: "vol1",
							SourcePVCInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName: "source-pvc-1",
								Capacity: k8sv1.ResourceList{
									k8sv1.ResourceStorage: tenGi,
								},
								Requests: k8sv1.ResourceList{
									k8sv1.ResourceStorage: tenGi,
								},
							},
						},
						{
							VolumeName: "vol2",
							SourcePVCInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName: "source-pvc-2",
								Capacity: k8sv1.ResourceList{
									k8sv1.ResourceStorage: fiveGi,
								},
								Requests: k8sv1.ResourceList{
									k8sv1.ResourceStorage: fiveGi,
								},
							},
						},
					},
				},
			}
			result := GetTotalSizeMigratedVolumes(vmi)
			expectedSize := tenGi.Value() + fiveGi.Value()
			Expect(result.Value()).To(Equal(expectedSize))
		})
	})
})
