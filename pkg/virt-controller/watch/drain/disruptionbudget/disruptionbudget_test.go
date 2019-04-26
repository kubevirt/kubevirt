package disruptionbudget_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/disruptionbudget"
)

var _ = Describe("Disruptionbudget", func() {

	var ctrl *gomock.Controller
	var stop chan struct{}
	var virtClient *kubecli.MockKubevirtClient
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var pdbInformer cache.SharedIndexInformer
	var pdbSource *framework.FakeControllerSource
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var kubeClient *fake.Clientset
	var pdbFeeder *testutils.PodDisruptionBudgetFeeder
	var vmiFeeder *testutils.VirtualMachineFeeder

	var controller *disruptionbudget.DisruptionBudgetController

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go pdbInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			pdbInformer.HasSynced,
		)).To(BeTrue())
	}

	addVirtualMachine := func(vmi *v1.VirtualMachineInstance) {
		mockQueue.ExpectAdds(1)
		vmiSource.Add(vmi)
		mockQueue.Wait()
	}

	shouldExpectPDBDeletion := func(pdb *v1beta1.PodDisruptionBudget) {
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
			pdb := update.GetObject().(*v1beta1.PodDisruptionBudget)
			Expect(ok).To(BeTrue())
			Expect(pdb.Spec.MinAvailable.String()).To(Equal("2"))
			Expect(update.GetObject().(*v1beta1.PodDisruptionBudget).Spec.Selector.MatchLabels[v1.CreatedByLabel]).To(Equal(string(uid)))
			return true, update.GetObject(), nil
		})
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		pdbInformer, pdbSource = testutils.NewFakeInformerFor(&v1beta1.PodDisruptionBudget{})
		recorder = record.NewFakeRecorder(100)

		controller = disruptionbudget.NewDisruptionBudgetController(vmiInformer, pdbInformer, recorder, virtClient)
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		pdbFeeder = testutils.NewPodDisruptionBudgetFeeder(mockQueue, pdbSource)
		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(v12.NamespaceDefault).Return(vmiInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().PolicyV1beta1().Return(kubeClient.PolicyV1beta1()).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		syncCaches(stop)

	})

	Context("A VirtualMachineInstance given which does not want to live-migrate on evictions", func() {

		It("should do nothing, if no pdb exists", func() {
			addVirtualMachine(newVirtualMachine("testvm"))
			controller.Execute()
		})

		It("should remove the pdb, if it is added to the cache", func() {
			vmi := newVirtualMachine("testvm")
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi)
			pdbFeeder.Add(pdb)

			shouldExpectPDBDeletion(pdb)
			controller.Execute()
			testutils.ExpectEvent(recorder, disruptionbudget.SuccessfulDeletePodDisruptionBudgetReason)
		})
	})

	Context("A VirtualMachineInstance given which wants to live-migrate on evictions", func() {

		It("should do nothing, if a pdb exists", func() {
			vmi := newVirtualMachine("testvm")
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi)
			pdbFeeder.Add(pdb)

			controller.Execute()
		})

		It("should remove the pdb if the VMI disappears", func() {
			vmi := newVirtualMachine("testvm")
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi)
			pdbFeeder.Add(pdb)

			controller.Execute()

			vmiFeeder.Delete(vmi)
			shouldExpectPDBDeletion(pdb)
			controller.Execute()
			testutils.ExpectEvent(recorder, disruptionbudget.SuccessfulDeletePodDisruptionBudgetReason)
		})

		It("should remove the pdb if the VMI does not want to be migrated anymore", func() {
			vmi := newVirtualMachine("testvm")
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi)
			pdbFeeder.Add(pdb)

			controller.Execute()

			vmi.Spec.EvictionStrategy = nil
			vmiFeeder.Modify(vmi)
			shouldExpectPDBDeletion(pdb)
			controller.Execute()
			testutils.ExpectEvent(recorder, disruptionbudget.SuccessfulDeletePodDisruptionBudgetReason)
		})

		It("should add the pdb, if it does not exist", func() {
			vmi := newVirtualMachine("testvm")
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			addVirtualMachine(vmi)

			shouldExpectPDBCreation(vmi.UID)
			controller.Execute()
			testutils.ExpectEvent(recorder, disruptionbudget.SuccessfulCreatePodDisruptionBudgetReason)
		})

		It("should recreate the pdb, if it disappears", func() {
			vmi := newVirtualMachine("testvm")
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi)
			pdbFeeder.Add(pdb)
			controller.Execute()

			shouldExpectPDBCreation(vmi.UID)
			pdbFeeder.Delete(pdb)
			controller.Execute()
			testutils.ExpectEvent(recorder, disruptionbudget.SuccessfulCreatePodDisruptionBudgetReason)
		})

		It("should recreate the pdb, if the pdb is orphaned", func() {
			vmi := newVirtualMachine("testvm")
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			addVirtualMachine(vmi)
			pdb := newPodDisruptionBudget(vmi)
			pdbFeeder.Add(pdb)
			controller.Execute()

			shouldExpectPDBCreation(vmi.UID)
			newPdb := pdb.DeepCopy()
			newPdb.OwnerReferences = nil
			pdbFeeder.Modify(newPdb)
			controller.Execute()
			testutils.ExpectEvent(recorder, disruptionbudget.SuccessfulCreatePodDisruptionBudgetReason)
		})
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})
})

func newVirtualMachine(name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMI("testvm")
	vmi.Namespace = v12.NamespaceDefault
	vmi.UID = "1234"
	return vmi
}

func newPodDisruptionBudget(vmi *v1.VirtualMachineInstance) *v1beta1.PodDisruptionBudget {
	one := intstr.FromInt(1)
	return &v1beta1.PodDisruptionBudget{
		ObjectMeta: v13.ObjectMeta{
			OwnerReferences: []v13.OwnerReference{
				*v13.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
			},
			GenerateName: "kubevirt-disruption-budget-",
			Namespace:    vmi.Namespace,
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &one,
			Selector: &v13.LabelSelector{
				MatchLabels: map[string]string{
					v1.CreatedByLabel: string(vmi.UID),
				},
			},
		},
	}
}

func newEvictionStrategy() *v1.EvictionStrategy {
	strategy := v1.EvictionStrategyLiveMigrate
	return &strategy
}
