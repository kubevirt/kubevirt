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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package watch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/client-go/api"

	virtv1 "kubevirt.io/api/core/v1"
	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	utiltype "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("Migration watcher", func() {

	var ctrl *gomock.Controller
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var migrationSource *framework.FakeControllerSource
	var vmiSource *framework.FakeControllerSource
	var podSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var podInformer cache.SharedIndexInformer
	var migrationInformer cache.SharedIndexInformer
	var nodeInformer cache.SharedIndexInformer
	var pdbInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *MigrationController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var podFeeder *testutils.PodFeeder
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset
	var networkClient *fakenetworkclient.Clientset
	var pvcInformer cache.SharedIndexInformer
	var qemuGid int64 = 107

	shouldExpectMigrationFinalizerRemoval := func(migration *virtv1.VirtualMachineInstanceMigration) {
		migrationInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) (interface{}, interface{}) {
			Expect(len(arg.(*virtv1.VirtualMachineInstanceMigration).Finalizers)).To(Equal(0))
			return arg, nil
		})
	}

	shouldExpectPodCreation := func(uid types.UID, migrationUid types.UID, expectedAntiAffinityCount int, expectedAffinityCount int, expectedNodeAffinityCount int) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Labels[virtv1.CreatedByLabel]).To(Equal(string(uid)))
			Expect(update.GetObject().(*k8sv1.Pod).Labels[virtv1.MigrationJobLabel]).To(Equal(string(migrationUid)))

			Expect(update.GetObject().(*k8sv1.Pod).Spec.Affinity).ToNot(BeNil())
			Expect(update.GetObject().(*k8sv1.Pod).Spec.Affinity.PodAntiAffinity).ToNot(BeNil())
			Expect(len(update.GetObject().(*k8sv1.Pod).Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(expectedAntiAffinityCount))

			if expectedAffinityCount > 0 {
				Expect(len(update.GetObject().(*k8sv1.Pod).Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(expectedAffinityCount))
			}
			if expectedNodeAffinityCount > 0 {
				Expect(len(update.GetObject().(*k8sv1.Pod).Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)).To(Equal(expectedNodeAffinityCount))
			}

			return true, update.GetObject(), nil
		})
	}

	shouldExpectPodDeletion := func() {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			_, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())
			return true, nil, nil
		})
	}

	shouldExpectAttachmentPodCreation := func(uid types.UID, migrationUid types.UID) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Labels[virtv1.MigrationJobLabel]).To(Equal(string(migrationUid)))

			Expect(update.GetObject().(*k8sv1.Pod).Spec.Affinity).ToNot(BeNil())
			Expect(update.GetObject().(*k8sv1.Pod).Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(len(update.GetObject().(*k8sv1.Pod).Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)).To(Equal(1))

			return true, update.GetObject(), nil
		})
	}

	shouldExpectPDBPatch := func(vmi *virtv1.VirtualMachineInstance, vmim *virtv1.VirtualMachineInstanceMigration) {
		kubeClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			patch, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			Expect(patch.GetPatchType()).To(Equal(types.StrategicMergePatchType))

			expectedPatch := fmt.Sprintf(`{"spec":{"minAvailable": 2},"metadata":{"labels":{"%s": "%s"}}}`, virtv1.MigrationNameLabel, vmim.Name)
			Expect(string(patch.GetPatch())).To(Equal(expectedPatch))

			pdb := newPDB(patch.GetName(), vmi, 2)
			pdb.Labels = map[string]string{
				virtv1.MigrationNameLabel: vmim.Name,
			}

			return true, pdb, nil
		})
	}

	shouldExpectGenericMigrationUpdate := func() {
		migrationInterface.EXPECT().UpdateStatus(gomock.Any()).DoAndReturn(func(arg interface{}) (interface{}, interface{}) {
			return arg, nil
		})
	}

	shouldExpectMigrationSchedulingState := func(migration *virtv1.VirtualMachineInstanceMigration) {
		migrationInterface.EXPECT().UpdateStatus(gomock.Any()).DoAndReturn(func(arg interface{}) (interface{}, interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstanceMigration).Status.Phase).To(Equal(virtv1.MigrationScheduling))
			return arg, nil
		})
	}

	shouldExpectMigrationPreparingTargetState := func(migration *virtv1.VirtualMachineInstanceMigration) {
		migrationInterface.EXPECT().UpdateStatus(gomock.Any()).DoAndReturn(func(arg interface{}) (interface{}, interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstanceMigration).Status.Phase).To(Equal(virtv1.MigrationPreparingTarget))
			return arg, nil
		})
	}

	shouldExpectMigrationTargetReadyState := func(migration *virtv1.VirtualMachineInstanceMigration) {
		migrationInterface.EXPECT().UpdateStatus(gomock.Any()).DoAndReturn(func(arg interface{}) (interface{}, interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstanceMigration).Status.Phase).To(Equal(virtv1.MigrationTargetReady))
			return arg, nil
		})
	}

	shouldExpectMigrationRunningState := func(migration *virtv1.VirtualMachineInstanceMigration) {
		migrationInterface.EXPECT().UpdateStatus(gomock.Any()).DoAndReturn(func(arg interface{}) (interface{}, interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstanceMigration).Status.Phase).To(Equal(virtv1.MigrationRunning))
			return arg, nil
		})
	}

	shouldExpectMigrationCompletedState := func(migration *virtv1.VirtualMachineInstanceMigration) {
		migrationInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) (interface{}, interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstanceMigration).Status.Phase).To(Equal(virtv1.MigrationSucceeded))
			return arg, nil
		})
	}

	shouldExpectMigrationFailedState := func(migration *virtv1.VirtualMachineInstanceMigration) {
		migrationInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) (interface{}, interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstanceMigration).Status.Phase).To(Equal(virtv1.MigrationFailed))
			return arg, nil
		})
	}

	shouldExpectVirtualMachineInstancePatch := func(vmi *virtv1.VirtualMachineInstance, patch string) {
		vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
	}

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go podInformer.Run(stop)
		go migrationInformer.Run(stop)
		go nodeInformer.Run(stop)
		go pdbInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			podInformer.HasSynced,
			migrationInformer.HasSynced,
			nodeInformer.HasSynced,
			pdbInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		migrationInformer, migrationSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstanceMigration{})
		podInformer, podSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		pdbInformer, _ = testutils.NewFakeInformerFor(&v1beta1.PodDisruptionBudget{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		nodeInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Node{})

		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})

		controller = NewMigrationController(
			services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid),
			vmiInformer,
			podInformer,
			migrationInformer,
			nodeInformer,
			pvcInformer,
			pdbInformer,
			recorder,
			virtClient,
			config,
		)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		podFeeder = testutils.NewPodFeeder(mockQueue, podSource)

		// Set up mock client
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Return(migrationInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8sv1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().PolicyV1beta1().Return(kubeClient.PolicyV1beta1()).AnyTimes()
		networkClient = fakenetworkclient.NewSimpleClientset()
		virtClient.EXPECT().NetworkClient().Return(networkClient).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		syncCaches(stop)
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})

	addVirtualMachineInstance := func(vmi *virtv1.VirtualMachineInstance) {
		sourcePod := newSourcePodForVirtualMachine(vmi)
		podInformer.GetStore().Add(sourcePod)
		mockQueue.ExpectAdds(1)
		vmiSource.Add(vmi)
		mockQueue.Wait()
	}

	addMigration := func(migration *virtv1.VirtualMachineInstanceMigration) {
		mockQueue.ExpectAdds(1)
		migrationSource.Add(migration)
		mockQueue.Wait()
	}

	addNode := func(node *k8sv1.Node) {
		err := nodeInformer.GetIndexer().Add(node)
		Expect(err).ShouldNot(HaveOccurred())
	}

	addPDB := func(pdb *v1beta1.PodDisruptionBudget) {
		err := pdbInformer.GetIndexer().Add(pdb)
		Expect(err).ShouldNot(HaveOccurred())
	}

	Context("Migration with hotplug volumes", func() {
		var (
			vmi           *virtv1.VirtualMachineInstance
			migration     *virtv1.VirtualMachineInstanceMigration
			targetPod     *k8sv1.Pod
			attachmentPod *k8sv1.Pod
		)

		BeforeEach(func() {
			vmi = newVirtualMachineWithHotplugVolume("testvmi", virtv1.Running)
			migration = newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			targetPod = newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			attachmentPod = newAttachmentPodForVirtualMachine(targetPod, migration, k8sv1.PodRunning)
		})

		It("should create target attachment pod", func() {
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(targetPod)
			shouldExpectAttachmentPodCreation(vmi.UID, migration.UID)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should set migration state to scheduling if attachment pod exists", func() {
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(targetPod)
			podFeeder.Add(attachmentPod)

			shouldExpectMigrationSchedulingState(migration)
			controller.Execute()
		})

		It("should hand pod over to target virt-handler if attachment pod is ready and running", func() {
			vmi.Status.NodeName = "node02"
			migration.Status.Phase = virtv1.MigrationScheduled
			targetPod.Spec.NodeName = "node01"
			targetPod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(targetPod)
			podFeeder.Add(attachmentPod)

			patch := fmt.Sprintf(`[{ "op": "add", "path": "/status/migrationState", "value": {"targetNode":"node01","targetPod":"%s","targetAttachmentPodUID":"%s","sourceNode":"node02","migrationUid":"testmigration"} }, { "op": "test", "path": "/metadata/labels", "value": {} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io/migrationTargetNodeName":"node01"} }]`, targetPod.Name, attachmentPod.UID)

			shouldExpectVirtualMachineInstancePatch(vmi, patch)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
		})

		It("should fail the migration if the attachment pod goes to final state", func() {
			attachmentPod.Status.Phase = k8sv1.PodFailed

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(targetPod)
			podFeeder.Add(attachmentPod)

			shouldExpectMigrationFailedState(migration)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedMigrationReason)
		})
	})

	Context("Migration object in pending state", func() {
		It("should create target pod", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			shouldExpectPodCreation(vmi.UID, migration.UID, 1, 0, 0)

			controller.Execute()
			testutils.ExpectEvents(recorder, SuccessfulCreatePodReason)
		})
		It("should not create target pod if multiple pods exist in a non finalized state for VMI", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			pod1 := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			pod1.Labels[virtv1.MigrationJobLabel] = "some other job"
			pod2 := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod2.Labels[virtv1.MigrationJobLabel] = "some other job"
			podInformer.GetStore().Add(pod1)
			podInformer.GetStore().Add(pod2)

			addMigration(migration)
			addVirtualMachineInstance(vmi)

			controller.Execute()
		})

		It("should create another target pods if only 4 migrations are in progress", func() {
			// It should create a pod for this one
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)

			// Ensure that 4 migrations are there which are in non-final state
			for i := 0; i < 4; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				vmi.Status.NodeName = fmt.Sprintf("node%v", i)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			// Add two pending migrations without a target pod to see that tye get ignored
			for i := 0; i < 2; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("xtestvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("xtestmigration%v", i), vmi.Name, virtv1.MigrationPending)
				vmi.Status.NodeName = fmt.Sprintf("node%v", i)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			shouldExpectPodCreation(vmi.UID, migration.UID, 1, 0, 0)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should not overload the cluster and only run 5 migrations in parallel", func() {
			// It should create a pod for this one if we would not limit migrations
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)

			// Ensure that 5 migrations are there which are in non-final state
			for i := 0; i < 5; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)
				vmi.Status.NodeName = fmt.Sprintf("node%v", i)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			controller.Execute()
		})

		It("should not overload the cluster and detect pending migrations as running if they have a target pod", func() {
			// It should create a pod for this one if we would not limit migrations
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)

			// Ensure that 3 migrations are there which are running
			for i := 0; i < 3; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)
				vmi.Status.NodeName = fmt.Sprintf("node%v", i)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			// Ensure that 2 migrations are pending but have a target pod
			for i := 0; i < 2; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("xtestvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("xtestmigration%v", i), vmi.Name, virtv1.MigrationPending)
				pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
				vmi.Status.NodeName = fmt.Sprintf("node%v", i)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
				podInformer.GetStore().Add(pod)
			}

			controller.Execute()
		})

		It("should create another target pods if there is only one outbound migration on the node", func() {
			// It should create a pod for this one
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)

			// Ensure that 4 migrations are there which are in non-final state
			for i := 0; i < 1; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			shouldExpectPodCreation(vmi.UID, migration.UID, 1, 0, 0)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should not overload the node and only run 2 outbound migrations in parallel", func() {
			// It should create a pod for this one if we would not limit migrations
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)

			// Ensure that 5 migrations are there which are in non-final state
			for i := 0; i < 2; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			controller.Execute()
		})

		It("should create target pod and not override existing affinity rules", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			antiAffinityTerm := k8sv1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"somelabel": "somekey",
					},
				},
				TopologyKey: "kubernetes.io/hostname",
			}
			affinityTerm := k8sv1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"someotherlabel": "someotherkey",
					},
				},
				TopologyKey: "kubernetes.io/hostname",
			}
			antiAffinityRule := &k8sv1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{antiAffinityTerm},
			}
			affinityRule := &k8sv1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{affinityTerm},
			}

			nodeAffinityRule := &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{"somenode"},
								},
							},
						},
					},
				},
			}

			vmi.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity:    nodeAffinityRule,
				PodAntiAffinity: antiAffinityRule,
				PodAffinity:     affinityRule,
			}

			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			shouldExpectPodCreation(vmi.UID, migration.UID, 2, 1, 1)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should place migration in scheduling state if pod exists", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationSchedulingState(migration)
			controller.Execute()
		})

		table.DescribeTable("should handle pod stuck in unschedulable state", func(phase virtv1.VirtualMachineInstanceMigrationPhase, shouldTimeout bool, timeLapse int64, annotationVal string) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)

			if annotationVal != "" {
				migration.Annotations[virtv1.MigrationUnschedulablePodTimeoutSecondsAnnotation] = annotationVal
			}

			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)

			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{
				Type:   k8sv1.PodScheduled,
				Status: k8sv1.ConditionFalse,
				Reason: k8sv1.PodReasonUnschedulable,
			})
			now := now()
			pod.CreationTimestamp = metav1.NewTime(now.Time.Add(time.Duration(-timeLapse) * time.Second))

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			if shouldTimeout {
				shouldExpectPodDeletion()
			}

			if phase == virtv1.MigrationPending {
				shouldExpectGenericMigrationUpdate()
			}
			controller.Execute()

			if shouldTimeout {
				testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			}
		},
			table.Entry("in pending state", virtv1.MigrationPending, true, defaultUnschedulablePendingTimeoutSeconds, ""),
			table.Entry("in scheduling state", virtv1.MigrationScheduling, true, defaultUnschedulablePendingTimeoutSeconds, ""),
			table.Entry("in scheduled state", virtv1.MigrationScheduled, false, defaultUnschedulablePendingTimeoutSeconds, ""),
			table.Entry("in pending state but timeout not hit", virtv1.MigrationPending, false, defaultUnschedulablePendingTimeoutSeconds-1, ""),
			table.Entry("in pending state with custom timeout", virtv1.MigrationPending, true, int64(10), "10"),
			table.Entry("in pending state with custom timeout not hit", virtv1.MigrationPending, false, int64(10), "11"),
			table.Entry("in scheduling state but timeout not hit", virtv1.MigrationScheduling, false, defaultUnschedulablePendingTimeoutSeconds-1, ""),
		)

		table.DescribeTable("should handle pod stuck in pending phase for extended period of time", func(phase virtv1.VirtualMachineInstanceMigrationPhase, shouldTimeout bool, timeLapse int64, annotationVal string) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)

			now := now()
			pod.CreationTimestamp = metav1.NewTime(now.Time.Add(time.Duration(-timeLapse) * time.Second))

			if annotationVal != "" {
				migration.Annotations[virtv1.MigrationPendingPodTimeoutSecondsAnnotation] = annotationVal
			}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			if shouldTimeout {
				shouldExpectPodDeletion()
			}

			if phase == virtv1.MigrationPending {
				shouldExpectGenericMigrationUpdate()
			}
			controller.Execute()

			if shouldTimeout {
				testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			}
		},
			table.Entry("in pending state", virtv1.MigrationPending, true, defaultCatchAllPendingTimeoutSeconds, ""),
			table.Entry("in scheduling state", virtv1.MigrationScheduling, true, defaultCatchAllPendingTimeoutSeconds, ""),
			table.Entry("in scheduled state", virtv1.MigrationScheduled, false, defaultCatchAllPendingTimeoutSeconds, ""),
			table.Entry("in pending state but timeout not hit", virtv1.MigrationPending, false, defaultCatchAllPendingTimeoutSeconds-1, ""),
			table.Entry("in pending state with custom timeout", virtv1.MigrationPending, true, int64(10), "10"),
			table.Entry("in pending state with custom timeout not hit", virtv1.MigrationPending, false, int64(10), "11"),
			table.Entry("in scheduling state but timeout not hit", virtv1.MigrationScheduling, false, defaultCatchAllPendingTimeoutSeconds-1, ""),
		)
	})
	Context("Migration should immediately fail if", func() {

		table.DescribeTable("vmi moves to final state", func(phase virtv1.VirtualMachineInstanceMigrationPhase) {
			vmi := newVirtualMachine("testvmi", virtv1.Succeeded)
			vmi.DeletionTimestamp = now()
			migration := newMigration("testmigration", vmi.Name, phase)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
			}
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationFailedState(migration)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedMigrationReason)
		},
			table.Entry("in running state", virtv1.MigrationRunning),
			table.Entry("in unset state", virtv1.MigrationPhaseUnset),
			table.Entry("in pending state", virtv1.MigrationPending),
			table.Entry("in scheduled state", virtv1.MigrationScheduled),
			table.Entry("in scheduling state", virtv1.MigrationScheduling),
			table.Entry("in target ready state", virtv1.MigrationTargetReady),
		)
		table.DescribeTable("Pod moves to final state", func(phase virtv1.VirtualMachineInstanceMigrationPhase) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
			}
			if phase == virtv1.MigrationTargetReady {
				vmi.Status.MigrationState.StartTimestamp = now()
			}
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodSucceeded)
			pod.Spec.NodeName = "node01"

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationFailedState(migration)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedMigrationReason)
		},
			table.Entry("in running state", virtv1.MigrationRunning),
			table.Entry("in unset state", virtv1.MigrationPhaseUnset),
			table.Entry("in pending state", virtv1.MigrationPending),
			table.Entry("in scheduled state", virtv1.MigrationScheduled),
			table.Entry("in scheduling state", virtv1.MigrationScheduling),
			table.Entry("in target ready state", virtv1.MigrationTargetReady),
		)
		table.DescribeTable("VMI's migrate state moves to final state", func(phase virtv1.VirtualMachineInstanceMigrationPhase) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:   migration.UID,
				Failed:         true,
				Completed:      true,
				StartTimestamp: now(),
				EndTimestamp:   now(),
			}
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod.Spec.NodeName = "node01"

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationFailedState(migration)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedMigrationReason)
		},
			table.Entry("in running state", virtv1.MigrationRunning),
			table.Entry("in unset state", virtv1.MigrationPhaseUnset),
			table.Entry("in pending state", virtv1.MigrationPending),
			table.Entry("in scheduled state", virtv1.MigrationScheduled),
			table.Entry("in scheduling state", virtv1.MigrationScheduling),
			table.Entry("in target ready state", virtv1.MigrationTargetReady),
		)
	})
	Context("Migration object ", func() {

		table.DescribeTable("should hand pod over to target virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod.Spec.NodeName = "node01"
			pod.Status.ContainerStatuses = containerStatus

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			patch := fmt.Sprintf(`[{ "op": "add", "path": "/status/migrationState", "value": {"targetNode":"node01","targetPod":"%s","sourceNode":"node02","migrationUid":"testmigration"} }, { "op": "test", "path": "/metadata/labels", "value": {} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io/migrationTargetNodeName":"node01"} }]`, pod.Name)

			shouldExpectVirtualMachineInstancePatch(vmi, patch)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
		},
			table.Entry("with running compute container and no infra container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}},
			),
			table.Entry("with running compute container and no ready istio-proxy container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}, {Name: "istio-proxy", Ready: false}},
			),
		)

		table.DescribeTable("should not hand pod over to target virt-handler if pod is not ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod.Spec.NodeName = "node01"
			pod.Status.ContainerStatuses = containerStatus

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			controller.Execute()
		},
			table.Entry("with not ready infra container and not ready compute container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}, {Name: "kubevirt-infra", Ready: false}},
			),
			table.Entry("with not ready compute container and no infra container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}},
			),
		)

		It("should hand pod over to target virt-handler with migration config", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)

			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod.Spec.NodeName = "node01"
			pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			patch := fmt.Sprintf(`[{ "op": "add", "path": "/status/migrationState", "value": {"targetNode":"node01","targetPod":"%s","sourceNode":"node02","migrationUid":"testmigration"} }, { "op": "test", "path": "/metadata/labels", "value": {} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io/migrationTargetNodeName":"node01"} }]`, pod.Name)
			shouldExpectVirtualMachineInstancePatch(vmi, patch)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
		})

		It("should hand pod over to target virt-handler overriding previous state", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: "1111-2222-3333-4444",
			}
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod.Spec.NodeName = "node01"
			pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			patch := fmt.Sprintf(`[{ "op": "test", "path": "/status/migrationState", "value": {"migrationUid":"1111-2222-3333-4444"} }, { "op": "replace", "path": "/status/migrationState", "value": {"targetNode":"node01","targetPod":"%s","sourceNode":"node02","migrationUid":"testmigration"} }, { "op": "test", "path": "/metadata/labels", "value": {} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io/migrationTargetNodeName":"node01"} }]`, pod.Name)

			shouldExpectVirtualMachineInstancePatch(vmi, patch)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
		})

		It("should not hand pod over target pod that's already handed over", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
			}
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			pod.Spec.NodeName = "node01"

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			controller.Execute()

			// expect nothing to occur
		})

		It("should transition to preparing target phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
				TargetNode:   "node01",
				SourceNode:   "node02",
				TargetPod:    pod.Name,
			}
			vmi.Labels[virtv1.MigrationTargetNodeNameLabel] = "node01"
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationPreparingTargetState(migration)

			controller.Execute()
		})
		It("should transition to target prepared phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPreparingTarget)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			pod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationTargetReadyState(migration)

			controller.Execute()
		})
		It("should transition to running phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationTargetReady)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			pod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
				StartTimestamp:    now(),
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationRunningState(migration)

			controller.Execute()
		})
		It("should transition to completed phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationRunning)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			pod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
				StartTimestamp:    now(),
				EndTimestamp:      now(),
				Failed:            false,
				Completed:         true,
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			shouldExpectMigrationCompletedState(migration)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulMigrationReason)
		})
		It("should delete itself if VMI no longer exists", func() {
			migration := newMigration("testmigration", "somevmi", virtv1.MigrationRunning)
			addMigration(migration)

			migrationInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1).Return(nil)

			controller.Execute()
		})
		It("should abort the migration", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationRunning)
			condition := virtv1.VirtualMachineInstanceMigrationCondition{
				Type:          virtv1.VirtualMachineInstanceMigrationAbortRequested,
				Status:        k8sv1.ConditionTrue,
				LastProbeTime: *now(),
			}
			migration.Status.Conditions = append(migration.Status.Conditions, condition)
			pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			pod.Spec.NodeName = "node01"
			migration.DeletionTimestamp = now()
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
				StartTimestamp:    now(),
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Return(vmi, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulAbortMigrationReason)
		})
		table.DescribeTable("should finalize migration on VMI if target pod fails before migration starts", func(phase virtv1.VirtualMachineInstanceMigrationPhase, hasPod bool, podPhase k8sv1.PodPhase, initializeMigrationState bool) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = "node02"
			migration := newMigration("testmigration", vmi.Name, phase)

			vmi.Status.MigrationState = nil
			if initializeMigrationState {
				vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					MigrationUID: migration.UID,
					TargetNode:   "node01",
					SourceNode:   "node02",
				}
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			if hasPod {
				pod := newTargetPodForVirtualMachine(vmi, migration, podPhase)
				pod.Spec.NodeName = "node01"
				podFeeder.Add(pod)
			}

			if phase == virtv1.MigrationFailed {
				shouldExpectMigrationFinalizerRemoval(migration)
			} else {
				shouldExpectMigrationFailedState(migration)
			}

			if initializeMigrationState {
				patch := `[{ "op": "test", "path": "/status/migrationState", "value": {"targetNode":"node01","sourceNode":"node02","migrationUid":"testmigration"} }, { "op": "replace", "path": "/status/migrationState", "value": {"startTimestamp":"%s","endTimestamp":"%s","targetNode":"node01","sourceNode":"node02","completed":true,"failed":true,"migrationUid":"testmigration"} }]`

				vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).DoAndReturn(func(name interface{}, ptype interface{}, vmiStatusPatch []byte, options interface{}) (*virtv1.VirtualMachineInstance, error) {

					vmiSP := []utiltype.PatchOperation{}
					err := json.Unmarshal(vmiStatusPatch, &vmiSP)
					Expect(err).To(BeNil())
					Expect(vmiSP).To(HaveLen(2))

					b, err := json.Marshal(vmiSP[1].Value)
					Expect(err).To(BeNil())

					newMS := virtv1.VirtualMachineInstanceMigrationState{}
					err = json.Unmarshal(b, &newMS)
					Expect(err).To(BeNil())
					Expect(newMS.StartTimestamp).ToNot(BeNil())
					Expect(newMS.EndTimestamp).ToNot(BeNil())

					expected := fmt.Sprintf(patch, newMS.StartTimestamp.UTC().Format(time.RFC3339), newMS.EndTimestamp.UTC().Format(time.RFC3339))
					Expect(expected).To(Equal(string(vmiStatusPatch)))

					return vmi, nil
				})
			}

			controller.Execute()

			// in this case, we have two failed events. one for the VMI and one on the Migration object.
			if initializeMigrationState {
				testutils.ExpectEvent(recorder, FailedMigrationReason)
			}
			if phase != virtv1.MigrationFailed {
				testutils.ExpectEvent(recorder, FailedMigrationReason)
			}
		},
			table.Entry("in preparing target state", virtv1.MigrationPreparingTarget, true, k8sv1.PodFailed, true),
			table.Entry("in target ready state", virtv1.MigrationTargetReady, true, k8sv1.PodFailed, true),
			table.Entry("in failed state", virtv1.MigrationFailed, true, k8sv1.PodFailed, true),
			table.Entry("in failed state before pod is created", virtv1.MigrationFailed, false, k8sv1.PodFailed, false),
			table.Entry("in failed state and pod does not exist", virtv1.MigrationFailed, false, k8sv1.PodFailed, false),
		)
		table.DescribeTable("with CPU mode which is", func(toDefineHostModelCPU bool) {
			const nodeName = "testNode"

			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.NodeName = nodeName
			if toDefineHostModelCPU {
				vmi.Spec.Domain.CPU = &virtv1.CPU{Model: virtv1.CPUModeHostModel}
			}

			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			node := newNode(nodeName)
			if toDefineHostModelCPU {
				node.ObjectMeta.Labels = map[string]string{
					virtv1.HostModelCPULabel + "fake":              "true",
					virtv1.HostModelRequiredFeaturesLabel + "fake": "true",
				}
			}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addNode(node)

			expectPodToHaveProperNodeSelector := func(pod *k8sv1.Pod) {
				podHasCpuModeLabelSelector := false
				for key, _ := range pod.Spec.NodeSelector {
					if strings.Contains(key, virtv1.HostModelCPULabel) {
						podHasCpuModeLabelSelector = true
						break
					}
				}

				Expect(podHasCpuModeLabelSelector).To(Equal(toDefineHostModelCPU))
			}
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				creation, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				pod := creation.GetObject().(*k8sv1.Pod)
				expectPodToHaveProperNodeSelector(pod)
				return true, creation.GetObject(), nil
			})
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		},
			table.Entry("host-model should be targeted only to nodes which support the model", true),
			table.Entry("non-host-model should not be targeted to nodes which support the model", false),
		)
	})

	Context("Migration with protected VMI (PDB)", func() {
		It("should update PDB before starting the migration", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			evictionStrategy := virtv1.EvictionStrategyLiveMigrate
			vmi.Spec.EvictionStrategy = &evictionStrategy
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			pdb := newPDB("pdb-test", vmi, 1)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPDB(pdb)

			shouldExpectPDBPatch(vmi, migration)
			controller.Execute()

			testutils.ExpectEvents(recorder, successfulUpdatePodDisruptionBudgetReason)
		})
		It("should create the target Pod after the k8s PDB controller processed the PDB mutation", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			evictionStrategy := virtv1.EvictionStrategyLiveMigrate
			vmi.Spec.EvictionStrategy = &evictionStrategy
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			pdb := newPDB("pdb-test", vmi, 2)
			pdb.Generation = 42
			pdb.Status.DesiredHealthy = int32(pdb.Spec.MinAvailable.IntValue())
			pdb.Status.ObservedGeneration = pdb.Generation
			pdb.Labels = map[string]string{
				virtv1.MigrationNameLabel: migration.Name,
			}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPDB(pdb)

			shouldExpectPodCreation(vmi.UID, migration.UID, 1, 0, 0)

			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulCreatePodReason)
		})
	})
})

func newPDB(name string, vmi *virtv1.VirtualMachineInstance, pods int) *v1beta1.PodDisruptionBudget {
	minAvailable := intstr.FromInt(pods)

	return &v1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, virtv1.VirtualMachineInstanceGroupVersionKind),
			},
			Name:      name,
			Namespace: vmi.Namespace,
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					virtv1.CreatedByLabel: string(vmi.UID),
				},
			},
		},
	}
}

func newMigration(name string, vmiName string, phase virtv1.VirtualMachineInstanceMigrationPhase) *virtv1.VirtualMachineInstanceMigration {

	migration := &virtv1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k8sv1.NamespaceDefault,
			Annotations: map[string]string{
				virtv1.ControllerAPILatestVersionObservedAnnotation:  virtv1.ApiLatestVersion,
				virtv1.ControllerAPIStorageVersionObservedAnnotation: virtv1.ApiStorageVersion,
			},
		},
		Spec: virtv1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
	migration.TypeMeta = metav1.TypeMeta{
		APIVersion: virtv1.GroupVersion.String(),
		Kind:       "VirtualMachineInstanceMigration",
	}
	migration.UID = types.UID(name)
	migration.Status.Phase = phase
	return migration
}

func newVirtualMachine(name string, phase virtv1.VirtualMachineInstancePhase) *virtv1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI(name)
	vmi.UID = types.UID(name)
	vmi.Status.Phase = phase
	vmi.Status.NodeName = "tefwegwrerg"
	vmi.ObjectMeta.Labels = make(map[string]string)
	return vmi
}

func newVirtualMachineWithHotplugVolume(name string, phase virtv1.VirtualMachineInstancePhase) *virtv1.VirtualMachineInstance {
	vmi := newVirtualMachine(name, phase)
	vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
		{
			HotplugVolume: &virtv1.HotplugVolumeStatus{},
		},
	}
	return vmi
}

func newSourcePodForVirtualMachine(vmi *virtv1.VirtualMachineInstance) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: vmi.Namespace,
			Labels: map[string]string{
				virtv1.AppLabel:       "virt-launcher",
				virtv1.CreatedByLabel: string(vmi.UID),
			},
			Annotations: map[string]string{
				virtv1.DomainAnnotation: vmi.Name,
			},
		},
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "test"},
			},
		},
		Spec: k8sv1.PodSpec{
			NodeName: vmi.Status.NodeName,
			Volumes:  []k8sv1.Volume{},
		},
	}
}

func newTargetPodForVirtualMachine(vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: vmi.Namespace,
			Labels: map[string]string{
				virtv1.AppLabel:          "virt-launcher",
				virtv1.CreatedByLabel:    string(vmi.UID),
				virtv1.MigrationJobLabel: string(migration.UID),
			},
			Annotations: map[string]string{
				virtv1.DomainAnnotation:           vmi.Name,
				virtv1.MigrationJobNameAnnotation: migration.Name,
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "test"},
			},
		},
	}
}

func newAttachmentPodForVirtualMachine(ownerPod *k8sv1.Pod, migration *virtv1.VirtualMachineInstanceMigration, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: ownerPod.Namespace,
			UID:       "test-uid",
			Labels: map[string]string{
				virtv1.AppLabel:          "hotplug-disk",
				virtv1.MigrationJobLabel: string(migration.UID),
			},
			Annotations: map[string]string{
				virtv1.MigrationJobNameAnnotation: migration.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ownerPod, schema.GroupVersionKind{
					Group:   k8sv1.SchemeGroupVersion.Group,
					Version: k8sv1.SchemeGroupVersion.Version,
					Kind:    "Pod",
				}),
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "test"},
			},
		},
	}
}

func newNode(name string) *k8sv1.Node {
	node := &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: virtv1.GroupVersion.String(),
		},
	}

	node.Status.Phase = k8sv1.NodeRunning

	return node
}
