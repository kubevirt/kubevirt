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
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"
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

	Context("PVC block device test with client", func() {

		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		kubeClient := fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		kubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), &filePvc1, metav1.CreateOptions{})
		kubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), &filePvc2, metav1.CreateOptions{})
		kubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), &blockPvc, metav1.CreateOptions{})

		It("should handle non existing PVC", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromClient(virtClient, namespace, "doesNotExist")
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).To(BeNil(), "PVC is nil")
			Expect(exists).To(BeFalse(), "PVC was not found")
			Expect(isBlock).To(BeFalse(), "Is filesystem PVC")
		})

		It("should detect filesystem device for empty VolumeMode", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromClient(virtClient, namespace, file1Name)
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).ToNot(BeNil(), "PVC isn't nil")
			Expect(exists).To(BeTrue(), "PVC was found")
			Expect(isBlock).To(BeFalse(), "Is filesystem PVC")
		})

		It("should detect filesystem device for filesystem VolumeMode", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromClient(virtClient, namespace, file2Name)
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).ToNot(BeNil(), "PVC isn't nil")
			Expect(exists).To(BeTrue(), "PVC was found")
			Expect(isBlock).To(BeFalse(), "Is filesystem PVC")
		})

		It("should detect block device for block VolumeMode", func() {
			pvc, exists, isBlock, err := IsPVCBlockFromClient(virtClient, namespace, blockName)
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).ToNot(BeNil(), "PVC isn't nil")
			Expect(pvc.Name).To(Equal(blockName), "correct PVC was found")
			Expect(exists).To(BeTrue(), "PVC was found")
			Expect(isBlock).To(BeTrue(), "Is blockdevice PVC")
		})
		It("should detect shared block device for block VolumeMode", func() {
			pvc, isShared, err := IsSharedPVCFromClient(virtClient, namespace, blockName)
			Expect(err).ToNot(HaveOccurred(), "no error occurred")
			Expect(pvc).ToNot(BeNil(), "PVC isn't nil")
			Expect(pvc.Name).To(Equal(blockName), "correct PVC was found")
			Expect(isShared).To(BeTrue(), "Is PVC Shared")
		})
	})

})
