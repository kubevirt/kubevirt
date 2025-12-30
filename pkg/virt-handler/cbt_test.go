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

package virthandler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CBTHandler", func() {
	var (
		mockCtrl        *gomock.Controller
		vmi             *v1.VirtualMachineInstance
		virtClient      *kubecli.MockKubevirtClient
		kubevirtCli     *kubevirtfake.Clientset
		trackerInformer cache.SharedIndexInformer
		handler         *CBTHandler
	)

	const testNamespace = "test-ns"

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
		kubevirtCli = kubevirtfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(mockCtrl)
		trackerInformer, _ = testutils.NewFakeInformerWithIndexersFor(&backupv1.VirtualMachineBackupTracker{}, controller.GetVirtualMachineBackupTrackerInformerIndexers())
		handler = NewCBTHandler(virtClient, trackerInformer)
	})

	createTracker := func(name, vmName string, hasCheckpoint bool, alreadyMarked bool) *backupv1.VirtualMachineBackupTracker {
		tracker := &backupv1.VirtualMachineBackupTracker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: backupv1.VirtualMachineBackupTrackerSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: pointer.P("kubevirt.io"),
					Kind:     "VirtualMachine",
					Name:     vmName,
				},
			},
		}
		if hasCheckpoint {
			tracker.Status = &backupv1.VirtualMachineBackupTrackerStatus{
				LatestCheckpoint: &backupv1.BackupCheckpoint{
					Name: "checkpoint-1",
				},
				CheckpointRedefinitionRequired: pointer.P(alreadyMarked),
			}
		}
		return tracker
	}

	addTracker := func(tracker *backupv1.VirtualMachineBackupTracker) {
		_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
			context.Background(), tracker, metav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, trackerInformer.GetStore().Add(tracker)).To(Succeed())
	}

	pvcVolume := func(name, claimName string) v1.Volume {
		return v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
				},
			},
		}
	}

	diskWithDataStore := func(name string, hasDataStore bool) api.Disk {
		disk := api.Disk{Alias: api.NewUserDefinedAlias(name), Source: api.DiskSource{}}
		if hasDataStore {
			disk.Source.DataStore = &api.DataStore{Type: "file"}
		}
		return disk
	}

	Context("HandleChangedBlockTracking", func() {
		BeforeEach(func() {
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing)
		})

		DescribeTable("should handle CBT enablement when initializing based on volume/disk configuration",
			func(volumes []v1.Volume, disks []api.Disk, expectedState v1.ChangedBlockTrackingState) {
				vmi.Spec.Volumes = volumes
				domain := &api.Domain{Spec: api.DomainSpec{Devices: api.Devices{Disks: disks}}}
				err := handler.HandleChangedBlockTracking(vmi, domain)
				Expect(err).ToNot(HaveOccurred())
				Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(expectedState))
			},
			Entry("all eligible volumes have DataStore",
				[]v1.Volume{pvcVolume("pvc1", "test-pvc"), pvcVolume("pvc2", "test-pvc2")},
				[]api.Disk{diskWithDataStore("pvc1", true), diskWithDataStore("pvc2", true)},
				v1.ChangedBlockTrackingEnabled),
			Entry("one eligible volume lacks DataStore",
				[]v1.Volume{pvcVolume("pvc1", "test-pvc"), pvcVolume("pvc2", "test-pvc2")},
				[]api.Disk{diskWithDataStore("pvc1", true), diskWithDataStore("pvc2", false)},
				v1.ChangedBlockTrackingInitializing),
			Entry("mixed volumes, only eligible ones checked",
				[]v1.Volume{
					{Name: "container-disk", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test:latest"}}},
					pvcVolume("pvc1", "test-pvc"),
				},
				[]api.Disk{diskWithDataStore("container-disk", false), diskWithDataStore("pvc1", true)},
				v1.ChangedBlockTrackingEnabled),
			Entry("only non-eligible volumes",
				[]v1.Volume{{Name: "container-disk", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test:latest"}}}},
				[]api.Disk{diskWithDataStore("container-disk", false)},
				v1.ChangedBlockTrackingEnabled),
			Entry("eligible volume with no matching disk",
				[]v1.Volume{pvcVolume("pvc1", "test-pvc")},
				[]api.Disk{diskWithDataStore("different-disk", true)},
				v1.ChangedBlockTrackingInitializing),
		)

		DescribeTable("should not update VMI CBT state when",
			func(cbtState *v1.ChangedBlockTrackingState, domainIsNil bool) {
				if cbtState != nil {
					cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, *cbtState)
				} else {
					vmi.Status.ChangedBlockTracking = nil
				}
				vmi.Spec.Volumes = []v1.Volume{pvcVolume("pvc1", "test-pvc")}

				var domain *api.Domain
				if !domainIsNil {
					domain = &api.Domain{Spec: api.DomainSpec{Devices: api.Devices{Disks: []api.Disk{diskWithDataStore("pvc1", true)}}}}
				}

				err := handler.HandleChangedBlockTracking(vmi, domain)
				Expect(err).ToNot(HaveOccurred())

				if cbtState == nil {
					Expect(vmi.Status.ChangedBlockTracking).To(BeNil())
				} else {
					Expect(vmi.Status.ChangedBlockTracking.State).To(Equal(*cbtState))
				}
			},
			Entry("domain is nil", pointer.P(v1.ChangedBlockTrackingInitializing), true),
			Entry("status is nil", nil, false),
			Entry("status is Undefined", pointer.P(v1.ChangedBlockTrackingUndefined), false),
			Entry("status is Enabled", pointer.P(v1.ChangedBlockTrackingEnabled), false),
			Entry("status is Disabled", pointer.P(v1.ChangedBlockTrackingDisabled), false),
			Entry("status is PendingRestart", pointer.P(v1.ChangedBlockTrackingPendingRestart), false),
			Entry("status is FGDisabled", pointer.P(v1.ChangedBlockTrackingFGDisabled), false),
		)
	})

	Context("markTrackersForRedefinition", func() {
		It("should return error when informer is nil", func() {
			handler := NewCBTHandler(virtClient, nil)
			vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{State: v1.ChangedBlockTrackingInitializing}
			err := handler.markTrackersForRedefinition(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tracker informer or clientset is nil"))
		})

		It("should return error when clientset is nil", func() {
			handler := NewCBTHandler(nil, trackerInformer)
			vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{State: v1.ChangedBlockTrackingInitializing}
			err := handler.markTrackersForRedefinition(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tracker informer or clientset is nil"))
		})

		DescribeTable("should skip trackers that don't need redefinition",
			func(modifyTracker func(*backupv1.VirtualMachineBackupTracker)) {
				if modifyTracker != nil {
					tracker := createTracker("tracker1", "test-vmi", false, false)
					modifyTracker(tracker)
					addTracker(tracker)
				}
				// expect no patch calls to the tracker
				virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
					Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace)).Times(0)

				err := handler.markTrackersForRedefinition(vmi)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry("when no trackers exist", nil),
			Entry("without status", func(t *backupv1.VirtualMachineBackupTracker) {
				t.Status = nil
			}),
			Entry("without latest checkpoint", func(t *backupv1.VirtualMachineBackupTracker) {
				t.Status = &backupv1.VirtualMachineBackupTrackerStatus{
					LatestCheckpoint: nil,
				}
			}),
			Entry("already marked for redefinition", func(t *backupv1.VirtualMachineBackupTracker) {
				t.Status = &backupv1.VirtualMachineBackupTrackerStatus{
					LatestCheckpoint: &backupv1.BackupCheckpoint{
						Name: "checkpoint-1",
					},
					CheckpointRedefinitionRequired: pointer.P(true),
				}
			}),
		)

		It("should mark multiple trackers for redefinition", func() {
			tracker1 := createTracker("tracker1", "test-vmi", true, false)
			tracker2 := createTracker("tracker2", "test-vmi", true, false)
			addTracker(tracker1)
			addTracker(tracker2)

			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace)).Times(2)

			err := handler.markTrackersForRedefinition(vmi)
			Expect(err).ToNot(HaveOccurred())

			// Verify both trackers were marked
			updated1, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated1.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated1.Status.CheckpointRedefinitionRequired).To(BeTrue())

			updated2, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker2", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated2.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated2.Status.CheckpointRedefinitionRequired).To(BeTrue())
		})

		It("should only mark trackers that need redefinition", func() {
			// tracker1 has checkpoint and needs marking
			tracker1 := createTracker("tracker1", "test-vmi", true, false)
			// tracker2 is already marked, should be skipped
			tracker2 := createTracker("tracker2", "test-vmi", true, true)
			// tracker3 has no checkpoint, should be skipped
			tracker3 := createTracker("tracker3", "test-vmi", false, false)
			addTracker(tracker1)
			addTracker(tracker2)
			addTracker(tracker3)

			// Only one call expected since tracker2 and tracker3 should be skipped
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace)).Times(1)

			err := handler.markTrackersForRedefinition(vmi)
			Expect(err).ToNot(HaveOccurred())

			// Verify tracker1 was marked
			updated1, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated1.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated1.Status.CheckpointRedefinitionRequired).To(BeTrue())

			// Verify tracker2 was not updated (still has its original state)
			updated2, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker2", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated2.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated2.Status.CheckpointRedefinitionRequired).To(BeTrue())

			// Verify tracker3 has no status (was skipped)
			updated3, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker3", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated3.Status).To(BeNil())
		})
	})

	Context("backupTrackersForVMI", func() {
		It("should return empty list when no trackers exist", func() {
			trackers := handler.backupTrackersForVMI(vmi)
			Expect(trackers).To(BeEmpty())
		})

		It("should return trackers matching VMI namespace/name", func() {
			tracker := createTracker("tracker1", "test-vmi", false, false)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			trackers := handler.backupTrackersForVMI(vmi)
			Expect(trackers).To(HaveLen(1))
			Expect(trackers[0].Name).To(Equal("tracker1"))
		})

		It("should not return trackers for different VMI", func() {
			tracker := createTracker("tracker1", "different-vmi", false, false)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			trackers := handler.backupTrackersForVMI(vmi)
			Expect(trackers).To(BeEmpty())
		})

		It("should return nil when informer is nil", func() {
			handler := NewCBTHandler(virtClient, nil)
			trackers := handler.backupTrackersForVMI(vmi)
			Expect(trackers).To(BeNil())
		})
	})
})
