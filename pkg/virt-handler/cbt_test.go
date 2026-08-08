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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CBTHandler", func() {
	var (
		vmi             *v1.VirtualMachineInstance
		trackerInformer cache.SharedIndexInformer
		handler         *CBTHandler
	)

	const (
		testNamespace = "test-ns"
		testNodeName  = "test-node"
		testPodUID    = "test-pod-uid"
		stalePodUID   = "stale-pod-uid"
	)

	BeforeEach(func() {
		vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
		trackerInformer, _ = testutils.NewFakeInformerWithIndexersFor(&backupv1.VirtualMachineBackupTracker{}, controller.GetVirtualMachineBackupTrackerInformerIndexers())
		handler = NewCBTHandler(trackerInformer)
	})

	createTracker := func(name, vmName string, hasCheckpoint bool, syncedPodUID *types.UID) *backupv1.VirtualMachineBackupTracker {
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
				LastTrackedPodUID: syncedPodUID,
			}
		}
		return tracker
	}

	addTracker := func(tracker *backupv1.VirtualMachineBackupTracker) {
		ExpectWithOffset(1, trackerInformer.GetStore().Add(tracker)).To(Succeed())
	}

	setActivePod := func(vmi *v1.VirtualMachineInstance, podUID types.UID) {
		vmi.Status.NodeName = testNodeName
		vmi.Status.ActivePods = map[types.UID]string{podUID: testNodeName}
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

		It("should stay Initializing when a tracker needs redefinition", func() {
			setActivePod(vmi, testPodUID)
			addTracker(createTracker("tracker1", "test-vmi", true, nil))

			vmi.Spec.Volumes = []v1.Volume{pvcVolume("pvc1", "test-pvc")}
			domain := &api.Domain{Spec: api.DomainSpec{Devices: api.Devices{Disks: []api.Disk{diskWithDataStore("pvc1", true)}}}}

			err := handler.HandleChangedBlockTracking(vmi, domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingInitializing))
		})

		It("should enable when tracker has matching LastTrackedPodUID", func() {
			setActivePod(vmi, testPodUID)
			addTracker(createTracker("tracker1", "test-vmi", true, new(types.UID(testPodUID))))

			vmi.Spec.Volumes = []v1.Volume{pvcVolume("pvc1", "test-pvc")}
			domain := &api.Domain{Spec: api.DomainSpec{Devices: api.Devices{Disks: []api.Disk{diskWithDataStore("pvc1", true)}}}}

			err := handler.HandleChangedBlockTracking(vmi, domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingEnabled))
		})

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

	Context("anyTrackerNeedsRedefinition", func() {
		DescribeTable("with a single tracker and active pod",
			func(hasCheckpoint bool, trackedPodUID *types.UID, expected bool) {
				setActivePod(vmi, testPodUID)
				addTracker(createTracker("tracker1", "test-vmi", hasCheckpoint, trackedPodUID))
				needsRedefinition, err := handler.anyTrackerNeedsRedefinition(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(needsRedefinition).To(Equal(expected))
			},
			Entry("no checkpoint", false, nil, false),
			Entry("untracked checkpoint", true, nil, true),
			Entry("stale tracked pod", true, new(types.UID(stalePodUID)), true),
			Entry("current tracked pod", true, new(types.UID(testPodUID)), false),
		)

		It("does not need redefinition without trackers", func() {
			setActivePod(vmi, testPodUID)
			needsRedefinition, err := handler.anyTrackerNeedsRedefinition(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(needsRedefinition).To(BeFalse())
		})

		It("does not need redefinition when VMI has no active pods", func() {
			addTracker(createTracker("tracker1", "test-vmi", true, nil))
			needsRedefinition, err := handler.anyTrackerNeedsRedefinition(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(needsRedefinition).To(BeFalse())
		})

		It("detects redefinition after migration with stale source pod", func() {
			vmi.Status.NodeName = "target-node"
			vmi.Status.ActivePods = map[types.UID]string{
				types.UID("target-pod-uid"): "target-node",
			}
			addTracker(createTracker("tracker1", "test-vmi", true, new(types.UID("source-pod-uid"))))
			needsRedefinition, err := handler.anyTrackerNeedsRedefinition(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(needsRedefinition).To(BeTrue())
		})

		It("needs redefinition when at least one tracker is stale", func() {
			setActivePod(vmi, testPodUID)
			addTracker(createTracker("tracker1", "test-vmi", true, new(types.UID(testPodUID))))
			addTracker(createTracker("tracker2", "test-vmi", true, new(types.UID(stalePodUID))))
			needsRedefinition, err := handler.anyTrackerNeedsRedefinition(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(needsRedefinition).To(BeTrue())
		})
	})

	Context("backupTrackersForVMI", func() {
		It("should return empty list when no trackers exist", func() {
			trackers, err := handler.backupTrackersForVMI(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(trackers).To(BeEmpty())
		})

		It("should return trackers matching VMI namespace/name", func() {
			tracker := createTracker("tracker1", "test-vmi", false, nil)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			trackers, err := handler.backupTrackersForVMI(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(trackers).To(HaveLen(1))
			Expect(trackers[0].Name).To(Equal("tracker1"))
		})

		It("should not return trackers for different VMI", func() {
			tracker := createTracker("tracker1", "different-vmi", false, nil)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			trackers, err := handler.backupTrackersForVMI(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(trackers).To(BeEmpty())
		})

		It("should return nil when informer is nil", func() {
			handler := NewCBTHandler(nil)
			trackers, err := handler.backupTrackersForVMI(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(trackers).To(BeNil())
		})
	})
})
