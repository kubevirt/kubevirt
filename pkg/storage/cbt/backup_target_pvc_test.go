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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	backupv1 "kubevirt.io/api/backup/v1alpha1"

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
		testPVCName          string = "test-backup-pvc"
		testBackupVolumeName string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

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

	DescribeTable("backupTargetPVCAttached",
		func(setup func(*v1.VirtualMachineInstance), vmiNil, expected bool) {
			var vmi *v1.VirtualMachineInstance
			if !vmiNil {
				vmi = testVMI
				if setup != nil {
					setup(vmi)
				}
			}
			Expect(backupController.backupTargetPVCAttached(vmi, testBackupVolumeName)).To(Equal(expected))
		},
		Entry("false when VMI is nil", nil, true, false),
		Entry("false when volume status doesn't exist", func(vmi *v1.VirtualMachineInstance) {
			vmi.Status.VolumeStatus = []v1.VolumeStatus{}
		}, false, false),
		Entry("false when volume exists but not mounted", func(vmi *v1.VirtualMachineInstance) {
			vmi.Status.VolumeStatus = []v1.VolumeStatus{{
				Name:          testBackupVolumeName,
				HotplugVolume: &v1.HotplugVolumeStatus{},
				Phase:         v1.VolumeReady,
			}}
		}, false, false),
		Entry("true when volume is HotplugVolumeMounted", func(vmi *v1.VirtualMachineInstance) {
			vmi.Status.VolumeStatus = []v1.VolumeStatus{{
				Name:          testBackupVolumeName,
				HotplugVolume: &v1.HotplugVolumeStatus{},
				Phase:         v1.HotplugVolumeMounted,
			}}
		}, false, true),
	)

	DescribeTable("attachBackupTargetPVC",
		func(existingVolumes []v1.UtilityVolume, expectedOp string) {
			testVMI.Spec.UtilityVolumes = existingVolumes

			vmiInterface.EXPECT().Patch(
				context.Background(),
				testVMI.Name,
				types.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).DoAndReturn(func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchStr := string(data)
				Expect(patchStr).To(ContainSubstring("/spec/utilityVolumes"))
				Expect(patchStr).To(ContainSubstring(fmt.Sprintf("\"op\":\"%s\"", expectedOp)))
				Expect(patchStr).To(ContainSubstring("\"type\":\"Backup\""))
				return testVMI, nil
			})

			Expect(backupController.attachBackupTargetPVC(testVMI, testPVCName, testBackupVolumeName)).To(Succeed())
		},
		Entry("Add when utilityVolumes is empty", []v1.UtilityVolume{}, "add"),
		Entry("Replace when utilityVolumes already has volumes", []v1.UtilityVolume{{
			Name: "existing-utility-volume",
			PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: "existing-pvc",
			},
			Type: pointer.P(v1.MemoryDump),
		}}, "replace"),
	)

	DescribeTable("detachBackupTargetPVC",
		func(setupVolumes func(backupVolumeName string) []v1.UtilityVolume, expectPatch bool, expectedOp string) {
			testVMI.Spec.UtilityVolumes = setupVolumes(testBackupVolumeName)

			if expectPatch {
				vmiInterface.EXPECT().Patch(
					context.Background(),
					testVMI.Name,
					types.JSONPatchType,
					gomock.Any(),
					gomock.Any(),
				).DoAndReturn(func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
					patchStr := string(data)
					Expect(patchStr).To(ContainSubstring(fmt.Sprintf("\"op\":\"%s\"", expectedOp)))
					Expect(patchStr).To(ContainSubstring("/spec/utilityVolumes"))
					return testVMI, nil
				})
			}

			Expect(backupController.detachBackupTargetPVC(testVMI, testBackupVolumeName)).To(Succeed())
		},
		Entry("no-op when utilityVolumes is empty",
			func(string) []v1.UtilityVolume { return []v1.UtilityVolume{} }, false, ""),
		Entry("Replace when other volumes remain",
			func(backupVolumeName string) []v1.UtilityVolume {
				return []v1.UtilityVolume{
					{
						Name: backupVolumeName,
						PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: testPVCName,
						},
						Type: pointer.P(v1.Backup),
					},
					{
						Name: "other-utility-volume",
						PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "other-pvc",
						},
						Type: pointer.P(v1.MemoryDump),
					},
				}
			}, true, "replace"),
		Entry("Remove when no volumes remain",
			func(backupVolumeName string) []v1.UtilityVolume {
				return []v1.UtilityVolume{{
					Name: backupVolumeName,
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: testPVCName,
					},
					Type: pointer.P(v1.Backup),
				}}
			}, true, "remove"),
	)

	DescribeTable("verifyBackupTargetPVC",
		func(pvcNameFn func() *string, pvc *corev1.PersistentVolumeClaim, expectErrSubstring string, expectNotFound bool) {
			if pvc != nil {
				Expect(backupController.pvcStore.Add(pvc)).To(Succeed())
			}
			err := backupController.verifyBackupTargetPVC(pvcNameFn(), "default")
			if expectErrSubstring == "" && !expectNotFound {
				Expect(err).ToNot(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectErrSubstring))
			Expect(isBackupTargetPVCNotFound(err)).To(Equal(expectNotFound))
		},
		Entry("fail when PVC name is nil", func() *string { return nil }, nil, "empty", false),
		Entry("return not-found when PVC doesn't exist", func() *string { return pointer.P("non-existent-pvc") }, nil,
			fmt.Sprintf(pvcNotFoundMsg, "default", "non-existent-pvc"), true),
		Entry("fail when PVC is block mode", func() *string { return &testPVCName }, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: testPVCName, Namespace: "default"},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeMode: pointer.P(corev1.PersistentVolumeBlock)},
		}, "block", false),
		Entry("succeed when PVC is filesystem mode", func() *string { return &testPVCName }, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: testPVCName, Namespace: "default"},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeMode: pointer.P(corev1.PersistentVolumeFilesystem)},
		}, "", false),
	)

	DescribeTable("Error handling",
		func(setup func(vmi *v1.VirtualMachineInstance, backupVolumeName, pvcName string), callAttach bool, errSubstring string) {
			setup(testVMI, testBackupVolumeName, testPVCName)
			vmiInterface.EXPECT().Patch(
				gomock.Any(),
				gomock.Any(),
				types.JSONPatchType,
				gomock.Any(),
				gomock.Any(),
			).Return(nil, fmt.Errorf("%s", errSubstring))

			var err error
			if callAttach {
				err = backupController.attachBackupTargetPVC(testVMI, testPVCName, testBackupVolumeName)
			} else {
				err = backupController.detachBackupTargetPVC(testVMI, testBackupVolumeName)
			}
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errSubstring))
		},
		Entry("attach patch errors",
			func(vmi *v1.VirtualMachineInstance, _, _ string) { vmi.Spec.UtilityVolumes = []v1.UtilityVolume{} },
			true,
			"attach patch failed",
		),
		Entry("detach patch errors",
			func(vmi *v1.VirtualMachineInstance, backupVolumeName, pvcName string) {
				vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{
					Name: backupVolumeName,
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
					Type: pointer.P(v1.Backup),
				}}
			},
			false,
			"detach patch failed",
		),
	)

	DescribeTable("bootCandidateCBTVolume",
		func(disks []v1.Disk, volumeNames []string, expected string) {
			opts := []libvmi.Option{
				libvmi.WithNamespace("default"),
				libvmi.WithName("test-vmi"),
			}
			for _, name := range volumeNames {
				opts = append(opts, libvmi.WithPersistentVolumeClaim(name, name+"-pvc"))
			}
			vmi := libvmi.New(opts...)
			if disks != nil {
				vmi.Spec.Domain.Devices.Disks = disks
			}
			name, err := bootCandidateCBTVolume(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal(expected))
		},
		Entry("lowest bootOrder CBT disk",
			[]v1.Disk{
				{Name: "data", BootOrder: pointer.P(uint(2))},
				{Name: "root", BootOrder: pointer.P(uint(1))},
			},
			[]string{"data", "root"},
			"root",
		),
		Entry("first CBT-eligible volume when no bootOrder is set",
			nil,
			[]string{"disk0", "disk1"},
			"disk0",
		),
	)

	DescribeTable("pvcStorageSize",
		func(request, capacity, expected string) {
			pvc := &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{},
					},
				},
			}
			if request != "" {
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(request)
			}
			if capacity != "" {
				pvc.Status.Capacity = corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(capacity),
				}
			}
			size := pvcStorageSize(pvc)
			Expect(size).ToNot(BeNil())
			Expect(size.Cmp(resource.MustParse(expected))).To(Equal(0))
		},
		Entry("min(request, capacity) when capacity is larger like hostpath", "4Gi", "100Gi", "4Gi"),
		Entry("fall back to request when capacity is missing", "2Gi", "", "2Gi"),
		Entry("use capacity when smaller than request", "10Gi", "8Gi", "8Gi"),
	)

	DescribeTable("backupTargetPVCSizePercent",
		func(specPercent *int, expected int) {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					TargetPvcSizePercent: specPercent,
				},
			}
			Expect(backupTargetPVCSizePercent(backup)).To(Equal(expected))
		},
		Entry("defaults to 120 when unset", nil, 120),
		Entry("uses explicit value", pointer.P(50), 50),
	)

	DescribeTable("validateBackupTargetPVCSizePercent",
		func(percent int, expectErr bool) {
			err := validateBackupTargetPVCSizePercent(percent)
			if expectErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).ToNot(HaveOccurred())
		},
		Entry("accepts minimum", 1, false),
		Entry("accepts maximum", 1000, false),
		Entry("rejects below minimum", 0, true),
		Entry("rejects above maximum", 1001, true),
	)

	DescribeTable("calculateBackupTargetPVCSize",
		func(sourceSize string, percent int, expected string) {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("test-vmi"),
				libvmi.WithPersistentVolumeClaim("disk0", "disk0-pvc"),
			)
			sourcePVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "disk0-pvc", Namespace: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(sourceSize),
						},
					},
				},
			}
			Expect(backupController.pvcStore.Add(sourcePVC)).To(Succeed())

			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					TargetPvcSizePercent: pointer.P(percent),
				},
			}
			size, err := backupController.calculateBackupTargetPVCSize(backup, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(size.Cmp(resource.MustParse(expected))).To(Equal(0))
		},
		Entry("default 120 percent of 10Gi", "10Gi", 120, "12Gi"),
		Entry("custom 50 percent of 1Gi", "1Gi", 50, "512Mi"),
		Entry("ceil rounds up partial bytes", "3Gi", 50, "1536Mi"),
	)

	DescribeTable("validateReusableAutoBackupTargetPVC",
		func(pvc *corev1.PersistentVolumeClaim, expectErrSubstring string) {
			backup := &backupv1.VirtualMachineBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup",
					Namespace: "default",
					UID:       "backup-uid",
				},
			}
			err := validateReusableAutoBackupTargetPVC(backup, pvc)
			if expectErrSubstring != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectErrSubstring))
				return
			}
			Expect(err).ToNot(HaveOccurred())
		},
		Entry("succeed when owned FS PVC", &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "owned-pvc",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(&backupv1.VirtualMachineBackup{
						ObjectMeta: metav1.ObjectMeta{Name: "test-backup", UID: "backup-uid"},
					}, backupv1.SchemeGroupVersion.WithKind(
						backupv1.VirtualMachineBackupGroupVersionKind.Kind)),
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{VolumeMode: pointer.P(corev1.PersistentVolumeFilesystem)},
		}, ""),
		Entry("fail when PVC is block mode", &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "block-pvc",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(&backupv1.VirtualMachineBackup{
						ObjectMeta: metav1.ObjectMeta{Name: "test-backup", UID: "backup-uid"},
					}, backupv1.SchemeGroupVersion.WithKind(
						backupv1.VirtualMachineBackupGroupVersionKind.Kind)),
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{VolumeMode: pointer.P(corev1.PersistentVolumeBlock)},
		}, "block"),
		Entry("fail when PVC is not owned by backup", &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foreign-pvc",
				Namespace: "default",
			},
			Spec: corev1.PersistentVolumeClaimSpec{VolumeMode: pointer.P(corev1.PersistentVolumeFilesystem)},
		}, "not owned"),
	)

	DescribeTable("storageClassFromPVC",
		func(pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, expectSC, expectErrSubstring string) {
			k8sClient := fake.NewSimpleClientset()
			if pv != nil {
				_, err := k8sClient.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

			sc, err := backupController.storageClassFromPVC(pvc)
			if expectErrSubstring != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectErrSubstring))
				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(sc).To(Equal(expectSC))
		},
		Entry("use PVC storageClassName when set",
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "disk", Namespace: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: pointer.P("pvc-sc"),
				},
			},
			nil,
			"pvc-sc",
			"",
		),
		Entry("fail when unbound and no storageClassName",
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "disk", Namespace: "default"},
				Spec:       corev1.PersistentVolumeClaimSpec{},
			},
			nil,
			"",
			"not bound",
		),
		Entry("resolve storageClassName from bound PV",
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "disk", Namespace: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					VolumeName: "disk-pv",
				},
			},
			&corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{Name: "disk-pv"},
				Spec: corev1.PersistentVolumeSpec{
					StorageClassName: "pv-sc",
				},
			},
			"pv-sc",
			"",
		),
		Entry("fail when bound PV has no storageClassName",
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "disk", Namespace: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					VolumeName: "disk-pv",
				},
			},
			&corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{Name: "disk-pv"},
				Spec:       corev1.PersistentVolumeSpec{},
			},
			"",
			"has no storageClassName",
		),
	)
})
