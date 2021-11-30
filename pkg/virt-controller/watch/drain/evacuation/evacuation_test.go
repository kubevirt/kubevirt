package evacuation_test

import (
	"time"

	"github.com/golang/mock/gomock"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/evacuation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Evacuation", func() {
	var ctrl *gomock.Controller
	var stop chan struct{}
	var virtClient *kubecli.MockKubevirtClient
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var nodeSource *framework.FakeControllerSource
	var nodeInformer cache.SharedIndexInformer
	var migrationInformer cache.SharedIndexInformer
	var migrationSource *framework.FakeControllerSource
	var podInformer cache.SharedIndexInformer
	var podSource *framework.FakeControllerSource
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var kubeClient *fake.Clientset
	var migrationFeeder *testutils.MigrationFeeder
	var vmiFeeder *testutils.VirtualMachineFeeder

	var controller *evacuation.EvacuationController

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go migrationInformer.Run(stop)
		go nodeInformer.Run(stop)
		go podInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			migrationInformer.HasSynced,
			nodeInformer.HasSynced,
			podInformer.HasSynced,
		)).To(BeTrue())
	}

	addNode := func(node *v12.Node) {
		mockQueue.ExpectAdds(1)
		nodeSource.Add(node)
		mockQueue.Wait()
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)

		vmiInformer, vmiSource = testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
			"node": func(obj interface{}) (strings []string, e error) {
				return []string{obj.(*v1.VirtualMachineInstance).Status.NodeName}, nil
			},
		})
		migrationInformer, migrationSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceMigration{})
		nodeInformer, nodeSource = testutils.NewFakeInformerFor(&v12.Node{})
		podInformer, podSource = testutils.NewFakeInformerFor(&v12.Pod{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		controller = evacuation.NewEvacuationController(vmiInformer, migrationInformer, nodeInformer, podInformer, recorder, virtClient, config)
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		migrationFeeder = testutils.NewMigrationFeeder(mockQueue, migrationSource)
		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstanceMigration(v12.NamespaceDefault).Return(migrationInterface).AnyTimes()
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

	Context("migration object creation", func() {
		It("should have expected values and annotations", func() {
			migration := evacuation.GenerateNewMigration("my-vmi", "somenode")
			Expect(migration.Spec.VMIName).To(Equal("my-vmi"))
			Expect(migration.Annotations[v1.EvacuationMigrationAnnotation]).To(Equal("somenode"))
		})

	})

	Context("no node eviction in progress", func() {

		It("should do nothing with VMIs", func() {
			node := newNode("testnode")
			addNode(node)
			vmi := newVirtualMachine("testvm", node.Name)
			vmiFeeder.Add(vmi)

			controller.Execute()
		})

		It("should do nothing if the target node is not evicting", func() {
			node := newNode("testnode")
			node1 := newNode("anothernode")
			node.Spec.Taints = append(node.Spec.Taints, *newTaint())
			addNode(node)
			addNode(node1)
			vmi := newVirtualMachine("testvm", node1.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmiFeeder.Add(vmi)

			controller.Execute()
		})
	})

	Context("node eviction in progress", func() {

		It("should evict the VMI", func() {
			node := newNode("testnode")
			node1 := newNode("anothernode")
			node.Spec.Taints = append(node.Spec.Taints, *newTaint())
			addNode(node)
			addNode(node1)

			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmiFeeder.Add(vmi)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, evacuation.SuccessfulCreateVirtualMachineInstanceMigrationReason)
		})

		It("should ignore VMIs which are not migratable", func() {
			node := newNode("testnode")
			node1 := newNode("anothernode")
			node.Spec.Taints = append(node.Spec.Taints, *newTaint())
			addNode(node)
			addNode(node1)

			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{Type: v1.VirtualMachineInstanceIsMigratable, Status: v12.ConditionFalse}}
			vmiFeeder.Add(vmi)

			vmi1 := newVirtualMachine("testvm1", node.Name)
			vmi1.Spec.EvictionStrategy = newEvictionStrategy()
			vmi1.Status.Conditions = nil
			vmiFeeder.Add(vmi1)

			controller.Execute()
			testutils.ExpectEvents(recorder,
				evacuation.FailedCreateVirtualMachineInstanceMigrationReason,
				evacuation.FailedCreateVirtualMachineInstanceMigrationReason,
			)
		})

		It("should not evict VMIs if 5 migrations are in progress", func() {
			node := newNode("testnode")
			node.Spec.Taints = append(node.Spec.Taints, *newTaint())
			addNode(node)

			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi1 := newVirtualMachine("testvm1", node.Name)
			vmi1.Spec.EvictionStrategy = newEvictionStrategy()
			vmiFeeder.Add(vmi)
			vmiFeeder.Add(vmi1)

			migrationFeeder.Add(newMigration("mig1", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig2", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig3", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig4", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig5", vmi.Name, v1.MigrationRunning))

			controller.Execute()

		})

		It("should start another migration if one completes and we have a free spot", func() {
			node := newNode("testnode")
			node.Spec.Taints = append(node.Spec.Taints, *newTaint())
			addNode(node)

			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi1 := newVirtualMachine("testvm1", node.Name)
			vmi1.Spec.EvictionStrategy = newEvictionStrategy()
			vmiFeeder.Add(vmi)
			vmiFeeder.Add(vmi1)
			migration := newMigration("mig1", vmi.Name, v1.MigrationRunning)

			migrationFeeder.Add(migration)
			migrationFeeder.Add(newMigration("mig2", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig3", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig4", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig5", vmi.Name, v1.MigrationRunning))

			controller.Execute()

			migration.Status.Phase = v1.MigrationSucceeded
			migrationFeeder.Modify(migration)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, evacuation.SuccessfulCreateVirtualMachineInstanceMigrationReason)
		})
	})

	Context("VMIs marked for eviction", func() {

		It("Should evict the VMI", func() {
			node := newNode("foo")
			addNode(node)
			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi.Status.EvacuationNodeName = node.Name
			vmiFeeder.Add(vmi)
			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, evacuation.SuccessfulCreateVirtualMachineInstanceMigrationReason)
		})

		It("Should record a warning on a not migratable VMI", func() {
			node := newNode("foo")
			addNode(node)
			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: v12.ConditionFalse,
				},
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: v12.ConditionFalse,
				},
			}
			vmi.Status.EvacuationNodeName = vmi.Status.NodeName
			vmiFeeder.Add(vmi)
			controller.Execute()
			testutils.ExpectEvent(recorder, evacuation.FailedCreateVirtualMachineInstanceMigrationReason)
		})

		It("Should not evict VMI if max migrations are in progress", func() {
			node := newNode("foo")
			addNode(node)
			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: v12.ConditionFalse,
				},
			}
			vmi.Status.EvacuationNodeName = node.Name
			vmiFeeder.Add(vmi)
			migrationFeeder.Add(newMigration("mig1", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig2", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig3", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig4", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig5", vmi.Name, v1.MigrationRunning))
			controller.Execute()
		})

		It("Should start a migration when we have a free spot", func() {
			node := newNode("foo")
			addNode(node)
			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: v12.ConditionTrue,
				},
			}
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi.Status.EvacuationNodeName = node.Name
			vmiFeeder.Add(vmi)

			vmi1 := newVirtualMachine("testvm1", node.Name)
			vmi1.Spec.EvictionStrategy = newEvictionStrategy()
			vmi1.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: v12.ConditionTrue,
				},
			}
			vmi1.Status.EvacuationNodeName = node.Name
			vmiFeeder.Add(vmi1)

			migration := newMigration("mig1", vmi.Name, v1.MigrationRunning)
			migrationFeeder.Add(migration)
			migrationFeeder.Add(newMigration("mig2", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig3", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig4", vmi.Name, v1.MigrationRunning))
			migrationFeeder.Add(newMigration("mig5", vmi.Name, v1.MigrationRunning))

			controller.Execute()

			migration.Status.Phase = v1.MigrationSucceeded
			migrationFeeder.Modify(migration)

			migrationInterface.EXPECT().Create(gomock.Any()).
				Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, evacuation.SuccessfulCreateVirtualMachineInstanceMigrationReason)
		})

		It("Should not create a migration if one is already in progress", func() {
			node := newNode("foo")
			addNode(node)
			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: v12.ConditionTrue,
				},
			}
			vmi.Spec.EvictionStrategy = newEvictionStrategy()
			vmi.Status.EvacuationNodeName = node.Name
			vmiFeeder.Add(vmi)

			migration := newMigration("mig1", vmi.Name, v1.MigrationRunning)
			migration.Status.Phase = v1.MigrationRunning

			migrationFeeder.Add(migration)

			controller.Execute()
		})

		It("should evict the VMI if only one pod is running", func() {
			node := newNode("testnode")
			node1 := newNode("anothernode")
			node.Spec.Taints = append(node.Spec.Taints, *newTaint())
			addNode(node)
			addNode(node1)

			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()

			podSource.Add(newPod(vmi, "runningPod", v12.PodRunning, true))
			podSource.Add(newPod(vmi, "succededPod", v12.PodSucceeded, true))
			podSource.Add(newPod(vmi, "failedPod", v12.PodFailed, true))
			podSource.Add(newPod(vmi, "notOwnedRunningPod", v12.PodRunning, false))
			// pods do not cause the queue to get added to
			// we just use them for caching purposes
			// so wait for cache to catch up with a brief sleep
			time.Sleep(1 * time.Second)

			vmiFeeder.Add(vmi)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, evacuation.SuccessfulCreateVirtualMachineInstanceMigrationReason)
		})

		It("should not evict the VMI with multiple pods active", func() {
			node := newNode("testnode")
			node1 := newNode("anothernode")
			node.Spec.Taints = append(node.Spec.Taints, *newTaint())
			addNode(node)
			addNode(node1)

			vmi := newVirtualMachine("testvm", node.Name)
			vmi.Spec.EvictionStrategy = newEvictionStrategy()

			podSource.Add(newPod(vmi, "runningPod", v12.PodRunning, true))
			podSource.Add(newPod(vmi, "pendingPod", v12.PodPending, true))
			// pods do not cause the queue to get added to
			// we just use them for caching purposes
			// so wait for cache to catch up with a brief sleep
			time.Sleep(1 * time.Second)

			vmiFeeder.Add(vmi)

			controller.Execute()
		})
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})
})

func newNode(name string) *v12.Node {
	return &v12.Node{
		ObjectMeta: v13.ObjectMeta{
			Name: name,
		},
		Spec: v12.NodeSpec{},
	}
}

func newVirtualMachine(name string, nodeName string) *v1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI("testvm")
	vmi.Name = name
	vmi.Status.NodeName = nodeName
	vmi.Namespace = v12.NamespaceDefault
	vmi.UID = "1234"
	vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{Type: v1.VirtualMachineInstanceIsMigratable, Status: v12.ConditionTrue}}
	return vmi
}

func newPod(vmi *v1.VirtualMachineInstance, name string, phase v12.PodPhase, ownedByVMI bool) *v12.Pod {
	pod := &v12.Pod{
		ObjectMeta: v13.ObjectMeta{
			Name:      name,
			Namespace: vmi.Namespace,
		},
		Status: v12.PodStatus{
			Phase: phase,
			ContainerStatuses: []v12.ContainerStatus{
				{Ready: false, Name: "compute", State: v12.ContainerState{Running: &v12.ContainerStateRunning{}}},
			},
		},
	}

	if ownedByVMI {
		pod.Labels = map[string]string{
			v1.AppLabel:       "virt-launcher",
			v1.CreatedByLabel: string(vmi.UID),
		}
		pod.Annotations = map[string]string{
			v1.DomainAnnotation: vmi.Name,
		}
	}

	return pod
}

func newMigration(name string, vmi string, phase v1.VirtualMachineInstanceMigrationPhase) *v1.VirtualMachineInstanceMigration {
	migration := kubecli.NewMinimalMigration(name)
	migration.Status.Phase = phase
	migration.Spec.VMIName = vmi
	migration.Namespace = v12.NamespaceDefault
	return migration
}

func newEvictionStrategy() *v1.EvictionStrategy {
	strategy := v1.EvictionStrategyLiveMigrate
	return &strategy
}

func newTaint() *v12.Taint {
	return &v12.Taint{
		Effect: v12.TaintEffectNoSchedule,
		Key:    "kubevirt.io/drain",
	}
}
