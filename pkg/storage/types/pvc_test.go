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
	kubev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

var _ = Describe("PVC type test", func() {

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

	Context("hotplug volume readiness", func() {
		const claimName = "hotplug"
		storageClassName := "sc"
		isController := true

		volume := virtv1.Volume{
			Name: claimName,
			VolumeSource: virtv1.VolumeSource{
				PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: kubev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
					Hotpluggable:                      true,
				},
			},
		}
		hotplugPVC := func(phase kubev1.PersistentVolumeClaimPhase, storageClassName *string) *kubev1.PersistentVolumeClaim {
			return &kubev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: claimName},
				Spec:       kubev1.PersistentVolumeClaimSpec{StorageClassName: storageClassName},
				Status:     kubev1.PersistentVolumeClaimStatus{Phase: phase},
			}
		}
		storageClass := func(mode storagev1.VolumeBindingMode) *storagev1.StorageClass {
			return &storagev1.StorageClass{
				ObjectMeta:        metav1.ObjectMeta{Name: storageClassName},
				VolumeBindingMode: &mode,
			}
		}
		readiness := func(pvc *kubev1.PersistentVolumeClaim, storageClass *storagev1.StorageClass, dataVolumes ...*cdiv1.DataVolume) (bool, bool, error) {
			pvcStore := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			Expect(pvcStore.Add(pvc)).To(Succeed())
			storageClassStore := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			if storageClass != nil {
				Expect(storageClassStore.Add(storageClass)).To(Succeed())
			}
			dataVolumeStore := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			return HotplugVolumeReadyToAttachToNode(namespace, volume, dataVolumes, dataVolumeStore, pvcStore, storageClassStore)
		}

		DescribeTable("should let a claim into the attachment pod once it is bound", func(phase kubev1.PersistentVolumeClaimPhase, storageClassName *string, storageClass *storagev1.StorageClass, expectReady bool) {
			ready, wffc, err := readiness(hotplugPVC(phase, storageClassName), storageClass)
			Expect(err).ToNot(HaveOccurred())
			Expect(ready).To(Equal(expectReady))
			Expect(wffc).To(BeFalse())
		},
			Entry("bound", kubev1.ClaimBound, &storageClassName, storageClass(storagev1.VolumeBindingImmediate), true),
			Entry("pending in an Immediate storage class", kubev1.ClaimPending, &storageClassName, storageClass(storagev1.VolumeBindingImmediate), false),
			Entry("pending in a WaitForFirstConsumer storage class, which the attachment pod binds", kubev1.ClaimPending, &storageClassName, storageClass(storagev1.VolumeBindingWaitForFirstConsumer), true),
			Entry("pending in a storage class that does not exist", kubev1.ClaimPending, &storageClassName, nil, false),
			Entry("pending without a storage class", kubev1.ClaimPending, nil, nil, false),
			Entry("lost", kubev1.ClaimLost, &storageClassName, storageClass(storagev1.VolumeBindingWaitForFirstConsumer), false),
		)

		DescribeTable("should leave a claim CDI owns to its DataVolume", func(phase cdiv1.DataVolumePhase, expectReady, expectWFFC bool) {
			pvc := hotplugPVC(kubev1.ClaimPending, &storageClassName)
			pvc.OwnerReferences = []metav1.OwnerReference{{Kind: "DataVolume", Name: claimName, Controller: &isController}}
			dataVolume := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: claimName},
				Status:     cdiv1.DataVolumeStatus{Phase: phase},
			}

			ready, wffc, err := readiness(pvc, storageClass(storagev1.VolumeBindingImmediate), dataVolume)
			Expect(err).ToNot(HaveOccurred())
			Expect(ready).To(Equal(expectReady))
			Expect(wffc).To(Equal(expectWFFC))
		},
			Entry("pending population", cdiv1.PendingPopulation, true, false),
			Entry("waiting for its first consumer", cdiv1.WaitForFirstConsumer, false, true),
			Entry("importing", cdiv1.ImportInProgress, false, false),
		)

		It("should report a claim that does not exist", func() {
			emptyStore := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			_, _, err := HotplugVolumeReadyToAttachToNode(namespace, volume, nil, emptyStore, emptyStore, emptyStore)
			Expect(err).To(BeAssignableToTypeOf(PvcNotFoundError{}))
		})
	})

})
