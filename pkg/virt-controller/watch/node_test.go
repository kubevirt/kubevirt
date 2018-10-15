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
	"github.com/pborman/uuid"
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
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var nodeSource *framework.FakeControllerSource
	var nodeInformer cache.SharedIndexInformer
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *NodeController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset
	var vmiFeeder *testutils.VirtualMachineFeeder

	syncCaches := func(stop chan struct{}) {
		go nodeInformer.Run(stop)
		go vmiInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, nodeInformer.HasSynced, vmiInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		nodeInformer, nodeSource = testutils.NewFakeInformerFor(&k8sv1.Node{})
		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		recorder = record.NewFakeRecorder(100)

		controller = NewNodeController(virtClient, nodeInformer, vmiInformer, recorder)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(v1.NamespaceAll).Return(vmiInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(v1.NamespaceDefault).Return(vmiInterface).AnyTimes()
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

	modifyNode := func(node *k8sv1.Node) {
		mockQueue.ExpectAdds(1)
		nodeSource.Modify(node)
		mockQueue.Wait()
	}

	deleteNode := func(node *k8sv1.Node) {
		mockQueue.ExpectAdds(1)
		nodeSource.Delete(node)
		mockQueue.Wait()
	}

	Context("pods and vmis given", func() {
		It("should only select stuck vmis", func() {
			node := NewHealthyNode("test")
			finalVMI := NewRunningVirtualMachine("finalVMI", node)
			finalVMI.Status.Phase = virtv1.Succeeded
			vmiWithPod := NewRunningVirtualMachine("vmiWithPod", node)
			podForVMI := NewHealthyPodForVirtualMachine("podForVMI", vmiWithPod)
			vmiWithPodInDifferentNamespace := NewRunningVirtualMachine("vmiWithPodInDifferentNamespace", node)
			podInDifferentNamespace := NewHealthyPodForVirtualMachine("podInDifferentnamespace", vmiWithPodInDifferentNamespace)
			podInDifferentNamespace.Namespace = "wrong"
			vmiWithoutPod := NewRunningVirtualMachine("vmiWithoutPod", node)

			vmis := filterStuckVirtualMachinesWithoutPods([]*virtv1.VirtualMachineInstance{
				finalVMI,
				vmiWithPod,
				vmiWithPodInDifferentNamespace,
				vmiWithoutPod,
			}, []*k8sv1.Pod{
				podForVMI,
				podInDifferentNamespace,
			})

			By("filtering out vmis in final state")
			Expect(vmis).ToNot(ContainElement(finalVMI))

			By("filtering out vmis which have a pod")
			Expect(vmis).ToNot(ContainElement(vmiWithPod))

			By("keeping vmis which are running and have no pod in their namespace")
			Expect(vmis).To(ContainElement(vmiWithoutPod))
			Expect(vmis).To(ContainElement(vmiWithPodInDifferentNamespace))
		})
	})

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
				patch, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				Expect(string(patch.GetPatch())).To(Equal(`{"metadata": { "labels": {"kubevirt.io/schedulable": "false"}}}`))
				return true, nil, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{}, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		table.DescribeTable("should set a vmi without a pod to failed state if the vmi is in ", func(phase virtv1.VirtualMachineInstancePhase) {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi1", node)
			vmi.Status.Phase = phase

			addNode(node)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		},
			table.Entry("running state", virtv1.Running),
			table.Entry("scheduled state", virtv1.Scheduled),
		)
		It("should set multiple vmis to failed in one go, even if some updates fail", func() {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi", node)
			vmi1 := NewRunningVirtualMachine("vmi1", node)
			vmi2 := NewRunningVirtualMachine("vmi2", node)

			addNode(node)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi, *vmi1, *vmi2}}, nil)
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any()).Times(1)
			vmiInterface.EXPECT().Patch(vmi1.Name, types.JSONPatchType, gomock.Any()).Return(nil, fmt.Errorf("some error")).Times(1)
			vmiInterface.EXPECT().Patch(vmi2.Name, types.JSONPatchType, gomock.Any()).Times(1)

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		It("should set a vmi without a pod to failed state, triggered by vmi add event", func() {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi1", node)

			vmiFeeder.Add(vmi)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		It("should set a vmi without a pod to failed state, triggered by node update", func() {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi1", node)

			nodeInformer.GetStore().Add(node)
			modifyNode(node.DeepCopy())
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		It("should set a vmi without a pod to failed state, triggered by node delete", func() {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi1", node)

			nodeInformer.GetStore().Add(node)
			deleteNode(node.DeepCopy())
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		It("should set a vmi without a pod to failed state, triggered by vmi modify event", func() {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi1", node)

			vmiInformer.GetStore().Add(vmi)
			vmiFeeder.Modify(vmi)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		It("should set a vmi without a pod to failed state, triggered by vmi modify event", func() {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi1", node)

			vmiInformer.GetStore().Add(vmi)
			vmiFeeder.Modify(vmi)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewUnhealthyPodForVirtualMachine("whatever", vmi)}}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		table.DescribeTable("should ignore a vmi without a pod if the vmi is in ", func(phase virtv1.VirtualMachineInstancePhase) {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi1", node)
			vmi.Status.Phase = phase

			addNode(node)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)

			controller.Execute()
		},
			table.Entry("unprocessed state", virtv1.VmPhaseUnset),
			table.Entry("pending state", virtv1.Pending),
			table.Entry("scheduling state", virtv1.Scheduling),
			table.Entry("failed state", virtv1.Failed),
			table.Entry("failed state", virtv1.Succeeded),
		)
		table.DescribeTable("should ignore a vmi which still has a healthy pod in", func(phase virtv1.VirtualMachineInstancePhase) {
			node := NewUnhealthyNode("testnode")
			vmi := NewRunningVirtualMachine("vmi", node)
			vmi.Status.Phase = phase
			vmi1 := NewRunningVirtualMachine("vmi1", node)
			vmi1.Status.Phase = phase

			addNode(node)
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewHealthyPodForVirtualMachine("whatever", vmi1)}}, nil
			})

			vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{*vmi}}, nil)
			By("checking that only a vmi with a pod gets removed")
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any())

			controller.Execute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
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

func NewRunningVirtualMachine(vmiName string, node *k8sv1.Node) *virtv1.VirtualMachineInstance {
	vmi := virtv1.NewMinimalVMI(vmiName)
	vmi.UID = types.UID(uuid.NewRandom().String())
	vmi.Status.Phase = virtv1.Running
	vmi.Status.NodeName = node.Name
	vmi.Labels = map[string]string{
		virtv1.NodeNameLabel: node.Name,
	}
	return vmi
}

func NewUnhealthyPodForVirtualMachine(podName string, vmi *virtv1.VirtualMachineInstance) *k8sv1.Pod {
	pod := NewHealthyPodForVirtualMachine(podName, vmi)
	pod.Status.Phase = k8sv1.PodFailed
	return pod
}

func NewHealthyPodForVirtualMachine(podName string, vmi *virtv1.VirtualMachineInstance) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: k8sv1.NamespaceDefault,
			Labels: map[string]string{
				virtv1.CreatedByLabel: string(vmi.UID),
			},
		},
		Spec: k8sv1.PodSpec{NodeName: vmi.Status.NodeName},
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
		},
	}
}
