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
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1alpha1 "kubevirt.io/api/snapshot/v1alpha1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
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
		ctrl                    *gomock.Controller
		vmInterface             *kubecli.MockVirtualMachineInterface
		vmInformer              cache.SharedIndexInformer
		snapshotInformer        cache.SharedIndexInformer
		restoreInformer         cache.SharedIndexInformer
		snapshotContentInformer cache.SharedIndexInformer
		pvcInformer             cache.SharedIndexInformer
		cloneInformer           cache.SharedIndexInformer
		cloneSource             *framework.FakeControllerSource
		stop                    chan struct{}

		controller *VMCloneController
		recorder   *record.FakeRecorder
		mockQueue  *testutils.MockWorkQueue

		client        *kubevirtfake.Clientset
		k8sClient     *k8sfake.Clientset
		testNamespace string
		sourceVM      *virtv1.VirtualMachine
		vmClone       *clonev1alpha1.VirtualMachineClone
	)

	syncCaches := func(stop chan struct{}) {
		go vmInformer.Run(stop)
		go snapshotInformer.Run(stop)
		go restoreInformer.Run(stop)
		go cloneInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, snapshotInformer.HasSynced,
			restoreInformer.HasSynced, cloneInformer.HasSynced)).To(BeTrue())
	}

	addVM := func(vm *virtv1.VirtualMachine) {
		err := vmInformer.GetStore().Add(vm)
		Expect(err).ShouldNot(HaveOccurred())
	}

	addClone := func(vmClone *clonev1alpha1.VirtualMachineClone) {
		var err error
		vmClone, err = client.CloneV1alpha1().VirtualMachineClones(testNamespace).Create(context.TODO(), vmClone, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		mockQueue.ExpectAdds(1)
		cloneSource.Add(vmClone)
		mockQueue.Wait()
	}

	addSnapshot := func(snapshot *snapshotv1alpha1.VirtualMachineSnapshot) {
		var err error
		snapshot, err = client.SnapshotV1alpha1().VirtualMachineSnapshots(testNamespace).Create(context.TODO(), snapshot, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		err = snapshotInformer.GetStore().Add(snapshot)
		Expect(err).ToNot(HaveOccurred())
	}

	addSnapshotContent := func(snapshotContent *snapshotv1alpha1.VirtualMachineSnapshotContent) {
		err := snapshotContentInformer.GetStore().Add(snapshotContent)
		Expect(err).ShouldNot(HaveOccurred())
	}

	addRestore := func(restore *snapshotv1alpha1.VirtualMachineRestore) {
		var err error
		restore, err = client.SnapshotV1alpha1().VirtualMachineRestores(testNamespace).Create(context.TODO(), restore, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		err = restoreInformer.GetStore().Add(restore)
		Expect(err).ToNot(HaveOccurred())
	}

	addPVC := func(pvc *k8sv1.PersistentVolumeClaim) {
		err := pvcInformer.GetStore().Add(pvc)
		Expect(err).ShouldNot(HaveOccurred())
	}

	expectSnapshotExists := func() {
		vmSnapshot, err := client.SnapshotV1alpha1().VirtualMachineSnapshots(testNamespace).Get(context.TODO(), testSnapshotName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmSnapshot).ToNot(BeNil())
		Expect(vmSnapshot.Spec.Source.Kind).To(Equal("VirtualMachine"))
		Expect(vmSnapshot.OwnerReferences).To(HaveLen(1))
		validateOwnerReference(vmSnapshot.OwnerReferences[0], vmClone)
	}

	expectSnapshotDoesNotExist := func() {
		_, err := client.SnapshotV1alpha1().VirtualMachineSnapshots(testNamespace).Get(context.TODO(), testSnapshotName, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Snapshot should not exists")
	}

	expectRestoreExists := func() {
		vmRestore, err := client.SnapshotV1alpha1().VirtualMachineRestores(testNamespace).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmRestore).ToNot(BeNil())
		Expect(vmRestore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
		Expect(vmRestore.OwnerReferences).To(HaveLen(1))
		validateOwnerReference(vmRestore.OwnerReferences[0], vmClone)
	}

	expectRestoreCreationFailure := func(snapshotName string, vmClone *clonev1alpha1.VirtualMachineClone, restoreName string) {
		client.Fake.PrependReactor("create", restoreResource, func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			restorecreated := create.GetObject().(*snapshotv1alpha1.VirtualMachineRestore)
			Expect(restorecreated.Spec.VirtualMachineSnapshotName).To(Equal(snapshotName))
			Expect(restorecreated.OwnerReferences).To(HaveLen(1))
			validateOwnerReference(restorecreated.OwnerReferences[0], vmClone)

			return true, nil, fmt.Errorf("when snapshot source and restore target VMs are different - target VM must not exist")
		})
	}

	expectRestoreDoesNotExist := func() {
		_, err := client.SnapshotV1alpha1().VirtualMachineRestores(testNamespace).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Restore should not exists")
	}

	expectCloneBeInPhase := func(phase clonev1alpha1.VirtualMachineClonePhase) {
		clone, err := client.CloneV1alpha1().VirtualMachineClones(testNamespace).Get(context.TODO(), vmClone.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(clone).ToNot(BeNil())
		Expect(clone.Status.Phase).To(Equal(phase))
	}

	expectEvent := func(event Event) {
		testutils.ExpectEvent(recorder, string(event))
	}

	setSnapshotSource := func(vmClone *clonev1alpha1.VirtualMachineClone, snapshotName string) {
		source := vmClone.Spec.Source
		source.APIGroup = pointer.P(snapshotAPIGroup)
		source.Kind = "VirtualMachineSnapshot"
		source.Name = snapshotName
	}

	setupInformers := func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())

		testNamespace = util.NamespaceTestDefault

		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		snapshotInformer, _ = testutils.NewFakeInformerFor(&snapshotv1alpha1.VirtualMachineSnapshot{})
		restoreInformer, _ = testutils.NewFakeInformerFor(&snapshotv1alpha1.VirtualMachineRestore{})
		cloneInformer, cloneSource = testutils.NewFakeInformerFor(&clonev1alpha1.VirtualMachineClone{})
		snapshotContentInformer, _ = testutils.NewFakeInformerFor(&snapshotv1alpha1.VirtualMachineSnapshotContent{})
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
	}

	setupResources := func() {
		sourceVMI := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
		)
		sourceVMI.Namespace = testNamespace
		sourceVM = libvmi.NewVirtualMachine(sourceVMI)
		sourceVM.Spec.Running = nil
		runStrategy := virtv1.RunStrategyHalted
		sourceVM.Spec.RunStrategy = &runStrategy

		vmClone = kubecli.NewMinimalCloneWithNS("testclone", util.NamespaceTestDefault)
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
		setupInformers()
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

		virtClient.EXPECT().VirtualMachine(testNamespace).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineClone(util.NamespaceTestDefault).Return(client.CloneV1alpha1().VirtualMachineClones(util.NamespaceTestDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineSnapshot(util.NamespaceTestDefault).Return(client.SnapshotV1alpha1().VirtualMachineSnapshots(util.NamespaceTestDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineRestore(util.NamespaceTestDefault).Return(client.SnapshotV1alpha1().VirtualMachineRestores(util.NamespaceTestDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineSnapshotContent(util.NamespaceTestDefault).Return(client.SnapshotV1alpha1().VirtualMachineSnapshotContents(util.NamespaceTestDefault)).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

		syncCaches(stop)
	})

	Context("basic controller operations", func() {
		Context("with source VM", func() {
			DescribeTable("should create snapshot if not exists yet", func(phase clonev1alpha1.VirtualMachineClonePhase) {
				vmClone.Status.Phase = phase

				addVM(sourceVM)
				addClone(vmClone)

				controller.Execute()
				expectEvent(SnapshotCreated)
				expectSnapshotExists()
				expectCloneBeInPhase(clonev1alpha1.SnapshotInProgress)
			},
				Entry("with phase unset", clonev1alpha1.PhaseUnset),
				Entry("with phase snapshot in progress", clonev1alpha1.SnapshotInProgress),
			)

			When("snapshot is created", func() {
				var snapshot *snapshotv1alpha1.VirtualMachineSnapshot

				BeforeEach(func() {
					snapshot = createVirtualMachineSnapshot(sourceVM)
				})

				It("and is not ready yet - should not do anything", func() {
					snapshot.Status.ReadyToUse = pointer.P(false)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.Phase = clonev1alpha1.SnapshotInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)

					controller.Execute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clonev1alpha1.SnapshotInProgress)
					expectRestoreDoesNotExist()
				})

				It("and is ready - should update status and create restore", func() {
					snapshotContent := createVirtualMachineSnapshotContent(sourceVM)

					snapshot.Status.ReadyToUse = pointer.P(true)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.Phase = clonev1alpha1.SnapshotInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addSnapshotContent(snapshotContent)

					controller.Execute()
					expectEvent(SnapshotReady)
					expectEvent(RestoreCreated)
					expectCloneBeInPhase(clonev1alpha1.RestoreInProgress)
					expectRestoreExists()
				})
			})

			When("restore is created", func() {
				var (
					snapshot *snapshotv1alpha1.VirtualMachineSnapshot
					restore  *snapshotv1alpha1.VirtualMachineRestore
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
					vmClone.Status.Phase = clonev1alpha1.RestoreInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					controller.Execute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clonev1alpha1.RestoreInProgress)
				})

				It("and is ready - should update status", func() {
					restore.Status.Complete = pointer.P(true)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.RestoreName = pointer.P(restore.Name)
					vmClone.Status.Phase = clonev1alpha1.RestoreInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					controller.Execute()
					expectEvent(RestoreReady)
					expectCloneBeInPhase(clonev1alpha1.CreatingTargetVM)
				})
			})

			When("snapshot and restore are finished", func() {
				var (
					snapshot *snapshotv1alpha1.VirtualMachineSnapshot
					restore  *snapshotv1alpha1.VirtualMachineRestore
				)

				BeforeEach(func() {
					snapshot = createVirtualMachineSnapshot(sourceVM)
					snapshot.Status.ReadyToUse = pointer.P(true)
					restore = createVirtualMachineRestore(sourceVM, snapshot.Name)
					restore.Status.Complete = pointer.P(true)
					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.RestoreName = pointer.P(restore.Name)
					vmClone.Status.Phase = clonev1alpha1.CreatingTargetVM
				})

				It("and the target VM is not ready - should do nothing", func() {
					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					controller.Execute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clonev1alpha1.CreatingTargetVM)
				})

				It("and the target VM ready should move to Succeeded phase", func() {
					targetVM := sourceVM.DeepCopy()
					targetVM.Name = vmClone.Spec.Target.Name

					addVM(sourceVM)
					addVM(targetVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					controller.Execute()
					expectEvent(TargetVMCreated)
					expectCloneBeInPhase(clonev1alpha1.Succeeded)
					expectSnapshotDoesNotExist()
					expectRestoreDoesNotExist()
				})
			})

			When("the clone process is finished and involves one or more PVCs", func() {
				var (
					pvc      *k8sv1.PersistentVolumeClaim
					snapshot *snapshotv1alpha1.VirtualMachineSnapshot
					restore  *snapshotv1alpha1.VirtualMachineRestore
				)

				BeforeEach(func() {
					snapshot = createVirtualMachineSnapshot(sourceVM, createOwnerReference(vmClone))
					snapshot.Status.ReadyToUse = pointer.P(true)

					pvc = createPVC(sourceVM.Namespace, k8sv1.ClaimPending)

					restore = createVirtualMachineRestore(sourceVM, snapshot.Name, createOwnerReference(vmClone))
					restore.Status.Complete = pointer.P(true)
					restore.Status.Restores = []snapshotv1alpha1.VolumeRestore{
						{PersistentVolumeClaimName: pvc.Name},
					}

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
					vmClone.Status.RestoreName = pointer.P(restore.Name)
					vmClone.Status.Phase = clonev1alpha1.Succeeded

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
					controller.Execute()
					Expect(recorder.Events).To(BeEmpty())
					expectCloneBeInPhase(clonev1alpha1.Succeeded)
					expectSnapshotExists()
					expectRestoreExists()
				})

				It("if all the pvc are bound, snapshot and restore should be deleted", func() {
					pvc = createPVC(sourceVM.Namespace, k8sv1.ClaimBound)
					addPVC(pvc)
					controller.Execute()
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
				vmClone.Status.Phase = clonev1alpha1.RestoreInProgress

				addVM(sourceVM)
				addClone(vmClone)
				addRestore(restore)

				controller.Execute()
				expectEvent(SnapshotDeleted)
				expectCloneBeInPhase(clonev1alpha1.Failed)
				expectSnapshotDoesNotExist()
			})

			It("when snapshot already exists and vmclone is not update yet- should update the clone phase", func() {
				vmClone.Status.Phase = clonev1alpha1.PhaseUnset

				addVM(sourceVM)
				addClone(vmClone)

				snapshot := createVirtualMachineSnapshot(sourceVM)
				addSnapshot(snapshot)

				controller.Execute()
				expectCloneBeInPhase(clonev1alpha1.SnapshotInProgress)
			})

			When("restore already exists and vmclone is not updated yet", func() {
				It("should update the clone phase", func() {
					snapshot := createVirtualMachineSnapshot(sourceVM)
					snapshot.Status.ReadyToUse = pointer.P(true)

					restore := createVirtualMachineRestore(sourceVM, snapshot.Name)

					vmClone.Status.SnapshotName = pointer.P(snapshot.Name)

					vmClone.Status.Phase = clonev1alpha1.SnapshotInProgress

					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addRestore(restore)

					controller.Execute()
					expectCloneBeInPhase(clonev1alpha1.RestoreInProgress)
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

					vmClone.Status.Phase = clonev1alpha1.SnapshotInProgress

					expectRestoreCreationFailure(snapshot.Name, vmClone, restore.Name)
					addVM(sourceVM)
					addClone(vmClone)
					addSnapshot(snapshot)
					addSnapshotContent(snapshotContent)

					controller.Execute()
					expectCloneBeInPhase(clonev1alpha1.RestoreInProgress)
					expectEvent(SnapshotReady)
					expectEvent(RestoreCreationFailed)
				})
			})

		})

		Context("with source snapshot", func() {
			It("when snapshot is not ready yet - should not do anything", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(false)

				setSnapshotSource(vmClone, snapshot.Name)

				vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
				vmClone.Status.Phase = clonev1alpha1.SnapshotInProgress

				addClone(vmClone)
				addSnapshot(snapshot)

				controller.Execute()
				Expect(recorder.Events).To(BeEmpty())
				expectCloneBeInPhase(clonev1alpha1.SnapshotInProgress)
				expectRestoreDoesNotExist()
			})

			It("when snapshot is ready - should update status and create restore", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(true)
				setSnapshotSource(vmClone, snapshot.Name)
				snapshotContent := createVirtualMachineSnapshotContent(sourceVM)

				addClone(vmClone)
				addSnapshot(snapshot)
				addSnapshotContent(snapshotContent)

				controller.Execute()
				expectEvent(SnapshotReady)
				expectEvent(RestoreCreated)
				expectCloneBeInPhase(clonev1alpha1.RestoreInProgress)
				expectRestoreExists()
			})

			It("when restore already exists and vmclone is not update yet - should update the clone phase", func() {
				snapshot := createVirtualMachineSnapshot(sourceVM)
				snapshot.Status.ReadyToUse = pointer.P(true)

				setSnapshotSource(vmClone, snapshot.Name)

				restore := createVirtualMachineRestore(sourceVM, snapshot.Name)

				vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
				vmClone.Status.Phase = clonev1alpha1.PhaseUnset

				addClone(vmClone)
				addSnapshot(snapshot)
				addRestore(restore)

				controller.Execute()
				expectCloneBeInPhase(clonev1alpha1.RestoreInProgress)
			})
		})
	})

	Context("generation of target VM", func() {
		BeforeEach(func() {
			snapshot := createVirtualMachineSnapshot(sourceVM)
			snapshot.Status.ReadyToUse = pointer.P(true)

			vmClone.Status.SnapshotName = pointer.P(snapshot.Name)
			vmClone.Status.Phase = clonev1alpha1.RestoreInProgress

			addVM(sourceVM)
			addSnapshot(snapshot)
			content := createVirtualMachineSnapshotContent(sourceVM)
			addSnapshotContent(content)
		})

		AfterEach(func() {
			expectEvent(RestoreCreated)
			expectCloneBeInPhase(clonev1alpha1.RestoreInProgress)
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
			restore, err := client.SnapshotV1alpha1().VirtualMachineRestores(testNamespace).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
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

				controller.Execute()
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

				controller.Execute()
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

				controller.Execute()
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

				controller.Execute()
				expectSMbiosSerial(emptySerial)
			})

			It("if serial is defined in clone spec - should use the one in clone spec", func() {
				vmClone.Spec.NewSMBiosSerial = pointer.P(manuallySetSerial)
				addClone(vmClone)
				controller.Execute()
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
				controller.Execute()
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

				controller.Execute()
				restore, err := client.SnapshotV1alpha1().VirtualMachineRestores(testNamespace).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
				expectedPatches := []string{`{"op": "replace", "path": "/spec/template/spec/domain/devices/interfaces/0/macAddress", "value": ""}`}
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

				controller.Execute()
				restore, err := client.SnapshotV1alpha1().VirtualMachineRestores(testNamespace).Get(context.TODO(), testRestoreName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restore.Spec.VirtualMachineSnapshotName).To(Equal(testSnapshotName))
				Expect(restore.Spec.Patches).ToNot(ContainElement(`{"op": "remove", "path": "/metadata/annotations/new_annotation_matching_filter"}`))
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
				controller.Execute()
				expectVMCreationFromPatches(expectedVM)
			})
		})

	})
})

func createVirtualMachineSnapshot(vm *virtv1.VirtualMachine, owner ...metav1.OwnerReference) *snapshotv1alpha1.VirtualMachineSnapshot {
	return &snapshotv1alpha1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testSnapshotName,
			Namespace:       vm.Namespace,
			UID:             "snapshot-UID",
			OwnerReferences: owner,
		},
		Spec: snapshotv1alpha1.VirtualMachineSnapshotSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.P(vmAPIGroup),
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
		},
		Status: &snapshotv1alpha1.VirtualMachineSnapshotStatus{},
	}
}

func createVirtualMachineSnapshotContent(vm *virtv1.VirtualMachine) *snapshotv1alpha1.VirtualMachineSnapshotContent {
	return &snapshotv1alpha1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSnapshotContentName,
			Namespace: vm.Namespace,
			UID:       "snapshotcontent-UID",
		},
		Spec: snapshotv1alpha1.VirtualMachineSnapshotContentSpec{
			Source: snapshotv1alpha1.SourceSpec{
				VirtualMachine: &snapshotv1alpha1.VirtualMachine{
					ObjectMeta: vm.ObjectMeta,
					Spec:       vm.Spec,
					Status:     vm.Status,
				},
			},
		},
	}
}

func createVirtualMachineRestore(vm *virtv1.VirtualMachine, snapshotName string, owner ...metav1.OwnerReference) *snapshotv1alpha1.VirtualMachineRestore {
	return &snapshotv1alpha1.VirtualMachineRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testRestoreName,
			Namespace:       vm.Namespace,
			UID:             "restore-UID",
			OwnerReferences: owner,
		},
		Spec: snapshotv1alpha1.VirtualMachineRestoreSpec{
			Target: k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.P(vmAPIGroup),
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
			VirtualMachineSnapshotName: snapshotName,
		},
		Status: &snapshotv1alpha1.VirtualMachineRestoreStatus{},
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
	Expect(ownerRef.Kind).To(Equal(clonev1alpha1.VirtualMachineCloneKind.Kind), err)
	Expect(ownerRef.APIVersion).To(Equal(clonev1alpha1.VirtualMachineCloneKind.GroupVersion().String()), err)
	Expect(ownerRef.BlockOwnerDeletion).ToNot(BeNil(), err)
	Expect(*ownerRef.BlockOwnerDeletion).To(BeTrue(), err)
	Expect(ownerRef.Controller).ToNot(BeNil(), err)
	Expect(*ownerRef.Controller).To(BeTrue(), err)
}

func createOwnerReference(owner metav1.Object) metav1.OwnerReference {
	return metav1.OwnerReference{
		UID:                owner.GetUID(),
		Name:               owner.GetName(),
		Kind:               clonev1alpha1.VirtualMachineCloneKind.Kind,
		APIVersion:         clonev1alpha1.VirtualMachineCloneKind.GroupVersion().String(),
		BlockOwnerDeletion: pointer.P(true),
		Controller:         pointer.P(true),
	}
}
