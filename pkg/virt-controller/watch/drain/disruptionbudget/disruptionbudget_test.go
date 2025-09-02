package disruptionbudget

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/testutils"
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

	initController := func() {

		controller, _ = NewDisruptionBudgetController(vmiInformer, pdbInformer, podInformer, vmimInformer, recorder, virtClient)
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
		initController()

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
			controller.vmiStore, controller.pdbIndexer,
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

	Context("A VirtualMachineInstance given which wants perform action on evictions", func() {

		DescribeTable("should delete the pdb, if a pdb exists", func(evictionStrategy v1.EvictionStrategy, vmi *v1.VirtualMachineInstance) {
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

			shouldExpectPDBDeletion(pdb)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodDisruptionBudgetReason)
		},
			Entry("with LiveMigrate eviction strategy and non-migratable VMI", v1.EvictionStrategyLiveMigrate, nonMigratableVirtualMachine()),
			Entry("with LiveMigrate eviction strategy and migratable VMI", v1.EvictionStrategyLiveMigrate, migratableVirtualMachine()),
			Entry("with External eviction strategy and migratable VMI", v1.EvictionStrategyExternal, migratableVirtualMachine()),
			Entry("with External eviction strategy and non-migratable VMI", v1.EvictionStrategyExternal, nonMigratableVirtualMachine()),
			Entry("with LiveMigrateIfPossible eviction strategy and migratable VMI", v1.EvictionStrategyLiveMigrateIfPossible, migratableVirtualMachine()),
		)

		DescribeTable("should remove the pdb if the VMI disappears", func(evictionStrategy v1.EvictionStrategy) {
			vmi := migratableVirtualMachine()
			vmi.Spec.EvictionStrategy = &evictionStrategy
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi, 1)
			pdbFeeder.Add(pdb)

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
