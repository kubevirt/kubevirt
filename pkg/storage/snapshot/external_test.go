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

package snapshot

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	overlayContentUID = types.UID("6d6b4d8f-1d1a-4f0b-9b1f-3f5b0b0a1a11")
	overlayVMIUID     = types.UID("d8f6b4d6-4f0b-1d1a-3f5b-9b1f0b0a1a22")
)

var _ = Describe("Overlay scratch volume", func() {
	overlaySnapshot := func(deadline *metav1.Duration, override *resource.Quantity) *snapshotv1.VirtualMachineSnapshot {
		return &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{Name: "snapshot", Namespace: testNamespace},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{
				FailureDeadline:    deadline,
				OverlayScratchSize: override,
			},
		}
	}

	volumeBackup := func(name, size string, storageClass *string) snapshotv1.VolumeBackup {
		return snapshotv1.VolumeBackup{
			VolumeName: name,
			PersistentVolumeClaim: snapshotv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "pvc-" + name, Namespace: testNamespace},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: storageClass,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(size)},
					},
				},
			},
		}
	}

	overlayContent := func(volumeBackups ...snapshotv1.VolumeBackup) *snapshotv1.VirtualMachineSnapshotContent {
		var volumes []v1.Volume
		for _, backup := range volumeBackups {
			volumes = append(volumes, v1.Volume{Name: backup.VolumeName})
		}

		return &snapshotv1.VirtualMachineSnapshotContent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vmsnapshot-content",
				Namespace: testNamespace,
				UID:       overlayContentUID,
			},
			Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
				Source: snapshotv1.SourceSpec{
					VirtualMachine: &snapshotv1.VirtualMachine{
						Spec: v1.VirtualMachineSpec{
							Template: &v1.VirtualMachineInstanceTemplateSpec{
								Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes},
							},
						},
					},
				},
				VolumeBackups: volumeBackups,
			},
		}
	}

	Context("name", func() {
		It("should be derived from the content UID", func() {
			content := overlayContent()
			Expect(scratchPVCName(content)).To(Equal("snap-scratch-" + string(overlayContentUID)))
		})

		It("should not be shared by two contents of the same snapshot name", func() {
			first := overlayContent()
			second := overlayContent()
			second.UID = types.UID("a-second-content")
			Expect(scratchPVCName(first)).ToNot(Equal(scratchPVCName(second)))
		})
	})

	Context("size", func() {
		It("should be whatever the user asked for", func() {
			override := resource.MustParse("3Gi")
			size, err := calculateScratchSize(overlaySnapshot(nil, &override), overlayContent(volumeBackup("disk1", "10Gi", nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(size).To(Equal(override))
		})

		It("should be bounded by the disk capacity plus its metadata", func() {
			// 2 x 10Gi x 1.1 = 22Gi, well under what a guest could write in the five
			// minutes the default deadline gives it.
			size, err := calculateScratchSize(
				overlaySnapshot(nil, nil),
				overlayContent(volumeBackup("disk1", "10Gi", nil), volumeBackup("disk2", "10Gi", nil)),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(size.String()).To(Equal("22Gi"))
		})

		It("should be bounded by what the guest can write before the deadline", func() {
			// 300s x 125Mi/s x 2 = 73.24Gi, rounded up.
			size, err := calculateScratchSize(overlaySnapshot(nil, nil), overlayContent(volumeBackup("disk1", "1Ti", nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(size.String()).To(Equal("74Gi"))
		})

		It("should follow a deadline the user shortened", func() {
			// 60s x 125Mi/s x 2 = 14.65Gi, rounded up.
			size, err := calculateScratchSize(
				overlaySnapshot(&metav1.Duration{Duration: time.Minute}, nil),
				overlayContent(volumeBackup("disk1", "1Ti", nil)),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(size.String()).To(Equal("15Gi"))
		})

		It("should fall back to the capacity when the deadline is disabled", func() {
			size, err := calculateScratchSize(
				overlaySnapshot(&metav1.Duration{Duration: 0}, nil),
				overlayContent(volumeBackup("disk1", "1Ti", nil)),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(size.String()).To(Equal("1127Gi"))
		})

		It("should round up to a whole Gi", func() {
			size, err := calculateScratchSize(overlaySnapshot(nil, nil), overlayContent(volumeBackup("disk1", "500Mi", nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(size.String()).To(Equal("1Gi"))
		})

		It("should fail when there is nothing to snapshot", func() {
			_, err := calculateScratchSize(overlaySnapshot(nil, nil), overlayContent())
			Expect(err).To(MatchError(ContainSubstring("no volumes to snapshot")))
		})
	})

	Context("storage class", func() {
		It("should be inherited from the first disk of the VM", func() {
			first, second := "first-class", "second-class"
			content := overlayContent(volumeBackup("disk1", "10Gi", &first), volumeBackup("disk2", "10Gi", &second))
			Expect(scratchStorageClass(content)).To(HaveValue(Equal(first)))
		})

		It("should follow the VM volume order, not the backup order", func() {
			first, second := "first-class", "second-class"
			content := overlayContent(volumeBackup("disk1", "10Gi", &first), volumeBackup("disk2", "10Gi", &second))
			// The backups are built by ranging over a map, so this ordering is as
			// legitimate as the other one.
			content.Spec.VolumeBackups[0], content.Spec.VolumeBackups[1] = content.Spec.VolumeBackups[1], content.Spec.VolumeBackups[0]
			Expect(scratchStorageClass(content)).To(HaveValue(Equal(first)))
		})

		It("should be unset when no disk of the VM is backed up", func() {
			class := "some-class"
			content := overlayContent(volumeBackup("disk1", "10Gi", &class))
			content.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes = []v1.Volume{{Name: "cloudinit"}}
			Expect(scratchStorageClass(content)).To(BeNil())
		})
	})

	Context("attachment", func() {
		var (
			ctrl         *gomock.Controller
			controller   *VMSnapshotController
			vmiInterface *kubecli.MockVirtualMachineInstanceInterface
			content      *snapshotv1.VirtualMachineSnapshotContent
			vmi          *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface).AnyTimes()

			controller = &VMSnapshotController{Client: virtClient}
			content = overlayContent(volumeBackup("disk1", "10Gi", nil))
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm", Namespace: testNamespace, UID: overlayVMIUID},
			}
		})

		expectPatch := func(assert func(patch string)) {
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ metav1.PatchOptions, _ ...string) (*v1.VirtualMachineInstance, error) {
					assert(string(data))
					return vmi, nil
				})
		}

		It("should hotplug the scratch volume under its own name", func() {
			expectPatch(func(patch string) {
				// The utility volume, the PVC and the overlay directory all go by
				// the same name.
				Expect(patch).To(ContainSubstring(`"name":"` + scratchPVCName(content) + `"`))
				Expect(patch).To(ContainSubstring(`"claimName":"` + scratchPVCName(content) + `"`))
				Expect(patch).To(ContainSubstring(`"type":"SnapshotOverlay"`))
			})

			Expect(controller.attachScratchVolume(vmi, content)).To(Succeed())
		})

		It("should unplug the scratch volume", func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{
				Name: scratchPVCName(content),
				Type: pointer.P(v1.SnapshotOverlay),
			}}

			expectPatch(func(patch string) {
				Expect(patch).To(ContainSubstring(`"op":"remove"`))
			})

			Expect(controller.detachScratchVolume(vmi, content)).To(Succeed())
		})

		It("should report the scratch volume as attached once it is mounted", func() {
			Expect(scratchVolumeAttached(vmi, content)).To(BeFalse())

			vmi.Status.VolumeStatus = []v1.VolumeStatus{{
				Name:          scratchPVCName(content),
				HotplugVolume: &v1.HotplugVolumeStatus{},
				Phase:         v1.HotplugVolumeMounted,
			}}
			Expect(scratchVolumeAttached(vmi, content)).To(BeTrue())
		})

		It("should not report the scratch volume as detached while the VMI still has it", func() {
			vmi.Status.VolumeStatus = []v1.VolumeStatus{{Name: scratchPVCName(content)}}
			Expect(scratchVolumeDetached(vmi, content)).To(BeFalse())

			vmi.Status.VolumeStatus = nil
			Expect(scratchVolumeDetached(vmi, content)).To(BeTrue())
		})
	})

	Context("lifecycle", func() {
		var (
			ctrl       *gomock.Controller
			controller *VMSnapshotController
			k8sClient  *k8sfake.Clientset
			recorder   *record.FakeRecorder
			content    *snapshotv1.VirtualMachineSnapshotContent
			vmSnapshot *snapshotv1.VirtualMachineSnapshot
			vmi        *v1.VirtualMachineInstance
			storeAdd   func(*corev1.PersistentVolumeClaim)
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

			pvcInformer, _ := testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
			recorder = record.NewFakeRecorder(10)

			controller = &VMSnapshotController{
				Client:      virtClient,
				PVCInformer: pvcInformer,
				Recorder:    recorder,
			}

			storeAdd = func(pvc *corev1.PersistentVolumeClaim) {
				Expect(pvcInformer.GetStore().Add(pvc)).To(Succeed())
			}

			content = overlayContent(volumeBackup("disk1", "10Gi", pointer.P("golden-class")))
			vmSnapshot = overlaySnapshot(nil, nil)
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm", Namespace: testNamespace, UID: overlayVMIUID},
			}
		})

		It("should create the volume the overlays live on", func() {
			pvc, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(pvc.Name).To(Equal(scratchPVCName(content)))
			Expect(pvc.Namespace).To(Equal(testNamespace))
			Expect(pvc.Annotations).To(HaveKeyWithValue(overlayOwnerVMIUIDAnnotation, string(overlayVMIUID)))
			Expect(pvc.Finalizers).To(ConsistOf(overlayScratchFinalizer))
			Expect(pvc.Spec.StorageClassName).To(HaveValue(Equal("golden-class")))
			// The overlays are qcow2 files in a directory, whatever the disks they
			// shadow are.
			Expect(pvc.Spec.VolumeMode).To(HaveValue(Equal(corev1.PersistentVolumeFilesystem)))
			Expect(pvc.Spec.AccessModes).To(ConsistOf(corev1.ReadWriteOnce))
			Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal("11Gi"))

			Expect(pvc.OwnerReferences).To(ConsistOf(metav1.OwnerReference{
				APIVersion:         snapshotv1.SchemeGroupVersion.String(),
				Kind:               "VirtualMachineSnapshotContent",
				Name:               content.Name,
				UID:                content.UID,
				Controller:         pointer.P(true),
				BlockOwnerDeletion: pointer.P(true),
			}))

			testutils.ExpectEvent(recorder, scratchPVCCreateEvent)
		})

		It("should return the existing volume when it is already there", func() {
			existing, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectEvent(recorder, scratchPVCCreateEvent)
			storeAdd(existing)

			again, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(again).To(Equal(existing))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should not create a volume it cannot size", func() {
			k8sClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(testing.Action) (bool, runtime.Object, error) {
				Fail("the volume was created without a size")
				return true, nil, nil
			})

			_, err := controller.createScratchPVC(vmSnapshot, overlayContent(), vmi)
			Expect(err).To(MatchError(ContainSubstring("no volumes to snapshot")))
		})

		It("should report a volume it could not create", func() {
			k8sClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(testing.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("exceeded quota")
			})

			_, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).To(MatchError(ContainSubstring("exceeded quota")))
		})

		It("should find the volume of a snapshot", func() {
			pvc, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectEvent(recorder, scratchPVCCreateEvent)
			storeAdd(pvc)

			found, err := controller.getScratchPVC(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(Equal(pvc))
		})

		It("should not find a volume of a snapshot that has none", func() {
			found, err := controller.getScratchPVC(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeNil())
		})

		It("should release and delete the volume", func() {
			pvc, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectEvent(recorder, scratchPVCCreateEvent)

			Expect(controller.deleteScratchPVC(pvc)).To(Succeed())

			_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})

		It("should only release a volume somebody else is already deleting", func() {
			pvc, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectEvent(recorder, scratchPVCCreateEvent)

			// The deletion request is held by our finalizer, so dropping it is all
			// that is left to do and a second delete would be noise.
			pvc.DeletionTimestamp = pointer.P(metav1.Now())
			k8sClient.Fake.PrependReactor("delete", "persistentvolumeclaims", func(testing.Action) (bool, runtime.Object, error) {
				Fail("a volume that is already being deleted was deleted again")
				return true, nil, nil
			})

			Expect(controller.deleteScratchPVC(pvc)).To(Succeed())

			updated, err := k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Finalizers).To(BeEmpty())
		})

		It("should tolerate a volume that is already gone", func() {
			pvc, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectEvent(recorder, scratchPVCCreateEvent)

			Expect(controller.deleteScratchPVC(pvc)).To(Succeed())
			pvc.Finalizers = nil
			Expect(controller.deleteScratchPVC(pvc)).To(Succeed())
		})

		It("should report a volume it could not release", func() {
			pvc, err := controller.createScratchPVC(vmSnapshot, content, vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectEvent(recorder, scratchPVCCreateEvent)

			k8sClient.Fake.PrependReactor("update", "persistentvolumeclaims", func(testing.Action) (bool, runtime.Object, error) {
				return true, nil, k8serrors.NewConflict(
					schema.GroupResource{Resource: "persistentvolumeclaims"}, pvc.Name, fmt.Errorf("stale"))
			})
			k8sClient.Fake.PrependReactor("delete", "persistentvolumeclaims", func(testing.Action) (bool, runtime.Object, error) {
				Fail("a volume that is still held by its finalizer was deleted")
				return true, nil, nil
			})

			Expect(controller.deleteScratchPVC(pvc)).To(MatchError(ContainSubstring("failed to release")))
		})
	})
})
