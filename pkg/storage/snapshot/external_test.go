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
	"k8s.io/client-go/util/workqueue"

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
						ObjectMeta: metav1.ObjectMeta{Name: "vm", Namespace: testNamespace},
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

	Context("mode", func() {
		It("should be direct unless the snapshot asked for something else", func() {
			Expect(snapshotMode(nil)).To(Equal(snapshotv1.SnapshotModeDirect))
			Expect(snapshotMode(overlaySnapshot(nil, nil))).To(Equal(snapshotv1.SnapshotModeDirect))

			external := overlaySnapshot(nil, nil)
			external.Spec.SnapshotMode = pointer.P(snapshotv1.SnapshotModeExternal)
			Expect(snapshotMode(external)).To(Equal(snapshotv1.SnapshotModeExternal))
		})

		It("should be read off the content, which outlives the snapshot", func() {
			content := overlayContent()
			Expect(externalMode(content)).To(BeFalse())

			content.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{}
			Expect(externalMode(content)).To(BeFalse())

			content.Status.SnapshotMode = pointer.P(snapshotv1.SnapshotModeExternal)
			Expect(externalMode(content)).To(BeTrue())
		})
	})

	Context("ownership", func() {
		scratch := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{overlayOwnerVMIUIDAnnotation: string(overlayVMIUID)},
			},
		}

		It("should recognise the launcher that made the overlays", func() {
			vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: overlayVMIUID}}
			Expect(overlaysOwnedByRunningVMI(scratch, vmi)).To(BeTrue())
		})

		It("should not recognise a launcher that never saw them", func() {
			vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "a-newer-vmi"}}
			Expect(overlaysOwnedByRunningVMI(scratch, vmi)).To(BeFalse())
			Expect(overlaysOwnedByRunningVMI(scratch, nil)).To(BeFalse())
		})
	})

	Context("flow", func() {
		var (
			ctrl         *gomock.Controller
			controller   *VMSnapshotController
			vmiInterface *kubecli.MockVirtualMachineInstanceInterface
			k8sClient    *k8sfake.Clientset
			recorder     *record.FakeRecorder
			content      *snapshotv1.VirtualMachineSnapshotContent
			vmSnapshot   *snapshotv1.VirtualMachineSnapshot
			vmi          *v1.VirtualMachineInstance
			addVMI       func()
			addScratch   func()
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface).AnyTimes()
			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

			pvcInformer, _ := testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			recorder = record.NewFakeRecorder(10)

			controller = &VMSnapshotController{
				Client:      virtClient,
				PVCInformer: pvcInformer,
				VMIInformer: vmiInformer,
				Recorder:    recorder,
			}

			vmSnapshot = overlaySnapshot(nil, nil)
			vmSnapshot.Spec.SnapshotMode = pointer.P(snapshotv1.SnapshotModeExternal)

			content = overlayContent(volumeBackup("disk1", "10Gi", nil))
			content.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
				SnapshotMode: pointer.P(snapshotv1.SnapshotModeExternal),
			}

			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm", Namespace: testNamespace, UID: overlayVMIUID},
				Status: v1.VirtualMachineInstanceStatus{
					Conditions: []v1.VirtualMachineInstanceCondition{{
						Type:   v1.VirtualMachineInstanceAgentConnected,
						Status: corev1.ConditionTrue,
					}},
				},
			}

			addVMI = func() {
				Expect(vmiInformer.GetStore().Add(vmi)).To(Succeed())
			}

			addScratch = func() {
				pvc := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:        scratchPVCName(content),
						Namespace:   testNamespace,
						Annotations: map[string]string{overlayOwnerVMIUIDAnnotation: string(overlayVMIUID)},
						Finalizers:  []string{overlayScratchFinalizer},
					},
				}
				_, err := k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvcInformer.GetStore().Add(pvc)).To(Succeed())
			}
		})

		// The scratch volume hotplugged and mounted, which the launcher needs
		// before it can be asked for anything.
		mountScratch := func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{
				Name: scratchPVCName(content),
				Type: pointer.P(v1.SnapshotOverlay),
			}}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{{
				Name:          scratchPVCName(content),
				HotplugVolume: &v1.HotplugVolumeStatus{},
				Phase:         v1.HotplugVolumeMounted,
			}}
		}

		reportPhase := func(status corev1.ConditionStatus, reason, message string) {
			vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
				Type:    v1.VirtualMachineInstanceOverlaySnapshotActive,
				Status:  status,
				Reason:  reason,
				Message: message,
			})
		}

		expectPatch := func() {
			vmiInterface.EXPECT().
				Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmi, nil)
		}

		overlayOptions := func() *v1.SnapshotOverlayOptions {
			return &v1.SnapshotOverlayOptions{VolumeName: scratchPVCName(content)}
		}

		Context("creating the disk overlays", func() {
			It("should refuse a source with no guest agent", func() {
				vmi.Status.Conditions = nil
				addVMI()

				_, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).To(MatchError(ContainSubstring("guest agent")))
				Expect(k8sClient.Actions()).To(BeEmpty())
			})

			It("should refuse a source that is not running", func() {
				_, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).To(MatchError(ContainSubstring("guest agent")))
				Expect(k8sClient.Actions()).To(BeEmpty())
			})

			It("should create the scratch volume first", func() {
				addVMI()

				prepared, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).ToNot(HaveOccurred())
				Expect(prepared).To(BeFalse())
				testutils.ExpectEvent(recorder, scratchPVCCreateEvent)

				_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).
					Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should hotplug the volume once it is there", func() {
				addVMI()
				addScratch()
				expectPatch()

				prepared, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).ToNot(HaveOccurred())
				Expect(prepared).To(BeFalse())
			})

			It("should ask the launcher once the volume is mounted", func() {
				mountScratch()
				addVMI()
				addScratch()
				vmiInterface.EXPECT().ExternalSnapshot(context.Background(), vmi.Name, overlayOptions()).Return(nil)

				prepared, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).ToNot(HaveOccurred())
				Expect(prepared).To(BeFalse())
			})

			It("should wait while the launcher is working", func() {
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayPreparing, "")
				mountScratch()
				addVMI()
				addScratch()

				prepared, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).ToNot(HaveOccurred())
				Expect(prepared).To(BeFalse())
			})

			It("should report ready once every disk is on an overlay", func() {
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlaysReady, "")
				mountScratch()
				addVMI()
				addScratch()

				prepared, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).ToNot(HaveOccurred())
				Expect(prepared).To(BeTrue())
			})

			It("should report a transaction that failed as a lost capture", func() {
				reportPhase(corev1.ConditionFalse, v1.VirtualMachineInstanceReasonOverlaySnapshotFailed, "the guest would not quiesce")
				mountScratch()
				addVMI()
				addScratch()

				_, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).To(MatchError(errCaptureLost))
				Expect(err).To(MatchError(ContainSubstring("the guest would not quiesce")))
			})

			It("should not create a second set of overlays under volumes already captured", func() {
				// One snapshot made of two points in time is not a snapshot.
				content.Status.VolumeSnapshotStatus = []snapshotv1.VolumeSnapshotStatus{{VolumeSnapshotName: "vs-disk1"}}
				mountScratch()
				addVMI()
				addScratch()

				_, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).To(MatchError(errCaptureLost))
				Expect(err).To(MatchError(ContainSubstring("merged before every volume was captured")))
			})

			// The caller removes the scratch volume on errCaptureLost, this
			// function only reports it.
			It("should leave the scratch volume alone when the capture is lost", func() {
				content.Status.VolumeSnapshotStatus = []snapshotv1.VolumeSnapshotStatus{{VolumeSnapshotName: "vs-disk1"}}
				mountScratch()
				addVMI()
				addScratch()

				_, err := controller.createDiskOverlays(vmSnapshot, content)
				Expect(err).To(MatchError(errCaptureLost))

				_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).
					Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("removing the scratch volume", func() {
			It("should take it off the VMI before deleting it", func() {
				mountScratch()
				addVMI()
				addScratch()
				expectPatch()

				done, err := controller.removeScratchVolume(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeFalse())

				_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).
					Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should delete it once it is off the VMI", func() {
				addVMI()
				addScratch()

				done, err := controller.removeScratchVolume(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeTrue())

				_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).
					Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			})

			It("should report done when there is no volume left", func() {
				addVMI()

				done, err := controller.removeScratchVolume(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeTrue())
			})
		})

		Context("committing the overlays", func() {
			It("should have nothing to do without a scratch volume", func() {
				addVMI()

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(BeZero())
				Expect(captured).To(BeFalse())
			})

			It("should do nothing when the launcher is gone", func() {
				addScratch()

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(BeZero())
				Expect(captured).To(BeFalse())
			})

			It("should do nothing when the overlays belong to a different launcher", func() {
				vmi.UID = "a-newer-vmi"
				addVMI()
				addScratch()

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(BeZero())
				Expect(captured).To(BeFalse())
			})

			It("should wait for every volume snapshot to be readyToUse", func() {
				// Created is not readyToUse, and a VolumeSnapshot still reading a
				// base image needs it to stay still.
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlaysReady, "")
				mountScratch()
				addVMI()
				addScratch()

				requeue, captured, err := controller.commitOverlays(content, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(Equal(snapshotRetryInterval))
				Expect(captured).To(BeFalse())
			})

			It("should record the capture before asking for the commit", func() {
				// A commit requested first and a status update that then failed
				// would leave nothing to say the capture was good.
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlaysReady, "")
				mountScratch()
				addVMI()
				addScratch()

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(Equal(snapshotRetryInterval))
				Expect(captured).To(BeTrue())
			})

			It("should commit once the capture is recorded", func() {
				content.Status.CreationTime = currentTime()
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlaysReady, "")
				mountScratch()
				addVMI()
				addScratch()
				vmiInterface.EXPECT().CommitSnapshot(context.Background(), vmi.Name, overlayOptions()).Return(nil)

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(Equal(snapshotRetryInterval))
				Expect(captured).To(BeTrue())
			})

			It("should wait for a commit that is already running", func() {
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayCommitting, "")
				mountScratch()
				addVMI()
				addScratch()

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(Equal(snapshotRetryInterval))
				Expect(captured).To(BeFalse())
			})

			It("should retry a commit that failed, and not call it a capture", func() {
				// The commit runs disk by disk, so a failure part way through
				// already pivoted the disks before it onto base images.
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayCommitFailed, "the block job stalled")
				mountScratch()
				addVMI()
				addScratch()
				vmiInterface.EXPECT().CommitSnapshot(context.Background(), vmi.Name, overlayOptions()).Return(nil)

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(Equal(snapshotRetryInterval))
				Expect(captured).To(BeFalse())
				testutils.ExpectEvent(recorder, overlayCommitFailedEvent)
			})

			It("should unplug the volume before deleting it", func() {
				// A PVC a pod still mounts hangs on pvc-protection.
				mountScratch()
				addVMI()
				addScratch()
				expectPatch()

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(Equal(snapshotRetryInterval))
				Expect(captured).To(BeFalse())

				_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).
					Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should delete the volume once it is off the VMI", func() {
				addVMI()
				addScratch()

				requeue, captured, err := controller.commitOverlays(content, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(BeZero())
				Expect(captured).To(BeFalse())

				_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).
					Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
				Expect(err).To(MatchError(ContainSubstring("not found")))
			})
		})

		Context("spotting overlays that went early", func() {
			// The capture is over and cannot be retried, a second set of overlays
			// would span two points in time.
			capturing := func() {
				content.Status.VolumeSnapshotStatus = []snapshotv1.VolumeSnapshotStatus{
					{VolumeSnapshotName: "vs-disk1"},
				}
			}

			It("should ignore a content that is not External", func() {
				content.Status.SnapshotMode = pointer.P(snapshotv1.SnapshotModeDirect)
				capturing()
				addVMI()

				Expect(controller.overlaysGoneEarly(content)).To(BeFalse())
			})

			It("should ignore a content whose overlays never opened", func() {
				addVMI()

				Expect(controller.overlaysGoneEarly(content)).To(BeFalse())
			})

			It("should ignore a content that already captured", func() {
				content.Status.CreationTime = currentTime()
				capturing()
				addVMI()

				Expect(controller.overlaysGoneEarly(content)).To(BeFalse())
			})

			It("should accept overlays that are still holding the base images", func() {
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlaysReady, "")
				capturing()
				addVMI()

				Expect(controller.overlaysGoneEarly(content)).To(BeFalse())
			})

			DescribeTable("should report a window that is no longer protecting the base images",
				func(status corev1.ConditionStatus, reason string) {
					reportPhase(status, reason, "")
					capturing()
					addVMI()

					Expect(controller.overlaysGoneEarly(content)).To(BeTrue())
				},
				Entry("a commit in flight", corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayCommitting),
				Entry("a commit that failed part way through", corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayCommitFailed),
				Entry("overlays reported gone", corev1.ConditionFalse, v1.VirtualMachineInstanceReasonOverlaySnapshotFailed),
			)

			It("should report a launcher that says nothing at all", func() {
				capturing()
				addVMI()

				Expect(controller.overlaysGoneEarly(content)).To(BeTrue())
			})

			It("should report a source that is gone", func() {
				capturing()

				Expect(controller.overlaysGoneEarly(content)).To(BeTrue())
			})

			It("should fail the snapshot without waiting for its deadline", func() {
				capturing()
				addVMI()
				vmSnapshot.CreationTimestamp = metav1.Now()
				vmSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{Phase: snapshotv1.InProgress}

				reason, failed, err := controller.vmSnapshotFailure(vmSnapshot, content)
				Expect(err).ToNot(HaveOccurred())
				Expect(failed).To(BeTrue())
				Expect(reason).To(Equal(vmSnapshotOverlaysGoneEarlyError))
			})

			It("should keep the reason the snapshot first failed with", func() {
				// Failed is sticky, and every later pass goes through the deadline,
				// which would otherwise relabel this a timeout.
				vmSnapshot.Spec.FailureDeadline = &metav1.Duration{Duration: -time.Minute}
				vmSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					Phase: snapshotv1.Failed,
					Conditions: []snapshotv1.Condition{
						newFailureCondition(corev1.ConditionTrue, vmSnapshotOverlaysGoneEarlyError),
					},
				}

				reason, failed, err := controller.vmSnapshotFailure(vmSnapshot, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(failed).To(BeTrue())
				Expect(reason).To(Equal(vmSnapshotOverlaysGoneEarlyError))
			})
		})

		Context("cleaning up", func() {
			It("should have nothing to wind down for a direct snapshot", func() {
				content.Status.SnapshotMode = nil
				addVMI()
				addScratch()

				done, err := controller.cleanupOverlays(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeTrue())
			})

			It("should have nothing to wind down without a scratch volume", func() {
				addVMI()

				done, err := controller.cleanupOverlays(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeTrue())
			})

			It("should not wind down when the launcher is gone", func() {
				addScratch()

				done, err := controller.cleanupOverlays(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeFalse())
			})

			It("should commit whatever the volume snapshots are doing", func() {
				// The gate on readyToUse protects an artifact being thrown away.
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlaysReady, "")
				mountScratch()
				addVMI()
				addScratch()
				vmiInterface.EXPECT().CommitSnapshot(context.Background(), vmi.Name, overlayOptions()).Return(nil)

				done, err := controller.cleanupOverlays(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeFalse())
			})

			It("should wait for a commit that is already running", func() {
				reportPhase(corev1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayCommitting, "")
				mountScratch()
				addVMI()
				addScratch()

				done, err := controller.cleanupOverlays(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeFalse())
			})

			It("should finish once the volume is released", func() {
				addVMI()
				addScratch()

				done, err := controller.cleanupOverlays(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeTrue())

				_, err = k8sClient.CoreV1().PersistentVolumeClaims(testNamespace).
					Get(context.Background(), scratchPVCName(content), metav1.GetOptions{})
				Expect(err).To(MatchError(ContainSubstring("not found")))
			})
		})
	})

	Context("wake-up", func() {
		var (
			controller *VMSnapshotController
			vmSnapshot *snapshotv1.VirtualMachineSnapshot
		)

		BeforeEach(func() {
			vmSnapshotInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
			controller = &VMSnapshotController{
				VMSnapshotInformer: vmSnapshotInformer,
				vmSnapshotContentQueue: workqueue.NewTypedRateLimitingQueue[string](
					workqueue.DefaultTypedControllerRateLimiter[string](),
				),
			}

			vmSnapshot = overlaySnapshot(nil, nil)
			vmSnapshot.UID = "snapshot-uid"
			DeferCleanup(func() { controller.vmSnapshotContentQueue.ShutDown() })
		})

		add := func() {
			Expect(controller.VMSnapshotInformer.GetStore().Add(vmSnapshot)).To(Succeed())
			controller.enqueueExternalContent(cacheKeyFunc(testNamespace, vmSnapshot.Name))
		}

		It("should wake the content of an external snapshot on a change to its VMI", func() {
			// The VMI is where the launcher reports how far it has got, and the
			// content acts on it.
			vmSnapshot.Spec.SnapshotMode = pointer.P(snapshotv1.SnapshotModeExternal)
			add()

			Expect(controller.vmSnapshotContentQueue.Len()).To(Equal(1))
			key, _ := controller.vmSnapshotContentQueue.Get()
			Expect(key).To(Equal(cacheKeyFunc(testNamespace, GetVMSnapshotContentName(vmSnapshot))))
		})

		It("should leave a direct snapshot alone", func() {
			add()
			Expect(controller.vmSnapshotContentQueue.Len()).To(BeZero())
		})

		It("should leave a snapshot it cannot find alone", func() {
			controller.enqueueExternalContent(cacheKeyFunc(testNamespace, "gone"))
			Expect(controller.vmSnapshotContentQueue.Len()).To(BeZero())
		})
	})

	DescribeTable("should recognize", func(mode snapshotv1.SnapshotMode, supported bool) {
		Expect(supportedSnapshotMode(mode)).To(Equal(supported))
	},
		Entry("Direct", snapshotv1.SnapshotModeDirect, true),
		Entry("External", snapshotv1.SnapshotModeExternal, true),
		Entry("a mode from a newer version", snapshotv1.SnapshotMode("Hybrid"), false),
		Entry("a mode that differs only in case", snapshotv1.SnapshotMode("external"), false),
		Entry("the empty mode, which never reaches it", snapshotv1.SnapshotMode(""), false),
	)
})
