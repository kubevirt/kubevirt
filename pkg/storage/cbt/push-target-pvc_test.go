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

package cbt

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Backup Target PVC with Utility Volumes", func() {
	var (
		ctrl                 *gomock.Controller
		virtClient           *kubecli.MockKubevirtClient
		vmiInterface         *kubecli.MockVirtualMachineInstanceInterface
		backupController     *VMBackupController
		testVMI              *v1.VirtualMachineInstance
		testPVCName          string
		testBackupVolumeName string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		testPVCName = "test-backup-pvc"
		testBackupVolumeName = backupTargetVolumeName("test-backup")

		pvcInformer, _ := testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})

		backupController = &VMBackupController{
			client:   virtClient,
			pvcStore: pvcInformer.GetStore(),
		}

		testVMI = libvmi.New(
			libvmi.WithNamespace("default"),
			libvmi.WithName("test-vmi"),
		)

		virtClient.EXPECT().VirtualMachineInstance(testVMI.Namespace).Return(vmiInterface).AnyTimes()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("backupTargetPVCAttached", func() {
		It("should return false when VMI is nil", func() {
			attached := backupController.backupTargetPVCAttached(nil, testBackupVolumeName)
			Expect(attached).To(BeFalse())
		})

		It("should return false when volume status doesn't exist", func() {
			testVMI.Status.VolumeStatus = []v1.VolumeStatus{}

			attached := backupController.backupTargetPVCAttached(testVMI, testBackupVolumeName)
			Expect(attached).To(BeFalse())
		})

		It("should return false when volume exists but not mounted", func() {
			testVMI.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:          testBackupVolumeName,
					HotplugVolume: &v1.HotplugVolumeStatus{},
					Phase:         v1.VolumeReady,
				},
			}

			attached := backupController.backupTargetPVCAttached(testVMI, testBackupVolumeName)
			Expect(attached).To(BeFalse())
		})

		It("should return true when volume is mounted with HotplugVolumeMounted phase", func() {
			testVMI.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:          testBackupVolumeName,
					HotplugVolume: &v1.HotplugVolumeStatus{},
					Phase:         v1.HotplugVolumeMounted,
				},
			}

			attached := backupController.backupTargetPVCAttached(testVMI, testBackupVolumeName)
			Expect(attached).To(BeTrue())
		})
	})

	Context("attachBackupTargetPVC", func() {
		It("should successfully attach utility volume with Add operation when list is empty", func() {
			testVMI.Spec.UtilityVolumes = []v1.UtilityVolume{}

			vmiInterface.EXPECT().Patch(
				context.Background(),
				testVMI.Name,
				k8stypes.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).DoAndReturn(func(ctx context.Context, name string, pt k8stypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchStr := string(data)
				Expect(patchStr).To(ContainSubstring("/spec/utilityVolumes"))
				Expect(patchStr).To(ContainSubstring("\"op\":\"add\""))
				Expect(patchStr).To(ContainSubstring("\"type\":\"Backup\""))
				return testVMI, nil
			})

			syncInfo := backupController.attachBackupTargetPVC(testVMI, testPVCName, testBackupVolumeName)
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).ToNot(HaveOccurred())
			Expect(syncInfo.event).To(Equal(backupInitializingEvent))
			Expect(syncInfo.reason).To(Equal(fmt.Sprintf(attachTargetPVCMsg, testPVCName, testVMI.Name)))
		})

		It("should successfully attach utility volume with Replace operation when list has existing volumes", func() {
			existingVolume := v1.UtilityVolume{
				Name: "existing-utility-volume",
				PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "existing-pvc",
				},
				Type: pointer.P(v1.MemoryDump),
			}
			testVMI.Spec.UtilityVolumes = []v1.UtilityVolume{existingVolume}

			vmiInterface.EXPECT().Patch(
				context.Background(),
				testVMI.Name,
				k8stypes.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).DoAndReturn(func(ctx context.Context, name string, pt k8stypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchStr := string(data)
				Expect(patchStr).To(ContainSubstring("/spec/utilityVolumes"))
				Expect(patchStr).To(ContainSubstring("\"op\":\"replace\""))
				Expect(patchStr).To(ContainSubstring("\"type\":\"Backup\""))
				return testVMI, nil
			})

			syncInfo := backupController.attachBackupTargetPVC(testVMI, testPVCName, testBackupVolumeName)
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).ToNot(HaveOccurred())
		})
	})

	Context("detachBackupTargetPVC", func() {
		It("should return nil when utilityVolumes is empty", func() {
			testVMI.Spec.UtilityVolumes = []v1.UtilityVolume{}

			syncInfo := backupController.detachBackupTargetPVC(testVMI, testBackupVolumeName)
			Expect(syncInfo).To(BeNil())
		})

		It("should remove only the backup-target-pvc volume with Replace operation", func() {
			backupVolume := v1.UtilityVolume{
				Name: testBackupVolumeName,
				PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: testPVCName,
				},
				Type: pointer.P(v1.Backup),
			}
			otherVolume := v1.UtilityVolume{
				Name: "other-utility-volume",
				PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "other-pvc",
				},
				Type: pointer.P(v1.MemoryDump),
			}

			testVMI.Spec.UtilityVolumes = []v1.UtilityVolume{backupVolume, otherVolume}

			vmiInterface.EXPECT().Patch(
				context.Background(),
				testVMI.Name,
				k8stypes.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).DoAndReturn(func(ctx context.Context, name string, pt k8stypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchStr := string(data)
				Expect(patchStr).To(ContainSubstring("\"op\":\"replace\""))
				Expect(patchStr).To(ContainSubstring("other-utility-volume"))
				// The patch will contain backup-target-pvc in the test operation, but not in the final value
				return testVMI, nil
			})

			syncInfo := backupController.detachBackupTargetPVC(testVMI, testBackupVolumeName)
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).ToNot(HaveOccurred())
			Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
			Expect(syncInfo.reason).To(Equal(fmt.Sprintf(detachTargetPVCMsg, testVMI.Name)))
		})

		It("should use Remove operation when no volumes remain after detach", func() {
			testVMI.Spec.UtilityVolumes = []v1.UtilityVolume{
				{
					Name: testBackupVolumeName,
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: testPVCName,
					},
					Type: pointer.P(v1.Backup),
				},
			}

			vmiInterface.EXPECT().Patch(
				context.Background(),
				testVMI.Name,
				k8stypes.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).DoAndReturn(func(ctx context.Context, name string, pt k8stypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchStr := string(data)
				Expect(patchStr).To(ContainSubstring("\"op\":\"remove\""))
				Expect(patchStr).To(ContainSubstring("/spec/utilityVolumes"))
				return testVMI, nil
			})

			syncInfo := backupController.detachBackupTargetPVC(testVMI, testBackupVolumeName)
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).ToNot(HaveOccurred())
			Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
			Expect(syncInfo.reason).To(Equal(fmt.Sprintf(detachTargetPVCMsg, testVMI.Name)))
		})
	})

	Context("verifyBackupTargetPVC", func() {
		It("should fail when PVC name is nil", func() {
			syncInfo := backupController.verifyBackupTargetPVC(nil, "default")
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).To(HaveOccurred())
			Expect(syncInfo.err.Error()).To(ContainSubstring("nil"))
		})

		It("should fail when PVC doesn't exist in store", func() {
			nonExistentPVC := "non-existent-pvc"
			syncInfo := backupController.verifyBackupTargetPVC(&nonExistentPVC, "default")
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.event).To(Equal(backupInitializingEvent))
			Expect(syncInfo.reason).To(Equal(fmt.Sprintf(pvcNotFoundMsg, "default", nonExistentPVC)))
		})

		It("should fail when PVC is block mode", func() {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPVCName,
					Namespace: "default",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					VolumeMode: pointer.P(corev1.PersistentVolumeBlock),
				},
			}
			backupController.pvcStore.Add(pvc)

			syncInfo := backupController.verifyBackupTargetPVC(&testPVCName, "default")
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).To(HaveOccurred())
			Expect(syncInfo.err.Error()).To(ContainSubstring("block"))
		})

		It("should succeed when PVC is filesystem mode", func() {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPVCName,
					Namespace: "default",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					VolumeMode: pointer.P(corev1.PersistentVolumeFilesystem),
				},
			}
			backupController.pvcStore.Add(pvc)

			syncInfo := backupController.verifyBackupTargetPVC(&testPVCName, "default")
			Expect(syncInfo).To(BeNil())
		})
	})

	Context("Error handling", func() {
		It("should handle patch errors during attach", func() {
			testVMI.Spec.UtilityVolumes = []v1.UtilityVolume{}

			vmiInterface.EXPECT().Patch(
				gomock.Any(),
				gomock.Any(),
				k8stypes.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).Return(nil, fmt.Errorf("attach patch failed"))

			syncInfo := backupController.attachBackupTargetPVC(testVMI, testPVCName, testBackupVolumeName)
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).To(HaveOccurred())
			Expect(syncInfo.err.Error()).To(ContainSubstring("attach patch failed"))
		})

		It("should handle patch errors during detach", func() {
			testVMI.Spec.UtilityVolumes = []v1.UtilityVolume{
				{
					Name: testBackupVolumeName,
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: testPVCName,
					},
					Type: pointer.P(v1.Backup),
				},
			}

			vmiInterface.EXPECT().Patch(
				gomock.Any(),
				gomock.Any(),
				k8stypes.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).Return(nil, fmt.Errorf("detach patch failed"))

			syncInfo := backupController.detachBackupTargetPVC(testVMI, testBackupVolumeName)
			Expect(syncInfo).NotTo(BeNil())
			Expect(syncInfo.err).To(HaveOccurred())
			Expect(syncInfo.err.Error()).To(ContainSubstring("detach patch failed"))
		})
	})
})
