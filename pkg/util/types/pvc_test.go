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

package types

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
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

	Context("StorageClasses", func() {

		var scCache cache.Store
		var sc *storagev1.StorageClass
		var pvc *kubev1.PersistentVolumeClaim

		BeforeEach(func() {
			scCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)

			sc = &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "teststorageclass",
				},
			}

			scCache.Add(sc)

			pvc = &kubev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "testnamespace",
					Name:      "testpvc",
				},
			}
		})

		It("should return StorageClass if explicitly specified", func() {
			pvc.Spec.StorageClassName = &sc.Name

			foundSc, err := GetStorageClass(pvc, scCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSc).To(Equal(sc))
		})

		It("should return StorageClass if is configured as default", func() {
			sc.Annotations = map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			}

			foundSc, err := GetStorageClass(pvc, scCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSc).To(Equal(sc))
		})

		It("should return no StorageClass if no default is configured and none is specified", func() {
			foundSc, err := GetStorageClass(pvc, scCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSc).To(BeNil())
		})

		It("should return no StorageClass if a default is configured and is a statically provisioned volume", func() {
			sc.Annotations = map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			}
			pvc.Spec.StorageClassName = pointer.StringPtr("")

			foundSc, err := GetStorageClass(pvc, scCache)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSc).To(BeNil())
		})

		Context("WaitForFirstConsumer detection", func() {
			BeforeEach(func() {
				pvc.Spec.StorageClassName = &sc.Name
			})

			It("should detect WaitForFirstConsumer when binding mode is explicitly specified", func() {
				bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
				sc.VolumeBindingMode = &bindingMode

				isWFFC, err := IsWaitForFirstConsumer(pvc, scCache)
				Expect(err).ToNot(HaveOccurred())
				Expect(isWFFC).To(BeTrue())
			})

			It("should not detect WaitForFirstConsumer when binding mode is specified to something else", func() {
				bindingMode := storagev1.VolumeBindingImmediate
				sc.VolumeBindingMode = &bindingMode

				isWFFC, err := IsWaitForFirstConsumer(pvc, scCache)
				Expect(err).ToNot(HaveOccurred())
				Expect(isWFFC).To(BeFalse())
			})

			It("should not detect WaitForFirstConsumer when binding mode is not specified", func() {
				isWFFC, err := IsWaitForFirstConsumer(pvc, scCache)
				Expect(err).ToNot(HaveOccurred())
				Expect(isWFFC).To(BeFalse())
			})
		})
	})
})
