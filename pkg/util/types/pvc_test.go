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
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	kubev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
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

	Context("PVC provisioning failure detection", func() {

		var pvcCache cache.Store
		var scCache cache.Store
		var kubeClient *fake.Clientset
		var virtClient *kubecli.MockKubevirtClient
		var pvc *kubev1.PersistentVolumeClaim

		BeforeEach(func() {
			pvcCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			scCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)

			kubeClient = fake.NewSimpleClientset()
			virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))

			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			pvc = &kubev1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "testnamespace",
					Name:      "testpvc",
				},
				Status: kubev1.PersistentVolumeClaimStatus{
					Phase: kubev1.ClaimPending,
				},
			}
		})

		It("should detect no provisioning failures when PVC is bound", func() {
			pvc.Status.Phase = kubev1.ClaimBound
			pvcCache.Add(pvc)

			failed, message, err := IsPVCFailedProvisioning(pvcCache, scCache, virtClient, pvc.Namespace, pvc.Name)

			Expect(failed).To(BeFalse())
			Expect(message).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
		})

		table.DescribeTable("should detect PVC provisioning failure events", func(eventReason string) {
			pvcCache.Add(pvc)

			kubeClient.Fake.PrependReactor("list", "events", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, &kubev1.EventList{
					Items: []kubev1.Event{
						{
							InvolvedObject: kubev1.ObjectReference{
								APIVersion: pvc.APIVersion,
								Kind:       pvc.Kind,
								Namespace:  pvc.Namespace,
								Name:       pvc.Name,
							},
							Reason: eventReason,
						},
					},
				}, nil
			})

			failed, message, err := IsPVCFailedProvisioning(pvcCache, scCache, virtClient, pvc.Namespace, pvc.Name)

			Expect(failed).To(BeTrue())
			Expect(message).ToNot(BeZero())
			Expect(err).ToNot(HaveOccurred())
		},
			table.Entry("ProvisioningFailed event", "ProvisioningFailed"),
			table.Entry("FailedBinding event", "FailedBinding"),
		)

		It("Should detect PVC provisioning failure when pending for more than timeout threshold", func() {
			pvc.CreationTimestamp = metav1.NewTime(time.Now().Add(-pendingPVCTimeoutThreshold * 2))
			pvcCache.Add(pvc)

			failed, message, err := IsPVCFailedProvisioning(pvcCache, scCache, virtClient, pvc.Namespace, pvc.Name)

			Expect(failed).To(BeTrue())
			Expect(message).ToNot(BeZero())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should detect no PVC provisioning failure when pending for less than timeout threshold", func() {
			pvc.CreationTimestamp = metav1.NewTime(time.Now().Add(-pendingPVCTimeoutThreshold / 2))
			pvcCache.Add(pvc)

			failed, message, err := IsPVCFailedProvisioning(pvcCache, scCache, virtClient, pvc.Namespace, pvc.Name)

			Expect(failed).To(BeFalse())
			Expect(message).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should detect no PVC provisioning failure when volume mode is WaitForFirstConsumer", func() {
			wffcMode := storagev1.VolumeBindingWaitForFirstConsumer
			sc := &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "teststorageclass",
				},
				VolumeBindingMode: &wffcMode,
			}
			scCache.Add(sc)

			pvc.CreationTimestamp = metav1.NewTime(time.Now().Add(-pendingPVCTimeoutThreshold * 2))
			pvc.Spec.StorageClassName = &sc.Name
			pvcCache.Add(pvc)

			failed, message, err := IsPVCFailedProvisioning(pvcCache, scCache, virtClient, pvc.Namespace, pvc.Name)

			Expect(failed).To(BeFalse())
			Expect(message).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
