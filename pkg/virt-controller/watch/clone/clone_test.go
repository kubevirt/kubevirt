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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package clone

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	clone "kubevirt.io/api/clone/v1beta1"
	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	kvcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	restoreResource         = "virtualmachinerestores"
	vmAPIGroup              = "kubevirt.io"
	snapshotAPIGroup        = "snapshot.kubevirt.io"
	testCloneUID            = "clone-uid"
	testSnapshotName        = "tmp-snapshot-clone-uid"
	testSnapshotContentName = "vmsnapshot-content-snapshot-UID"
	testRestoreName         = "tmp-restore-clone-uid"
)

var _ = Describe("Clone", func() {
	var (
		controller *VMCloneController
		recorder   *record.FakeRecorder
		mockQueue  *testutils.MockWorkQueue[string]

		client    *kubevirtfake.Clientset
		k8sClient *k8sfake.Clientset
		sourceVM  *virtv1.VirtualMachine
		vmClone   *clone.VirtualMachineClone
	)

	addVM := func(vm *virtv1.VirtualMachine) {
		err := controller.vmStore.Add(vm)
		Expect(err).ShouldNot(HaveOccurred())
	}

	addClone := func(vmClone *clone.VirtualMachineClone) {
		var err error
		vmClone, err = client.CloneV1beta1().VirtualMachineClones(metav1.NamespaceDefault).Create(context.TODO(), vmClone, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		controller.vmCloneIndexer.Add(vmClone)
		key, err := kvcontroller.KeyFunc(vmClone)
		Expect(err).To(Not(HaveOccurred()))
		mockQueue.Add(key)
	}

	addSnapshot := func(snapshot *snapshotv1.VirtualMachineSnapshot) {
		var err error
		snapshot, err = client.SnapshotV1beta1().VirtualMachineSnapshots(metav1.NamespaceDefault).Create(context.TODO(), snapshot, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		err = controller.snapshotStore.Add(snapshot)
		Expect(err).ToNot(HaveOccurred())
	}

	addSnapshotContent := func(snapshotContent *snapshotv1.VirtualMachineSnapshotContent) {
		err := controller.snapshotContentStore.Add(snapshotContent)
		Expect(err).ShouldNot(HaveOccurred())
	}

	addRestore := func(restore *snapshotv1.VirtualMachineRestore) {
		var err error
		restore, err = client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault).Create(context.TODO(), restore, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		err = controller.restoreStore.Add(restore)
		Expect(err).ToNot(HaveOccurred())
	}

	addPVC := func(pvc *k8sv1.PersistentVolumeClaim) {
		err := controller.pvcStore.Add(pvc)
		Expect(err).ShouldNot(HaveOccurred())
	}

	expectSnapshotExists := func() {
		vmSnapshot, err := client.SnapshotV1beta1().VirtualMachineSnapshots(metav1.NamespaceDefault).Get(context.TODO(), testSnapshotName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmSnapshot).ToNot(BeNil())
		Expect(vmSnapshot.Spec.Source.Kind).To(Equal("VirtualMachine"))
		Expect(vmSnapshot.OwnerReferences).To(HaveLen(1))
		validateOwnerReference(vmSnapshot.OwnerReferences[0], vmClone)
	}

	expectSnapshotDoesNotExist := func() {
		_, err := client.SnapshotV1beta1().VirtualMachineSnapshots(metav1.NamespaceDefault).Get(context.TODO(), testSnapshotName, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Snapshot should not exists")
	}

	expectRestoreExists := func() {
		vmRestore, err := client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmRestore).ToNot(BeNil())
		Expect(vmRestore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
		Expect(vmRestore.OwnerReferences).To(HaveLen(1))
		validateOwnerReference(vmRestore.OwnerReferences[0], vmClone)
	}

	expectRestoreCreationFailure := func(snapshotName string, vmClone *clone.VirtualMachineClone, restoreName string) {
		client.Fake.PrependReactor("create", restoreResource, func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			restorecreated := create.GetObject().(*snapshotv1.VirtualMachineRestore)
			Expect(restorecreated.Spec.VirtualMachineSnapshotName).To(Equal(snapshotName))
			Expect(restorecreated.OwnerReferences).To(HaveLen(1))
			validateOwnerReference(restorecreated.OwnerReferences[0], vmClone)

			return true, nil, fmt.Errorf("when snapshot source and restore target VMs are different - target VM must not exist")
		})
	}

	expectRestoreDoesNotExist := func() {
		_, err := client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Restore should not exists")
	}

	expectCloneBeInPhase := func(phase clone.VirtualMachineClonePhase) {
		clone, err := client.CloneV1beta1().VirtualMachineClones(metav1.NamespaceDefault).Get(context.TODO(), vmClone.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(clone).ToNot(BeNil())
		Expect(clone.Status.Phase).To(Equal(phase))
	}

	expectCloneDeletion := func() {
		_, err := client.CloneV1beta1().VirtualMachineClones(metav1.NamespaceDefault).Get(context.TODO(), vmClone.Name, metav1.GetOptions{})
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	}

	expectEvent := func(event Event) {
		testutils.ExpectEvent(recorder, string(event))
	}

	setSnapshotSource := func(vmClone *clone.VirtualMachineClone, snapshotName string) {
		source := vmClone.Spec.Source
		source.APIGroup = pointer.P(snapshotAPIGroup)
		source.Kind = "VirtualMachineSnapshot"
		source.Name = snapshotName
	}

	setupResources := func() {
		sourceVMI := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
		)
		sourceVMI.Namespace = metav1.NamespaceDefault
		sourceVM = libvmi.NewVirtualMachine(sourceVMI)

		vmClone = kubecli.NewMinimalCloneWithNS("testclone", metav1.NamespaceDefault)
		cloneSourceRef := &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(vmAPIGroup),
			Kind:     "VirtualMachine",
			Name:     sourceVM.Name,
		}
		cloneTargetRef := cloneSourceRef.DeepCopy()
		cloneTargetRef.Name = "test-target-vm"

		vmClone.Spec.Source = cloneSourceRef
		vmClone.Spec.Target = cloneTargetRef
		vmClone.UID = testCloneUID
		updateCloneConditions(vmClone,
			newProgressingCondition(k8sv1.ConditionTrue, "Still processing"),
			newReadyCondition(k8sv1.ConditionFalse, "Still processing"),
		)
	}

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		vmInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		snapshotInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
		restoreInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineRestore{})
		cloneInformer, _ := testutils.NewFakeInformerFor(&clone.VirtualMachineClone{})
		snapshotContentInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		controller, _ = NewVmCloneController(
			virtClient,
			cloneInformer,
			snapshotInformer,
			restoreInformer,
			vmInformer,
			snapshotContentInformer,
			pvcInformer,
			recorder)
		mockQueue = testutils.NewMockWorkQueue(controller.vmCloneQueue)
		controller.vmCloneQueue = mockQueue

		setupResources()

		client = kubevirtfake.NewSimpleClientset()

		virtClient.EXPECT().VirtualMachineClone(metav1.NamespaceDefault).Return(client.CloneV1beta1().VirtualMachineClones(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineSnapshot(metav1.NamespaceDefault).Return(client.SnapshotV1beta1().VirtualMachineSnapshots(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineRestore(metav1.NamespaceDefault).Return(client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineSnapshotContent(metav1.NamespaceDefault).Return(client.SnapshotV1beta1().VirtualMachineSnapshotContents(metav1.NamespaceDefault)).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
	})

	sanityExecute := func() {
		controllertesting.SanityExecute(controller, []cache.Store{
			controller.vmCloneIndexer, controller.snapshotContentStore, controller.restoreStore, controller.vmStore,
			controller.snapshotContentStore, controller.pvcStore,
		}, Default)
	}

	Context("basic controller operations", func() {
		Context("with source VM", func() {
			It("should report event if source VM doesn't exist, phase shouldn't change", func() {
				vmClone.Status.Phase = clone.PhaseUnset
				addClone(vmClone)

				controller.Execute()
				expectEvent(SourceDoesNotExist)
				expectCloneBeInPhase(clone.PhaseUnset)
			})

			It("clone should fail if source VM has backendstorage", func() {
				sourceVM.Spec.Template.Spec.Domain.Devices.TPM = &virtv1.TPMDevice{
					Persistent: pointer.P(true),
				}
				addVM(sourceVM)
				vmClone.Status.Phase = clone.PhaseUnset
				addClone(vmClone)

				controller.Execute()
				expectEvent(SourceWithBackendStorageInvalid)
				expectCloneBeInPhase(clone.Failed)
			})

			It("should report event if VM volumeSnapshots are invalid", func() {
				sourceVM.Spec.Template.Spec.Volumes = append(sourceVM.Spec.Template.Spec.Volumes, virtv1.Volume{
					Name: "disk0",
					VolumeSource: virtv1.VolumeSource{
						DataVolume: &virtv1.DataVolumeSource{
							Name: "testdv",
						},
					},
				})
				addVM(sourceVM)
				vmClone.Status.Phase = clone.PhaseUnset
				addClone(vmClone)

				controller.Execute()
				expectEvent(VMVolumeSnapshotsInvalid)
				expectCloneBeInPhase(clone.PhaseUnset)
			})

			It("should report event if VM volumeSnapshots are not snapshotable", func() {
				sourceVM.Spec.Template.Spec.Volumes = append(sourceVM.Spec.Template.Spec.Volumes, virtv1.Volume{
					Name: "disk0",
					VolumeSource: virtv1.VolumeSource{
						DataVolume: &virtv1.DataVolumeSource{
							Name: "testdv",
						},
					},
				})
				sourceVM.Status.VolumeSnapshotStatuses = []virtv1.VolumeSnapshotStatus{
					{
						Name:    "disk0",
						Enabled: false,
					},
				}
				addVM(sourceVM)
				vmClone.Status.Phase = clone.PhaseUnset
				addClone(vmClone)

				controller.Execute()
				expectEvent(VMVolumeSnapshotsInvalid)
				expectCloneBeInPhase(clone.PhaseUnset)
			})

			DescribeTable("should create snapshot if not exists yet", func(phase clone.VirtualMachineClonePhase) {
				vmClone.Status.Phase = phase

				addVM(sourceVM)
				addClone(vmClone)

				sanityExecute()
				expectEvent(SnapshotCreated)
				expectSnapshotExists()
				expectCloneBeInPhase(clone.SnapshotInProgress)
			},
				Entry("with phase unset", clone.PhaseUnset),
				Entry("with phase snapshot in progress", clone.SnapshotInProgress),
			)

			When("snapshot is created", func() {
				var snapshot *snapshotv1.VirtualMachineSnapshot

				BeforeEach(func() {
					snapshot = createVirtualMachineSnapshot(sourceVM)
				})

				It("and is not ready yet - should not do anything", func() {
					snapshot.Status.ReadyToUse = pointer.P(false)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.Phase = clone.SnapshotInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)

					sanityExecute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clone.SnapshotInProgress)
					expectRestoreDoesNotExist()
				})

				It("and is ready - should update status and create restore", func() {
					snapshotContent := createVirtualMachineSnapshotContent(sourceVM)

					snapshot.Status.ReadyToUse = pointer.P(true)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.Phase = clone.SnapshotInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addSnapshotContent(snapshotContent)

					sanityExecute()
					expectEvent(SnapshotReady)
					expectEvent(RestoreCreated)
					expectCloneBeInPhase(clone.RestoreInProgress)
					expectRestoreExists()
				})
			})

			When("restore is created", func() {
				var (
					snapshot *snapshotv1.VirtualMachineSnapshot
					restore  *snapshotv1.VirtualMachineRestore
				)

				BeforeEach(func() {
					snapshot = createVirtualMachineSnapshot(sourceVM)
					snapshot.Status.ReadyToUse = pointer.P(true)
					restore = createVirtualMachineRestore(sourceVM, snapshot.Name)
				})

				It("and is not ready yet - should do nothing", func() {
					restore.Status.Complete = pointer.P(false)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.RestoreName = pointer.P(restore.Name)
					vmClone.Status.Phase = clone.RestoreInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					sanityExecute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clone.RestoreInProgress)
				})

				It("and is ready - should update status", func() {
					restore.Status.Complete = pointer.P(true)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.RestoreName = pointer.P(restore.Name)
					vmClone.Status.Phase = clone.RestoreInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					sanityExecute()
					expectEvent(RestoreReady)
					expectCloneBeInPhase(clone.CreatingTargetVM)
				})
			})

			When("snapshot and restore are finished", func() {
				var (
					snapshot *snapshotv1.VirtualMachineSnapshot
					restore  *snapshotv1.VirtualMachineRestore
				)

				BeforeEach(func() {
					snapshot = createVirtualMachineSnapshot(sourceVM)
					snapshot.Status.ReadyToUse = pointer.P(true)
					restore = createVirtualMachineRestore(sourceVM, snapshot.Name)
					restore.Status.Complete = pointer.P(true)
					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.RestoreName = pointer.P(restore.Name)
					vmClone.Status.Phase = clone.CreatingTargetVM
				})

				It("and the target VM is not ready - should do nothing", func() {
					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					sanityExecute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clone.CreatingTargetVM)
				})

				It("and the target VM ready should move to Succeeded phase", func() {
					targetVM := sourceVM.DeepCopy()
					targetVM.Name = vmClone.Spec.Target.Name

					addVM(sourceVM)
					addVM(targetVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					sanityExecute()
					expectEvent(TargetVMCreated)
					expectCloneBeInPhase(clone.Succeeded)
					expectSnapshotDoesNotExist()
					expectRestoreDoesNotExist()
				})
			})

			When("the clone process is finished and involves one or more PVCs", func() {
				var (
					pvc      *k8sv1.PersistentVolumeClaim
					snapshot *snapshotv1.VirtualMachineSnapshot
					restore  *snapshotv1.VirtualMachineRestore
				)

				BeforeEach(func() {
					snapshot = createVirtualMachineSnapshot(sourceVM, createOwnerReference(vmClone))
					snapshot.Status.ReadyToUse = pointer.P(true)

					pvc = createPVC(sourceVM.Namespace, k8sv1.ClaimPending)

					restore = createVirtualMachineRestore(sourceVM, snapshot.Name, createOwnerReference(vmClone))
					restore.Status.Complete = pointer.P(true)
					restore.Status.Restores = []snapshotv1.VolumeRestore{
						{PersistentVolumeClaimName: pvc.Name},
					}

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.RestoreName = pointer.P(restore.Name)
					vmClone.Status.Phase = clone.Succeeded
					vmClone.Status.TargetName = pointer.P(vmClone.Spec.Target.Name)

					targetVM := sourceVM.DeepCopy()
					targetVM.Name = vmClone.Spec.Target.Name

					addVM(sourceVM)
					addVM(targetVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)
					addPVC(pvc)
				})

				It("if not all the PVCs are bound, nothing should happen", func() {
					sanityExecute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clone.Succeeded)
					expectSnapshotExists()
					expectRestoreExists()
				})

				It("if all the pvc are bound, snapshot and restore should be deleted", func() {
					pvc = createPVC(sourceVM.Namespace, k8sv1.ClaimBound)
					addPVC(pvc)
					sanityExecute()
					expectEvent(PVCBound)
					expectSnapshotDoesNotExist()
					expectRestoreDoesNotExist()
				})
			})

			It("when snapshot is deleted before restore is ready - should fail", func() {
				restore := createVirtualMachineRestore(sourceVM, testSnapshotName)
				restore.Status.Complete = pointer.P(false)

				vmClone.Status.SnapshotName = pointer.P(testSnapshotName)
				vmClone.Status.RestoreName = pointer.P(restore.Name)
				vmClone.Status.Phase = clone.RestoreInProgress

				addVM(sourceVM)
				addClone(vmClone)
				addRestore(restore)

				sanityExecute()
				expectEvent(SnapshotDeleted)
				expectCloneBeInPhase(clone.Failed)
				expectSnapshotDoesNotExist()
			})

			It("when snapshot already exists and vmclone is not update yet- should update the clone phase", func() {
				vmClone.Status.Phase = clone.PhaseUnset

				addVM(sourceVM)
				addClone(vmClone)

				snapshot := createVirtualMachineSnapshot(sourceVM)
				addSnapshot(snapshot)

				sanityExecute()
				expectCloneBeInPhase(clone.SnapshotInProgress)
			})

			When("restore already exists and vmclone is not updated yet", func() {
				It("should update the clone phase", func() {
					snapshot := createVirtualMachineSnapshot(sourceVM)
					snapshot.Status.ReadyToUse = pointer.P(true)
					snapshotContent := createVirtualMachineSnapshotContent(sourceVM)

					restore := createVirtualMachineRestore(sourceVM, snapshot.Name)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)

					vmClone.Status.Phase = clone.SnapshotInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addSnapshotContent(snapshotContent)
					addRestore(restore)

					sanityExecute()
					expectCloneBeInPhase(clone.RestoreInProgress)
					expectEvent(SnapshotReady)
				})
			})

			When("target VM already exists", func() {
				It("should fire the related event", func() {
					snapshot := createVirtualMachineSnapshot(sourceVM)
					snapshotContent := createVirtualMachineSnapshotContent(sourceVM)
					snapshot.Status.ReadyToUse = pointer.P(true)
					restore := createVirtualMachineRestore(sourceVM, snapshot.Name)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)

					vmClone.Status.Phase = clone.SnapshotInProgress

					expectRestoreCreationFailure(snapshot.Name, vmClone, restore.Name)
					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addSnapshotContent(snapshotContent)

					sanityExecute()
					expectCloneBeInPhase(clone.RestoreInProgress)
					expectEvent(SnapshotReady)
					expectEvent(RestoreCreationFailed)
				})
			})

			When("target VM is deleted", func() {
				It("should delete the vm clone resource", func() {
					vmClone.Status.TargetName = pointer.P("target-vm-name")
					vmClone.Status.Phase = clone.Succeeded

					addVM(sourceVM)
					addClone(vmClone)
					// no target vm added => target vm deleted

					sanityExecute()
					expectCloneDeletion()
				})
			})
		})

		Context("with source snapshot", func() {
			It("should report event if source Snapshot doesn't exist, phase shouldn't change", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				setSnapshotSource(vmClone, snapshot.Name)

				addClone(vmClone)

				sanityExecute()
				expectEvent(SourceDoesNotExist)
				expectCloneBeInPhase(clone.PhaseUnset)
			})

			It("when snapshot is not ready yet - should not do anything", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(false)

				setSnapshotSource(vmClone, snapshot.Name)

				vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
				vmClone.Status.Phase = clone.SnapshotInProgress

				addClone(vmClone)
				addSnapshot(snapshot)

				sanityExecute()
				Expect(recorder.Events).To(BeEmpty())
				expectCloneBeInPhase(clone.SnapshotInProgress)
				expectRestoreDoesNotExist()
			})

			It("should fail clone if snaphshot ready - but snapshotcontent is invalid", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(true)
				snapshotContent := createVirtualMachineSnapshotContent(sourceVM)
				snapshotContent.Spec.VirtualMachineSnapshotName = nil
				setSnapshotSource(vmClone, snapshot.Name)

				addClone(vmClone)
				addSnapshot(snapshot)
				addSnapshotContent(snapshotContent)

				sanityExecute()
				expectEvent(SnapshotContentInvalid)
				expectCloneBeInPhase(clone.Failed)
			})

			It("clone should fail if source VMSnapshot has backendstorage", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(true)
				snapshotContent := createVirtualMachineSnapshotContent(sourceVM)
				snapshotContent.Spec.Source.VirtualMachine.Spec.Template.Spec.Domain.Devices.TPM = &virtv1.TPMDevice{
					Persistent: pointer.P(true),
				}

				setSnapshotSource(vmClone, snapshot.Name)

				addClone(vmClone)
				addSnapshot(snapshot)
				addSnapshotContent(snapshotContent)

				sanityExecute()
				expectEvent(SnapshotContentInvalid)
				expectCloneBeInPhase(clone.Failed)
			})

			It("should fail clone if snaphshot ready - but not all volumes were snapshoted", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(true)
				snapshotContent := createVirtualMachineSnapshotContent(sourceVM)
				snapshotContent.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes = append(snapshotContent.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes, virtv1.Volume{
					Name: "vol1",
					VolumeSource: virtv1.VolumeSource{
						DataVolume: &virtv1.DataVolumeSource{
							Name: "dv1",
						},
					},
				})
				setSnapshotSource(vmClone, snapshot.Name)

				addClone(vmClone)
				addSnapshot(snapshot)
				addSnapshotContent(snapshotContent)

				sanityExecute()
				expectEvent(SnapshotContentInvalid)
				expectCloneBeInPhase(clone.Failed)
			})

			It("when snapshot is ready - should update status and create restore", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(true)
				setSnapshotSource(vmClone, snapshot.Name)
				snapshotContent := createVirtualMachineSnapshotContent(sourceVM)

				addClone(vmClone)
				addSnapshot(snapshot)
				addSnapshotContent(snapshotContent)

				sanityExecute()
				expectEvent(SnapshotReady)
				expectEvent(RestoreCreated)
				expectCloneBeInPhase(clone.RestoreInProgress)
				expectRestoreExists()
			})

			It("when restore already exists and vmclone is not update yet - should update the clone phase", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(true)
				snapshotContent := createVirtualMachineSnapshotContent(sourceVM)

				setSnapshotSource(vmClone, snapshot.Name)

				restore := createVirtualMachineRestore(sourceVM, snapshot.Name)

				vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
				vmClone.Status.Phase = clone.PhaseUnset

				addClone(vmClone)
				addSnapshot(snapshot)
				addSnapshotContent(snapshotContent)
				addRestore(restore)

				sanityExecute()
				expectCloneBeInPhase(clone.RestoreInProgress)
			})
		})
	})

	Context("generation of target VM", func() {
		BeforeEach(func() {
			snapshot := createVirtualMachineSnapshot(sourceVM)
			snapshot.Status.ReadyToUse = pointer.P(true)

			vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
			vmClone.Status.Phase = clone.RestoreInProgress

			addVM(sourceVM)
			addSnapshot(snapshot)
			content := createVirtualMachineSnapshotContent(sourceVM)
			addSnapshotContent(content)
		})

		AfterEach(func() {
			expectEvent(RestoreCreated)
			expectCloneBeInPhase(clone.RestoreInProgress)
		})

		offlinePatchVM := func(vm *virtv1.VirtualMachine, patches []string) (virtv1.VirtualMachine, error) {
			patchedVM := virtv1.VirtualMachine{}

			marshalledVM, err := json.Marshal(vm)
			if err != nil {
				return patchedVM, fmt.Errorf("cannot marshall VM %s: %v", vm.Name, err)
			}

			jsonPatch := "[\n" + strings.Join(patches, ",\n") + "\n]"

			patch, err := jsonpatch.DecodePatch([]byte(jsonPatch))
			if err != nil {
				return patchedVM, fmt.Errorf("cannot decode vm patches %s: %v", jsonPatch, err)
			}

			modifiedMarshalledVM, err := patch.Apply(marshalledVM)
			if err != nil {
				return patchedVM, fmt.Errorf("failed to apply patch for VM %s: %v", jsonPatch, err)
			}

			err = json.Unmarshal(modifiedMarshalledVM, &patchedVM)
			if err != nil {
				return patchedVM, fmt.Errorf("cannot unmarshal modified marshalled vm %s: %v", string(modifiedMarshalledVM), err)
			}

			return patchedVM, nil
		}

		expectVMCreationFromPatches := func(expectedVM *virtv1.VirtualMachine) {
			restore, err := client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(restore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
			patchedVM, err := offlinePatchVM(sourceVM, restore.Spec.Patches)
			Expect(err).ToNot(HaveOccurred())
			Expect(patchedVM.Spec).To(Equal(expectedVM.Spec))
		}

		Context("MAC address", func() {
			var macAddressesCreatedCount int

			generateNewMacAddress := func() string {
				const macAddressPattern = "DE-AD-00-00-BE-0%d"
				address := fmt.Sprintf(macAddressPattern, macAddressesCreatedCount)
				macAddressesCreatedCount++
				Expect(macAddressesCreatedCount).To(BeNumerically("<", 10), "test assumes less then 10 new macs created")

				return address
			}

			BeforeEach(func() {
				macAddressesCreatedCount = 0
			})

			It("should delete mac address if manual mac address is not provided", func() {
				interfaces := sourceVM.Spec.Template.Spec.Domain.Devices.Interfaces
				Expect(interfaces).To(HaveLen(1))
				interfaces[0].Name = "test-interface"
				interfaces[0].MacAddress = generateNewMacAddress()

				vmClone.Spec.NewMacAddresses = map[string]string{}
				addClone(vmClone)

				expectedVM := sourceVM.DeepCopy()
				expectedInterfaces := expectedVM.Spec.Template.Spec.Domain.Devices.Interfaces
				expectedInterfaces[0].MacAddress = ""

				sanityExecute()
				expectVMCreationFromPatches(expectedVM)
			})

			It("if mac is defined in clone spec - should use the one in clone spec", func() {
				interfaces := sourceVM.Spec.Template.Spec.Domain.Devices.Interfaces
				Expect(interfaces).To(HaveLen(1))
				interfaces[0].Name = "test-interface"
				interfaces[0].MacAddress = generateNewMacAddress()

				newMacAddress := generateNewMacAddress()

				vmClone.Spec.NewMacAddresses = map[string]string{interfaces[0].Name: newMacAddress}
				addClone(vmClone)

				expectedVM := sourceVM.DeepCopy()
				expectedInterfaces := expectedVM.Spec.Template.Spec.Domain.Devices.Interfaces
				expectedInterfaces[0].MacAddress = newMacAddress

				sanityExecute()
				expectVMCreationFromPatches(expectedVM)
			})

			It("should handle multiple patches", func() {
				const numberOfDevicesToAdd = 6
				const interfaceNamePattern = "test-interface-%d"

				newMacAddresses := make(map[string]string, numberOfDevicesToAdd/2)
				originalInterfaces := make([]virtv1.Interface, numberOfDevicesToAdd)
				expectedInterfaces := make([]virtv1.Interface, numberOfDevicesToAdd)

				// Changing only even addresses
				for i := 0; i < numberOfDevicesToAdd; i++ {
					iface := virtv1.Interface{}
					iface.Name = fmt.Sprintf(interfaceNamePattern, i)
					iface.MacAddress = generateNewMacAddress()

					originalInterfaces[i] = iface

					if i%2 == 0 {
						iface.MacAddress = generateNewMacAddress()
						newMacAddresses[iface.Name] = iface.MacAddress
					} else {
						// empty MAC for others
						iface.MacAddress = ""
					}
					expectedInterfaces[i] = iface
				}

				sourceVM.Spec.Template.Spec.Domain.Devices.Interfaces = originalInterfaces

				vmClone.Spec.NewMacAddresses = newMacAddresses
				addClone(vmClone)

				expectedVM := sourceVM.DeepCopy()
				expectedVM.Spec.Template.Spec.Domain.Devices.Interfaces = expectedInterfaces

				sanityExecute()
				expectVMCreationFromPatches(expectedVM)
			})
		})

		Context("SMBios Serial", func() {
			const manuallySetSerial = "manually-set-serial"
			const originalSerial = "original-serial"
			const emptySerial = ""

			BeforeEach(func() {
				sourceVM.Spec.Template.Spec.Domain.Firmware = &virtv1.Firmware{Serial: originalSerial}
			})

			expectSMbiosSerial := func(serial string) {
				expectedVM := sourceVM.DeepCopy()
				expectedFirmware := expectedVM.Spec.Template.Spec.Domain.Firmware
				Expect(expectedFirmware).ShouldNot(BeNil())
				expectedFirmware.Serial = serial

				expectVMCreationFromPatches(expectedVM)
			}

			It("should delete smbios serial if serial is not provided", func() {
				addClone(vmClone)

				sanityExecute()
				expectSMbiosSerial(emptySerial)
			})

			It("if serial is defined in clone spec - should use the one in clone spec", func() {
				vmClone.Spec.NewSMBiosSerial = pointer.P(manuallySetSerial)
				addClone(vmClone)
				sanityExecute()
				expectSMbiosSerial(manuallySetSerial)
			})
		})

		Context("Labels and annotations", func() {
			type mapType string
			const labels mapType = "labels"
			const annotations mapType = "annotations"

			expectLabelsOrAnnotations := func(m map[string]string, labelOrAnnotation mapType) {
				expectedVM := sourceVM.DeepCopy()

				if labelOrAnnotation == labels {
					expectedVM.Labels = m
				} else {
					expectedVM.Annotations = m
				}

				expectVMCreationFromPatches(expectedVM)
			}

			DescribeTable("filters", func(labelOrAnnotation mapType) {
				const trueStr = "true"
				m := map[string]string{
					"prefix1/something1":    trueStr,
					"prefix1/something2":    trueStr,
					"prefix2/something1":    trueStr,
					"prefix2/something2":    trueStr,
					"somePrefix/something":  trueStr,
					"somePrefix2/something": trueStr,
				}

				filters := []string{
					"prefix*",
					"!prefix1/something2",
					"!prefix2/*",
					"somePrefix2/something",
				}

				if labelOrAnnotation == labels {
					sourceVM.Labels = m
					vmClone.Spec.LabelFilters = filters
				} else {
					sourceVM.Annotations = m
					vmClone.Spec.AnnotationFilters = filters
				}
				addClone(vmClone)
				sanityExecute()
				expectLabelsOrAnnotations(map[string]string{
					"prefix1/something1":    trueStr,
					"somePrefix2/something": trueStr,
				}, labelOrAnnotation)
			},
				Entry("with labels", labels),
				Entry("with annotations", annotations),
			)

			It("should not strip lastRestoreUID annotation from newly created VM", func() {
				if sourceVM.Annotations == nil {
					sourceVM.Annotations = make(map[string]string)
				}
				sourceVM.Annotations["restore.kubevirt.io/lastRestoreUID"] = "bar"
				// controller mutates original ptrs annotations
				sourceVMCpy := sourceVM.DeepCopy()
				vmClone.Spec.AnnotationFilters = []string{"somekey/*"}
				addVM(sourceVM)
				addClone(vmClone)

				sanityExecute()
				restore, err := client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
				expectedPatches, err := generateStringPatchOperations(patch.New(patch.WithReplace("/spec/template/spec/domain/devices/interfaces/0/macAddress", "")))
				Expect(err).ToNot(HaveOccurred())
				Expect(restore.Spec.Patches).To(Equal(expectedPatches))
				patchedVM, err := offlinePatchVM(sourceVMCpy, restore.Spec.Patches)
				Expect(err).ToNot(HaveOccurred())
				Expect(patchedVM.Annotations).To(HaveKey("restore.kubevirt.io/lastRestoreUID"))
			})

			It("should generate patches from the vmsnapshotcontent, instead of the current VM", func() {
				if sourceVM.Annotations == nil {
					sourceVM.Annotations = make(map[string]string)
				}
				// add annotation to create dis-alignment between vm and vmsnapshotcontent
				sourceVM.Annotations["new_annotation_matching_filter"] = "ok"
				vmClone.Spec.AnnotationFilters = []string{"somekey/*"}
				addVM(sourceVM)
				addClone(vmClone)

				sanityExecute()
				restore, err := client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
				partialExpectedPatches, err := generateStringPatchOperations(patch.New(patch.WithRemove("/metadata/annotations/new_annotation_matching_filter")))
				Expect(err).ToNot(HaveOccurred())
				Expect(restore.Spec.Patches).ToNot(ContainElement(partialExpectedPatches[0]))
			})
		})

		Context("Firmware UUID", func() {
			const sourceFakeUUID = "source-fake-uuid"

			BeforeEach(func() {
				sourceVM.Spec.Template.Spec.Domain.Firmware = &virtv1.Firmware{UUID: sourceFakeUUID}
			})

			It("should strip firmware UUID from VM", func() {
				addClone(vmClone)

				expectedVM := sourceVM.DeepCopy()

				expectedFirmware := expectedVM.Spec.Template.Spec.Domain.Firmware
				Expect(expectedFirmware).ShouldNot(BeNil())

				expectedFirmware.UUID = ""
				sanityExecute()
				expectVMCreationFromPatches(expectedVM)
			})
		})

		Context("Target VM name", func() {
			expectTargetVMNameExist := func() {
				restore, err := client.SnapshotV1beta1().VirtualMachineRestores(metav1.NamespaceDefault).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
				patchedVM, err := offlinePatchVM(sourceVM, restore.Spec.Patches)
				Expect(err).ToNot(HaveOccurred())
				Expect(patchedVM.Name).ToNot(BeNil())
			}

			It("should generate target vm name if it is not provided", func() {
				vmClone.Spec.Target = &k8sv1.TypedLocalObjectReference{
					APIGroup: pointer.P(vmAPIGroup),
					Kind:     "VirtualMachine",
				}
				addClone(vmClone)
				controller.Execute()
				expectTargetVMNameExist()
			})
		})
	})
})

func createVirtualMachineSnapshot(vm *virtv1.VirtualMachine, owner ...metav1.OwnerReference) *snapshotv1.VirtualMachineSnapshot {
	return &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testSnapshotName,
			Namespace:       vm.Namespace,
			UID:             "snapshot-UID",
			OwnerReferences: owner,
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.P(vmAPIGroup),
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
		},
		Status: &snapshotv1.VirtualMachineSnapshotStatus{},
	}
}

func createVirtualMachineSnapshotContent(vm *virtv1.VirtualMachine) *snapshotv1.VirtualMachineSnapshotContent {
	return &snapshotv1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSnapshotContentName,
			Namespace: vm.Namespace,
			UID:       "snapshotcontent-UID",
		},
		Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
			Source: snapshotv1.SourceSpec{
				VirtualMachine: &snapshotv1.VirtualMachine{
					ObjectMeta: vm.ObjectMeta,
					Spec:       vm.Spec,
					Status:     vm.Status,
				},
			},
			VirtualMachineSnapshotName: pointer.P(testSnapshotName),
		},
	}
}

func createVirtualMachineRestore(vm *virtv1.VirtualMachine, snapshotName string, owner ...metav1.OwnerReference) *snapshotv1.VirtualMachineRestore {
	return &snapshotv1.VirtualMachineRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testRestoreName,
			Namespace:       vm.Namespace,
			UID:             "restore-UID",
			OwnerReferences: owner,
		},
		Spec: snapshotv1.VirtualMachineRestoreSpec{
			Target: k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.P(vmAPIGroup),
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
			VirtualMachineSnapshotName: snapshotName,
		},
		Status: &snapshotv1.VirtualMachineRestoreStatus{},
	}
}

func createPVC(namespace string, phase k8sv1.PersistentVolumeClaimPhase) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-pvc",
			Namespace: namespace,
			UID:       "pvc-UID",
		},
		Status: k8sv1.PersistentVolumeClaimStatus{
			Phase: phase,
		},
	}
}

func validateOwnerReference(ownerRef metav1.OwnerReference, expectedOwner metav1.Object) {
	const err = "owner reference verification failed"

	Expect(ownerRef.UID).To(Equal(expectedOwner.GetUID()), err)
	Expect(ownerRef.Name).To(Equal(expectedOwner.GetName()), err)
	Expect(ownerRef.Kind).To(Equal(clone.VirtualMachineCloneKind.Kind), err)
	Expect(ownerRef.APIVersion).To(Equal(clone.VirtualMachineCloneKind.GroupVersion().String()), err)
	Expect(ownerRef.BlockOwnerDeletion).ToNot(BeNil(), err)
	Expect(*ownerRef.BlockOwnerDeletion).To(BeTrue(), err)
	Expect(ownerRef.Controller).ToNot(BeNil(), err)
	Expect(*ownerRef.Controller).To(BeTrue(), err)
}

func createOwnerReference(owner metav1.Object) metav1.OwnerReference {
	return metav1.OwnerReference{
		UID:                owner.GetUID(),
		Name:               owner.GetName(),
		Kind:               clone.VirtualMachineCloneKind.Kind,
		APIVersion:         clone.VirtualMachineCloneKind.GroupVersion().String(),
		BlockOwnerDeletion: pointer.P(true),
		Controller:         pointer.P(true),
	}
}
