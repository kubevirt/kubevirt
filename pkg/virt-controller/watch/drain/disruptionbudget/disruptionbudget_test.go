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
 */

package disruptionbudget

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Disruptionbudget", func() {

	var ctrl *gomock.Controller
	var stop chan struct{}
	var virtClient *kubecli.MockKubevirtClient
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var pdbInformer cache.SharedIndexInformer
	var pdbSource *framework.FakeControllerSource
	var podInformer cache.SharedIndexInformer
	var vmimInformer cache.SharedIndexInformer
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue[string]
	var kubeClient *fake.Clientset
	var pdbFeeder *testutils.PodDisruptionBudgetFeeder[string]
	var vmiFeeder *testutils.VirtualMachineFeeder[string]
	var config *virtconfig.ClusterConfig

	var controller *DisruptionBudgetController

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go pdbInformer.Run(stop)
		go podInformer.Run(stop)
		go vmimInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			pdbInformer.HasSynced,
			podInformer.HasSynced,
			vmimInformer.HasSynced,
		)).To(BeTrue())
	}

	addVirtualMachine := func(vmi *v1.VirtualMachineInstance) {
		mockQueue.ExpectAdds(1)
		vmiSource.Add(vmi)
		mockQueue.Wait()
	}

	addMigration := func(vmim *v1.VirtualMachineInstanceMigration) {
		err := vmimInformer.GetIndexer().Add(vmim)
		Expect(err).ToNot(HaveOccurred())
	}

	addPod := func(pod *corev1.Pod) {
		err := podInformer.GetIndexer().Add(pod)
		Expect(err).ToNot(HaveOccurred())
	}

	shouldExpectPDBDeletion := func(pdb *policyv1.PodDisruptionBudget) {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("delete", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())
			Expect(pdb.Namespace).To(Equal(update.GetNamespace()))
			Expect(pdb.Name).To(Equal(update.GetName()))
			return true, nil, nil
		})
	}

	shouldExpectPDBCreation := func(uid types.UID) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			pdb := update.GetObject().(*policyv1.PodDisruptionBudget)
			Expect(ok).To(BeTrue())
			Expect(pdb.Spec.MinAvailable.String()).To(Equal("1"))
			Expect(update.GetObject().(*policyv1.PodDisruptionBudget).Spec.Selector.MatchLabels[v1.CreatedByLabel]).To(Equal(string(uid)))
			return true, update.GetObject(), nil
		})
	}

	shouldExpectPDBPatch := func(vmi *v1.VirtualMachineInstance) {
		kubeClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patchAction, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			Expect(patchAction.GetName()).To(Equal("pdb-" + vmi.Name))
			Expect(patchAction.GetPatchType()).To(Equal(types.JSONPatchType))
			patches := patch.New(
				patch.WithReplace("/spec/minAvailable", 1),
				patch.WithRemove(fmt.Sprintf("/metadata/labels/%s", patch.EscapeJSONPointer(
					v1.MigrationNameLabel))),
			)
			Expect(patches.GeneratePayload()).To(Equal(patchAction.GetPatch()))
			return true, &policyv1.PodDisruptionBudget{}, nil
		})
	}

	initController := func(kvConfig *v1.KubeVirtConfiguration) {
		config, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(kvConfig)

		controller, _ = NewDisruptionBudgetController(vmiInformer, pdbInformer, podInformer, vmimInformer, recorder, virtClient, config)
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		pdbFeeder = testutils.NewPodDisruptionBudgetFeeder(mockQueue, pdbSource)
		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		pdbInformer, pdbSource = testutils.NewFakeInformerFor(&policyv1.PodDisruptionBudget{})
		vmimInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceMigration{})
		podInformer, _ = testutils.NewFakeInformerFor(&corev1.Pod{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		initController(&v1.KubeVirtConfiguration{})

		// Set up mock client
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().PolicyV1().Return(kubeClient.PolicyV1()).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		syncCaches(stop)

	})

	sanityExecute := func() {
		controllertesting.SanityExecute(controller, []cache.Store{
			controller.vmiStore, controller.pdbIndexer, controller.podIndexer, controller.migrationIndexer,
		}, Default)
	}

	Context("A VirtualMachineInstance given which does not want to live-migrate on evictions", func() {

		It("should do nothing, if no pdb exists", func() {
			addVirtualMachine(nonMigratableVirtualMachine())
			sanityExecute()
		})

		It("should remove the pdb, if it is added to the cache", func() {
			vmi := nonMigratableVirtualMachine()
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		})
	})

	Context("With cluster that has 'EvictionStrategy' set to 'LiveMigrate'", func() {
		BeforeEach(func() {
			evictionStrategy := v1.EvictionStrategyLiveMigrate
			initController(&v1.KubeVirtConfiguration{EvictionStrategy: &evictionStrategy})
		})

		It("should add the pdb, if it does not exist", func() {
			vmi := nonMigratableVirtualMachine()
			addVirtualMachine(vmi)

			shouldExpectPDBCreation(vmi.UID)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodDisruptionBudgetReason)
		})
	})

	Context("A VirtualMachineInstance given which wants perform action on evictions", func() {

		DescribeTable("should do nothing, if a pdb exists", func(evictionStrategy v1.EvictionStrategy, vmi *v1.VirtualMachineInstance) {
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			sanityExecute()
		},
			Entry("with LiveMigrate eviction strategy and non-migratable VMI", v1.EvictionStrategyLiveMigrate, nonMigratableVirtualMachine()),
			Entry("with External eviction strategy and non-migratable VMI", v1.EvictionStrategyExternal, nonMigratableVirtualMachine()),
			Entry("with LiveMigrate eviction strategy and migratable VMI", v1.EvictionStrategyLiveMigrate, migratableVirtualMachine()),
			Entry("with LiveMigrateIfPossible eviction strategy and migratable VMI", v1.EvictionStrategyLiveMigrateIfPossible, migratableVirtualMachine()),
			Entry("with External eviction strategy and migratable VMI", v1.EvictionStrategyExternal, migratableVirtualMachine()),
		)

		DescribeTable("should remove the pdb if the VMI disappears", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			sanityExecute()

			vmiFeeder.Delete(vmi)
			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		DescribeTable("should remove the pdb if VMI doesn't exist", func(evictionStrategy v1.EvictionStrategy) {
			vmi := nonMigratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		DescribeTable("should recreate the PDB if the VMI is recreated", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			sanityExecute()

			vmiFeeder.Delete(vmi)
			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)

			pdbFeeder.Delete(pdb)
			vmi.UID = "45356"
			vmiFeeder.Add(vmi)
			shouldExpectPDBCreation(vmi.UID)
			sanityExecute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		DescribeTable("should delete a PDB which belongs to an old VMI", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)
			// new UID means new VMI
			vmi.UID = "changed"
			addVirtualMachine(vmi)

			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		DescribeTable("should not create a PDB for VMIs which are already marked for deletion", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			now := metav1.Now()
			vmi.DeletionTimestamp = &now
			addVirtualMachine(vmi)

			sanityExecute()

			vmiFeeder.Delete(vmi)
			sanityExecute()
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		It("should remove the pdb if the VMI does not want to be migrated anymore", func() {
			vmi := nonMigratableVirtualMachine()
			vmi.Spec.EvictionStrategy = newEvictionStrategyLiveMigrate()
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			sanityExecute()

			vmi.Spec.EvictionStrategy = nil
			vmiFeeder.Modify(vmi)
			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		})

		DescribeTable("should remove the pdb if the VMI reached Final state", func(vmiPhase v1.VirtualMachineInstancePhase) {
			vmi := nonMigratableVirtualMachine()
			vmi.Spec.EvictionStrategy = newEvictionStrategyLiveMigrate()

			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			vmi.Status.Phase = vmiPhase
			vmiFeeder.Add(vmi)

			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		},
			Entry("with Succeeded vmi phase", v1.Succeeded),
			Entry("with Failed vmi phase", v1.Failed),
		)

		DescribeTable("should add the pdb, if it does not exist", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)

			shouldExpectPDBCreation(vmi.UID)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		DescribeTable("should recreate the pdb, if it disappears", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)
			sanityExecute()

			shouldExpectPDBCreation(vmi.UID)
			pdbFeeder.Delete(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		DescribeTable("should recreate the pdb, if the pdb is orphaned", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)
			sanityExecute()

			shouldExpectPDBCreation(vmi.UID)
			newPdb := pdb.DeepCopy()
			newPdb.OwnerReferences = nil
			pdbFeeder.Modify(newPdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)

		It("should shrink the PDB after migration has completed", func() {
			vmi := nonMigratableVirtualMachine()
			vmi.Spec.EvictionStrategy = newEvictionStrategyLiveMigrate()
			vmim := newMigration("testmigration", vmi, v1.MigrationSucceeded)
			pod := newVMIPod(vmi, corev1.PodRunning)

			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 2)
			pdb.ObjectMeta.Labels = map[string]string{
				v1.MigrationNameLabel: vmim.Name,
			}
			pdbFeeder.Add(pdb)
			addMigration(vmim)
			addPod(pod)

			shouldExpectPDBPatch(vmi)

			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulUpdatePodDisruptionBudgetReason)
		})

		It("should shrink the PDB after migration object is gone", func() {
			vmi := nonMigratableVirtualMachine()
			vmi.Spec.EvictionStrategy = newEvictionStrategyLiveMigrate()

			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 2)
			pdb.ObjectMeta.Labels = map[string]string{
				v1.MigrationNameLabel: "testmigration",
			}
			pdbFeeder.Add(pdb)

			shouldExpectPDBPatch(vmi)

			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulUpdatePodDisruptionBudgetReason)
		})

		It("should not shrink the PDB while migration is running", func() {
			vmi := nonMigratableVirtualMachine()
			vmi.Spec.EvictionStrategy = newEvictionStrategyLiveMigrate()
			vmim := newMigration("testmigration", vmi, v1.MigrationRunning)

			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 2)
			pdb.ObjectMeta.Labels = map[string]string{
				v1.MigrationNameLabel: vmim.Name,
			}
			pdbFeeder.Add(pdb)
			addMigration(vmim)

			sanityExecute()
		})

		DescribeTable("should delete a PDB created by an old migration-controller", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy

			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 2)
			pdb.Name = "kubevirt-migration-pdb-" + vmi.Name
			pdb.ObjectMeta.Labels = map[string]string{
				v1.MigrationNameLabel: "testmigration",
			}
			pdbFeeder.Add(pdb)

			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy", v1.EvictionStrategyLiveMigrate),
			Entry("with LiveMigrateIfPossible eviction strategy", v1.EvictionStrategyLiveMigrateIfPossible),
			Entry("with External eviction strategy", v1.EvictionStrategyExternal),
		)
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
	})
})

func newVMIPod(vmi *v1.VirtualMachineInstance, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vmi.Namespace,
			Name:      vmi.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
			},
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
}

func newMigration(name string, vmi *v1.VirtualMachineInstance, phase v1.VirtualMachineInstanceMigrationPhase) *v1.VirtualMachineInstanceMigration {
	migration := &v1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: vmi.Namespace,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmi.Name,
		},
	}
	migration.Status.Phase = phase
	return migration
}

func nonMigratableVirtualMachine() *v1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI("testvm")
	vmi.Namespace = corev1.NamespaceDefault
	vmi.UID = "1234"
	return vmi
}

func migratableVirtualMachine() *v1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI("testvm")
	vmi.Namespace = corev1.NamespaceDefault
	vmi.UID = "1234"

	vmi.Status = v1.VirtualMachineInstanceStatus{
		Conditions: []v1.VirtualMachineInstanceCondition{
			{
				Type:   v1.VirtualMachineInstanceIsMigratable,
				Status: corev1.ConditionTrue,
			},
		},
	}
	return vmi
}

func newPodDisruptionBudget(vmi *v1.VirtualMachineInstance, pods int) *policyv1.PodDisruptionBudget {
	minAvailable := intstr.FromInt(pods)
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
			},
			Name:      "pdb-" + vmi.Name,
			Namespace: vmi.Namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					v1.CreatedByLabel: string(vmi.UID),
				},
			},
		},
	}
}

func newEvictionStrategyLiveMigrate() *v1.EvictionStrategy {
	strategy := v1.EvictionStrategyLiveMigrate
	return &strategy
}
