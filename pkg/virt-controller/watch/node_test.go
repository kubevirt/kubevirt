package watch

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Node controller with", func() {
	log.Log.SetIOWriter(GinkgoWriter)

	var ctrl *gomock.Controller
	var vmInterface *kubecli.MockVMInterface
	var nodeSource *framework.FakeControllerSource
	var nodeInformer cache.SharedIndexInformer
	var vmSource *framework.FakeControllerSource
	var vmInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *NodeController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset
	var vmFeeder *testutils.VirtualMachineFeeder

	syncCaches := func(stop chan struct{}) {
		go nodeInformer.Run(stop)
		go vmInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, nodeInformer.HasSynced, vmInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVMInterface(ctrl)

		nodeInformer, nodeSource = testutils.NewFakeInformerFor(&k8sv1.Node{})
		vmInformer, vmSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		recorder = record.NewFakeRecorder(100)

		controller = NewNodeController(virtClient, nodeInformer, vmInformer, recorder)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)

		// Set up mock client
		virtClient.EXPECT().VM(v1.NamespaceAll).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VM(v1.NamespaceDefault).Return(vmInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		syncCaches(stop)
	})

	addNode := func(node *k8sv1.Node) {
		mockQueue.ExpectAdds(1)
		nodeSource.Add(node)
		mockQueue.Wait()
	}

	Context("responsive virt-handler given", func() {
		It("should do nothing", func() {
			node := NewHealthyNode("testnode")

			addNode(node)

			controller.Execute()
		})
	})

	Context("unresponsive virt-handler given", func() {
		It("should set the node to unschedulable", func() {
			node := NewHealthyNode("testnode")
			node.Annotations[virtv1.VirtHandlerHeartbeat] = nowAsJSONWithOffset(-10 * time.Minute)

			addNode(node)

			kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, nil
			})

			vmInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineList{}, nil)

			controller.Execute()
		})
		table.DescribeTable("should set a vm without a pod to failed state if the vm is in ", func(phase virtv1.VMPhase) {
			node := NewUnhealthyNode("testnode")
			vm := NewRunningVirtualMachine("vm1", node)
			vm.Status.Phase = phase

			addNode(node)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineList{Items: []virtv1.VirtualMachine{*vm}}, nil)
			vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
		},
			table.Entry("running state", virtv1.Running),
			table.Entry("scheduled state", virtv1.Scheduled),
		)
		It("should set a vm without a pod to failed state, triggered by vm add event", func() {
			node := NewUnhealthyNode("testnode")
			vm := NewRunningVirtualMachine("vm1", node)

			vmFeeder.Add(vm)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineList{Items: []virtv1.VirtualMachine{*vm}}, nil)
			vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
		})
		It("should set a vm without a pod to failed state, triggered by vm modify event", func() {
			node := NewUnhealthyNode("testnode")
			vm := NewRunningVirtualMachine("vm1", node)

			vmInformer.GetStore().Add(vm)
			vmFeeder.Modify(vm)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineList{Items: []virtv1.VirtualMachine{*vm}}, nil)
			vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
		})
		table.DescribeTable("should ignore a vm without a pod if the vm is in ", func(phase virtv1.VMPhase) {
			node := NewUnhealthyNode("testnode")
			vm := NewRunningVirtualMachine("vm1", node)
			vm.Status.Phase = phase

			addNode(node)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineList{Items: []virtv1.VirtualMachine{*vm}}, nil)

			controller.Execute()
		},
			table.Entry("unprocessed state", virtv1.VmPhaseUnset),
			table.Entry("pending state", virtv1.Pending),
			table.Entry("scheduling state", virtv1.Scheduling),
			table.Entry("failed state", virtv1.Failed),
			table.Entry("failed state", virtv1.Succeeded),
		)
		table.DescribeTable("should ignore a vm which still has a pod in", func(phase virtv1.VMPhase) {
			node := NewUnhealthyNode("testnode")
			vm := NewRunningVirtualMachine("vm", node)
			vm.Status.Phase = phase
			vm1 := NewRunningVirtualMachine("vm1", node)
			vm1.Status.Phase = phase

			addNode(node)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{{
					ObjectMeta: v1.ObjectMeta{
						Name:      "whatever",
						Namespace: k8sv1.NamespaceDefault,
						Labels: map[string]string{
							virtv1.DomainLabel: vm1.Name,
						},
					},
				}}}, nil
			})

			vmInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineList{Items: []virtv1.VirtualMachine{*vm}}, nil)
			By("checking that only a vm with a pod gets removed")
			vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
		},
			table.Entry("running state", virtv1.Running),
			table.Entry("scheduled state", virtv1.Scheduled),
		)
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})

})

func NewHealthyNode(nodeName string) *k8sv1.Node {
	return &k8sv1.Node{
		ObjectMeta: v1.ObjectMeta{
			Name: nodeName,
			Annotations: map[string]string{
				virtv1.VirtHandlerHeartbeat: nowAsJSONWithOffset(0),
			},
			Labels: map[string]string{
				virtv1.NodeSchedulable: "true",
			},
		},
	}
}

func NewUnhealthyNode(nodeName string) *k8sv1.Node {
	node := NewHealthyNode(nodeName)
	node.Annotations[virtv1.VirtHandlerHeartbeat] = nowAsJSONWithOffset(-10 * time.Minute)
	node.Labels[virtv1.NodeSchedulable] = "false"
	return node
}

func nowAsJSONWithOffset(offset time.Duration) string {
	now := v1.Now()
	now = v1.NewTime(now.Add(offset))

	data, err := json.Marshal(now)
	Expect(err).ToNot(HaveOccurred())
	return strings.Trim(string(data), `"`)
}

func NewRunningVirtualMachine(vmName string, node *k8sv1.Node) *virtv1.VirtualMachine {
	vm := virtv1.NewMinimalVM(vmName)
	vm.UID = "1234"
	vm.Status.Phase = virtv1.Running
	vm.Status.NodeName = node.Name
	addInitializedAnnotation(vm)
	vm.Labels = map[string]string{
		virtv1.NodeNameLabel: node.Name,
	}
	return vm
}
