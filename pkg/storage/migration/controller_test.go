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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package migration

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	virtstoragev1alpha1 "kubevirt.io/api/storage/v1alpha1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	k8sfake "k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Volume Migration", func() {
	var (
		stop                    chan struct{}
		ctrl                    *gomock.Controller
		controller              *VolumeMigrationController
		virtClient              *kubecli.MockKubevirtClient
		volumeMigrationInformer cache.SharedIndexInformer
		migrationInformer       cache.SharedIndexInformer
		vmiInformer             cache.SharedIndexInformer
		vmInformer              cache.SharedIndexInformer
		pvcInformer             cache.SharedIndexInformer
		cdiInformer             cache.SharedIndexInformer
		cdiConfigInformer       cache.SharedIndexInformer
		mockQueue               *testutils.MockWorkQueue
		volMigrationSource      *framework.FakeControllerSource
		volMigClient            *kubevirtfake.Clientset
		k8sClient               *k8sfake.Clientset
		migrationInterface      *kubecli.MockVirtualMachineInstanceMigrationInterface
		vmiInterface            *kubecli.MockVirtualMachineInstanceInterface
		vmInterface             *kubecli.MockVirtualMachineInterface
	)
	const (
		testNs = "testns"
	)
	syncCaches := func(stop chan struct{}) {
		go volumeMigrationInformer.Run(stop)
		go migrationInformer.Run(stop)
		go vmiInformer.Run(stop)
		go vmInformer.Run(stop)
		go pvcInformer.Run(stop)
		go cdiInformer.Run(stop)
		go cdiConfigInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop,
			volumeMigrationInformer.HasSynced,
			vmiInformer.HasSynced,
			vmInformer.HasSynced,
			pvcInformer.HasSynced,
			cdiInformer.HasSynced,
			cdiConfigInformer.HasSynced,
			migrationInformer.HasSynced)).To(BeTrue())

	}
	addVolMigration := func(migration *virtstoragev1alpha1.VolumeMigration) {
		mockQueue.ExpectAdds(1)
		volMigrationSource.Add(migration)
		mockQueue.Wait()
	}
	deleteVolMigration := func(migration *virtstoragev1alpha1.VolumeMigration) {
		volMigrationSource.Delete(migration)
		mockQueue.Wait()
	}
	addVMI := func(vmi *virtv1.VirtualMachineInstance) {
		err := vmiInformer.GetStore().Add(vmi)
		Expect(err).ShouldNot(HaveOccurred())
	}
	addVM := func(vm *virtv1.VirtualMachine) {
		err := vmInformer.GetStore().Add(vm)
		Expect(err).ShouldNot(HaveOccurred())
	}
	addPVC := func(pvc *k8sv1.PersistentVolumeClaim) {
		err := pvcInformer.GetStore().Add(pvc)
		Expect(err).ShouldNot(HaveOccurred())
	}
	addMigration := func(mig *virtv1.VirtualMachineInstanceMigration) {
		err := migrationInformer.GetStore().Add(mig)
		Expect(err).ShouldNot(HaveOccurred())
	}

	expectVolumeMigrationUpdate := func(err error) {
		volMigClient.Fake.PrependReactor("update", "volumemigrations", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())

			return true, update.GetObject(), err
		})
	}
	expectPVCDeletion := func() {
		k8sClient.Fake.PrependReactor("delete", "persistentvolumeclaims",
			func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())
				return true, nil, nil
			})
	}

	setVMOwner := func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
		vmi.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				Kind:       virtv1.VirtualMachineGroupVersionKind.Kind,
				APIVersion: virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
				Name:       vm.Name,
				UID:        vm.UID,
				Controller: pointer.BoolPtr(true),
			},
		}
	}

	BeforeEach(func() {
		var err error
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		volumeMigrationInformer, volMigrationSource = testutils.NewFakeInformerFor(&virtstoragev1alpha1.VolumeMigration{})
		vmiInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		pvcInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		migrationInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstanceMigration{})
		cdiInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstanceMigration{})
		cdiConfigInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstanceMigration{})
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

		controller, err = NewVolumeMigrationController(virtClient,
			volumeMigrationInformer, migrationInformer,
			vmiInformer, vmInformer,
			pvcInformer,
			cdiInformer, cdiConfigInformer)
		Expect(err).ShouldNot(HaveOccurred())

		syncCaches(stop)

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		k8sClient = k8sfake.NewSimpleClientset()
		volMigClient = kubevirtfake.NewSimpleClientset()

		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstanceMigration(testNs).Return(migrationInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(testNs).Return(vmiInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachine(testNs).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VolumeMigration(testNs).
			Return(volMigClient.StorageV1alpha1().VolumeMigrations(testNs)).AnyTimes()
	})

	AfterEach(func() {
		close(stop)
	})
	Context("Update VolumeMigration status", func() {
		const (
			vmiMig  = "vmimigtest"
			vmiName = "test"
		)
		var (
			startTime metav1.Time
			endTime   metav1.Time
		)
		addDefaultReactors := func() {
			volMigClient.Fake.PrependReactor("update", "volumemigrations",
				func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					update, ok := action.(testing.UpdateAction)
					Expect(ok).To(BeTrue())
					sm, ok := update.GetObject().(*virtstoragev1alpha1.VolumeMigration)
					Expect(ok).To(BeTrue())

					return true, sm, nil
				})

		}

		createVMIMigWithStatus := func(status *virtv1.VirtualMachineInstanceMigrationStatus) *virtv1.VirtualMachineInstanceMigration {
			return &virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{Name: vmiMig, Namespace: testNs},
				Spec:       virtv1.VirtualMachineInstanceMigrationSpec{VMIName: vmiName},
				Status:     *(status.DeepCopy()),
			}
		}
		BeforeEach(func() {
			startTime = metav1.NewTime(metav1.Now().Add(time.Duration(-10) * time.Second))
			endTime = metav1.Time{Time: time.Now()}
			addDefaultReactors()
		})

		DescribeTable("updateStatusVolumeMigration", func(mig *virtv1.VirtualMachineInstanceMigration, status *virtstoragev1alpha1.VolumeMigrationStatus) {
			volMig := &virtstoragev1alpha1.VolumeMigration{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: testNs},
			}
			res, err := controller.updateStatusVolumeMigration(volMig, mig)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Status.StartTimestamp).Should(Equal(status.StartTimestamp))
			Expect(res.Status.EndTimestamp).Should(Equal(status.EndTimestamp))
			Expect(res.Status.Phase).Should(Equal(status.Phase))
			Expect(*res.Status.VirtualMachineInstanceName).Should(Equal(vmiName))
			Expect(*res.Status.VirtualMachineMigrationName).Should(Equal(vmiMig))
		},
			Entry("with pending VMI migration", createVMIMigWithStatus(&virtv1.VirtualMachineInstanceMigrationStatus{
				Phase:          virtv1.MigrationScheduling,
				MigrationState: &virtv1.VirtualMachineInstanceMigrationState{},
			}), &virtstoragev1alpha1.VolumeMigrationStatus{
				Phase: virtstoragev1alpha1.VolumeMigrationPhaseScheduling,
			}),
			Entry("with running VMI migration", createVMIMigWithStatus(&virtv1.VirtualMachineInstanceMigrationStatus{
				Phase: virtv1.MigrationRunning,
				MigrationState: &virtv1.VirtualMachineInstanceMigrationState{
					StartTimestamp: startTime.DeepCopy(),
				},
			}), &virtstoragev1alpha1.VolumeMigrationStatus{
				StartTimestamp: startTime.DeepCopy(),
				Phase:          virtstoragev1alpha1.VolumeMigrationPhaseRunning,
			}),
			Entry("with succeeded VMI migration", createVMIMigWithStatus(&virtv1.VirtualMachineInstanceMigrationStatus{
				Phase: virtv1.MigrationSucceeded,
				MigrationState: &virtv1.VirtualMachineInstanceMigrationState{
					StartTimestamp: startTime.DeepCopy(),
					EndTimestamp:   endTime.DeepCopy(),
				},
			}), &virtstoragev1alpha1.VolumeMigrationStatus{
				StartTimestamp: startTime.DeepCopy(),
				EndTimestamp:   endTime.DeepCopy(),
				Phase:          virtstoragev1alpha1.VolumeMigrationPhaseSucceeded,
			}),
			Entry("with failed VMI migration", createVMIMigWithStatus(&virtv1.VirtualMachineInstanceMigrationStatus{
				Phase: virtv1.MigrationFailed,
				MigrationState: &virtv1.VirtualMachineInstanceMigrationState{
					StartTimestamp: startTime.DeepCopy(),
					EndTimestamp:   endTime.DeepCopy(),
				},
			}), &virtstoragev1alpha1.VolumeMigrationStatus{
				StartTimestamp: startTime.DeepCopy(),
				EndTimestamp:   endTime.DeepCopy(),
				Phase:          virtstoragev1alpha1.VolumeMigrationPhaseFailed,
			}),
		)
	})

	Context("VolumeMigration lifecycle", func() {
		const (
			srcVol        = "src"
			dstVol        = "dst"
			volMigName    = "vol-mig"
			vmiName       = "testVmi"
			defaultPolicy = virtstoragev1alpha1.SourceReclaimPolicyDelete
		)
		createPVC := func(name string) *k8sv1.PersistentVolumeClaim {
			return &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: testNs,
				},
			}
		}
		createVolMig := func(name string,
			policy virtstoragev1alpha1.SourceReclaimPolicy) *virtstoragev1alpha1.VolumeMigration {
			return &virtstoragev1alpha1.VolumeMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      volMigName,
					Namespace: testNs,
				},
				Spec: virtstoragev1alpha1.VolumeMigrationSpec{
					MigratedVolume: []virtstoragev1alpha1.MigratedVolume{
						{
							SourceClaim:         srcVol,
							DestinationClaim:    dstVol,
							SourceReclaimPolicy: policy,
						},
					},
				},
			}

		}
		createMig := func(name string) *virtv1.VirtualMachineInstanceMigration {
			return &virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: testNs,
					Labels:    map[string]string{virtstoragev1alpha1.VolumeMigrationLabel: volMigName},
				},
			}
		}

		initVolMigCreation := func(vmOwner bool, policy virtstoragev1alpha1.SourceReclaimPolicy) *virtstoragev1alpha1.VolumeMigration {
			volMig := createVolMig(volMigName, policy)
			vmi := createVMIWithPVCs(vmiName, testNs, srcVol)
			if vmOwner {
				vm := createVMFromVMI(vmi)
				addVM(vm)
				setVMOwner(vm, vmi)
			}

			addVolMigration(volMig)
			expectVolumeMigrationUpdate(nil)
			addVMI(vmi)
			addPVC(createPVC(srcVol))
			addPVC(createPVC(dstVol))
			vmiInterface.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(
				&v1.VirtualMachineInstanceList{Items: []v1.VirtualMachineInstance{*vmi}}, nil)

			return volMig
		}
		initMigration := func(state *virtv1.VirtualMachineInstanceMigrationState, vmOwner bool,
			policy virtstoragev1alpha1.SourceReclaimPolicy) {
			volMig := initVolMigCreation(vmOwner, policy)
			mig := createMig(volMig.GetVirtualMachiheInstanceMigrationName(vmiName))
			if state != nil {
				mig.Status.MigrationState = state
			}
			addMigration(mig)
		}
		createMigState := func(completed bool, failed bool) *virtv1.VirtualMachineInstanceMigrationState {
			return &virtv1.VirtualMachineInstanceMigrationState{
				Completed: completed,
				Failed:    failed,
			}
		}

		It("Deletion", func() {
			volMig := createVolMig(volMigName, defaultPolicy)
			volMig.ObjectMeta.DeletionTimestamp = &metav1.Time{
				Time: time.Now(),
			}
			migName := volMig.GetVirtualMachiheInstanceMigrationName("test")

			migrationInterface.EXPECT().List(gomock.Any()).Times(1).Return(&v1.VirtualMachineInstanceMigrationList{
				Items: []v1.VirtualMachineInstanceMigration{{ObjectMeta: metav1.ObjectMeta{Name: migName}}},
			}, nil)
			migrationInterface.EXPECT().Delete(migName, gomock.Any()).Times(1).Return(nil)
			addVolMigration(volMig)
			deleteVolMigration(volMig)

			controller.Execute()
		})

		It("Creation", func() {
			initVolMigCreation(false, defaultPolicy)
			vmiInterface.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
			migrationInterface.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

			controller.Execute()
		})

		It("Update status after volume migration creation", func() {
			initMigration(nil, false, defaultPolicy)
			controller.Execute()
		})

		DescribeTable("Source PVC reclaim policy", func(policy string) {
			p := virtstoragev1alpha1.SourceReclaimPolicy(policy)
			initMigration(createMigState(true, false), false, p)
			migrationInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			if p == virtstoragev1alpha1.SourceReclaimPolicyDelete {
				expectPVCDeletion()
			}

			controller.Execute()
		},
			Entry("with delete policy", string(virtstoragev1alpha1.SourceReclaimPolicyDelete)),
			Entry("with retain policy", string(virtstoragev1alpha1.SourceReclaimPolicyRetain)),
			Entry("with wrong policy", "wrong"),
		)

		It("failed to update the finalizer", func() {
			volMig := createVolMig(volMigName, defaultPolicy)
			vmi := createVMIWithPVCs(vmiName, testNs, srcVol)
			addVolMigration(volMig)
			vmiInterface.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(
				&v1.VirtualMachineInstanceList{Items: []v1.VirtualMachineInstance{*vmi}}, nil)
			key, _ := controller.Queue.Get()

			expectVolumeMigrationUpdate(fmt.Errorf("failed to update"))

			Expect(controller.execute(key.(string))).Should(HaveOccurred())
		})
	})

	Context("classifyVolumesPerVMI", func() {

		const (
			srcVolName = "src"
			dstVolName = "dst"
			vmiName    = "test"
		)

		migVols := []virtstoragev1alpha1.MigratedVolume{
			{SourceClaim: srcVolName, DestinationClaim: dstVolName}}
		migVol2 := virtstoragev1alpha1.MigratedVolume{
			SourceClaim: "src2", DestinationClaim: "dst2",
		}

		volMig := &virtstoragev1alpha1.VolumeMigration{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: testNs},
			Spec: virtstoragev1alpha1.VolumeMigrationSpec{
				MigratedVolume: migVols,
			},
		}
		volMig2 := &virtstoragev1alpha1.VolumeMigration{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: testNs},
			Spec: virtstoragev1alpha1.VolumeMigrationSpec{
				MigratedVolume: append(migVols, migVol2),
			},
		}
		vmi := v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: vmiName,
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{{
					Name: "vol",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: srcVolName,
							},
						},
					},
				}},
			},
		}

		migVolVmi := map[string][]virtstoragev1alpha1.MigratedVolume{
			vmiName: migVols,
		}

		DescribeTable("classification of migrated volumes", func(volMig *virtstoragev1alpha1.VolumeMigration, vmis []v1.VirtualMachineInstance,
			expectedmigrVolPerVMI map[string][]virtstoragev1alpha1.MigratedVolume, expectedPendingVols []virtstoragev1alpha1.MigratedVolume) {
			vmiInterface.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(&v1.VirtualMachineInstanceList{Items: vmis}, nil)

			vmigrVolPerVMI, pendingVols, err := controller.classifyVolumesPerVMI(volMig)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(vmigrVolPerVMI, expectedmigrVolPerVMI)).To(BeTrue(),
				"Expected migrated volumes :%v to be equal to %v", vmigrVolPerVMI, expectedmigrVolPerVMI)
			Expect(equality.Semantic.DeepEqual(pendingVols, expectedPendingVols)).To(BeTrue(),
				"Expected pending volumes :%v to be equal to %v", pendingVols, expectedPendingVols)

		},
			Entry("migrated volumes", volMig, []v1.VirtualMachineInstance{vmi}, migVolVmi,
				[]virtstoragev1alpha1.MigratedVolume{}),
			Entry("pending volumes", volMig, []v1.VirtualMachineInstance{}, make(map[string][]virtstoragev1alpha1.MigratedVolume),
				migVols),
			Entry("migrated and pending volumes", volMig2, []v1.VirtualMachineInstance{vmi}, migVolVmi,
				[]virtstoragev1alpha1.MigratedVolume{migVol2}),
		)
	})

	Context("validateMigrateVolumes", func() {
		// This test suite assumes that the first volume is invalid when
		// there are rejected volumes
		const (
			srcVolName1 = "src1"
			dstVolName1 = "dst1"
			srcVolName2 = "src2"
			dstVolName2 = "dst2"
			vmiName     = "test"
		)

		migVols := []virtstoragev1alpha1.MigratedVolume{
			{SourceClaim: srcVolName1, DestinationClaim: dstVolName1},
			{SourceClaim: srcVolName2, DestinationClaim: dstVolName2},
		}

		createVols := func(hotplug bool) []v1.Volume {
			return []v1.Volume{
				{
					Name: "vol1",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: srcVolName1,
							},
							Hotpluggable: hotplug,
						},
					},
				},
				{
					Name: "vol2",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: srcVolName2,
						},
					},
				},
			}

		}
		createDisks := func(shareable bool) []v1.Disk {
			disks := []v1.Disk{
				{Name: "vol1"},
				{Name: "vol2"},
			}
			if shareable {
				disks[0].Shareable = pointer.BoolPtr(true)
			}

			return disks
		}
		createLUNDisk := func(_disk bool) []v1.Disk {
			disks := []v1.Disk{
				{
					Name: "vol1",
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{},
					},
				},
				{Name: "vol2"},
			}

			return disks
		}
		createFilesystems := func() []v1.Filesystem {
			return []v1.Filesystem{
				{Name: "vol1"},
			}
		}

		createEmptyFilesystems := func() []v1.Filesystem {
			return []v1.Filesystem{}
		}

		createVMIWithVolDiskFs := func(createDisks func(shareable bool) []v1.Disk,
			createFilesystems func() []v1.Filesystem,
			createVols func(hotplug bool) []v1.Volume,
			hotplug, shareable bool) *v1.VirtualMachineInstance {
			return &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: vmiName,
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Filesystems: createFilesystems(),
							Disks:       createDisks(shareable),
						},
					},
					Volumes: createVols(hotplug),
				},
			}
		}

		DescribeTable("evaluate rejected volumes for the volume migration",
			func(vmi *virtv1.VirtualMachineInstance, reason string) {
				volMig := &virtstoragev1alpha1.VolumeMigration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: testNs,
					},
					Spec: virtstoragev1alpha1.VolumeMigrationSpec{
						MigratedVolume: migVols,
					},
				}
				state := virtstoragev1alpha1.MigratedVolumeValidationRejected
				r := &reason
				if reason == "" {
					state = virtstoragev1alpha1.MigratedVolumeValidationValid
					r = nil
				}

				rejMig := validateRejectedVolumes(volMig, vmi)

				Expect(rejMig.Status.VolumeMigrationStates).To(HaveLen(2))
				Expect(rejMig.Status.VolumeMigrationStates[1].Reason).To(BeNil())
				Expect(rejMig.Status.VolumeMigrationStates[1].Validation).To(
					Equal(virtstoragev1alpha1.MigratedVolumeValidationValid))
				Expect(rejMig.Status.VolumeMigrationStates[0].Reason).To(Equal(r))
				Expect(rejMig.Status.VolumeMigrationStates[0].Validation).To(
					Equal(state))
			},
			Entry("no rejected volumes", createVMIWithVolDiskFs(createDisks, createEmptyFilesystems, createVols,
				false, false), ""),
			Entry("reject shareable volumes", createVMIWithVolDiskFs(createDisks, createEmptyFilesystems, createVols,
				false, true), virtstoragev1alpha1.ReasonRejectShareableVolumes),
			Entry("reject hotplug volumes", createVMIWithVolDiskFs(createDisks, createEmptyFilesystems, createVols,
				true, false), virtstoragev1alpha1.ReasonRejectHotplugVolumes),
			Entry("reject filesystem volumes", createVMIWithVolDiskFs(createDisks, createFilesystems, createVols,
				false, false), virtstoragev1alpha1.ReasonRejectFilesystemVolumes),
			Entry("reject lun volumes", createVMIWithVolDiskFs(createLUNDisk, createEmptyFilesystems, createVols,
				false, false), virtstoragev1alpha1.ReasonRejectLUNVolumes),
		)

	})
})
