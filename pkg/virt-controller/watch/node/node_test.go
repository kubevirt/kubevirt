package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	appv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/testing"

	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/testutils"
	watchtesting "kubevirt.io/kubevirt/pkg/virt-controller/watch/testing"
)

var _ = Describe("Node controller with", func() {

	var ctrl *gomock.Controller
	var fakeVirtClient *kubevirtfake.Clientset
	var nodeSource *framework.FakeControllerSource
	var nodeInformer cache.SharedIndexInformer
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *Controller
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue[string]
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset
	var vmiFeeder *testutils.VirtualMachineFeeder[string]

	syncCaches := func(stop chan struct{}) {
		go nodeInformer.Run(stop)
		go vmiInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, nodeInformer.HasSynced, vmiInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		fakeVirtClient = kubevirtfake.NewSimpleClientset()
		kubeClient = fake.NewSimpleClientset()

		nodeInformer, nodeSource = testutils.NewFakeInformerFor(&k8sv1.Node{})
		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		controller, _ = NewController(virtClient, kubeClient, nodeInformer, vmiInformer, recorder)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceAll).Return(fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceAll)).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
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

	addVMI := func(vmi *v1.VirtualMachineInstance) {
		_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
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

	expectVMIToFailedStatus := func(vmiName string) {
		updatedVMI, err := fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmiName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI.Status.Phase).To(Equal(v1.Failed))
		Expect(updatedVMI.Status.Reason).To(Equal(NodeUnresponsiveReason))
	}

	sanityExecute := func() {
		controllertesting.SanityExecute(controller, []cache.Store{
			controller.vmiStore, controller.nodeStore,
		}, Default)
	}

	Context("pods and vmis given", func() {
		It("should only select stuck vmis", func() {
			node := NewHealthyNode("test")
			vmiWithPod := watchtesting.NewRunningVirtualMachine("vmiWithPod", node)
			podForVMI := NewHealthyPodForVirtualMachine("podForVMI", vmiWithPod)
			vmiWithPodInDifferentNamespace := watchtesting.NewRunningVirtualMachine("vmiWithPodInDifferentNamespace", node)
			podInDifferentNamespace := NewHealthyPodForVirtualMachine("podInDifferentnamespace", vmiWithPodInDifferentNamespace)
			podInDifferentNamespace.Namespace = "wrong"
			vmiWithoutPod := watchtesting.NewRunningVirtualMachine("vmiWithoutPod", node)

			vmis := filterStuckVirtualMachinesWithoutPods([]*v1.VirtualMachineInstance{
				vmiWithPod,
				vmiWithPodInDifferentNamespace,
				vmiWithoutPod,
			}, []*k8sv1.Pod{
				podForVMI,
				podInDifferentNamespace,
			})

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

			sanityExecute()
		})
	})

	Context("unresponsive virt-handler given", func() {
		It("should set the node to unschedulable", func() {
			node := NewHealthyNode("testnode")
			node.Annotations[v1.VirtHandlerHeartbeat] = nowAsJSONWithOffset(-10 * time.Minute)

			addNode(node)

			kubeClient.Fake.PrependReactor("patch", "nodes", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				patch, ok := action.(k8stesting.PatchAction)
				Expect(ok).To(BeTrue())
				Expect(string(patch.GetPatch())).To(Equal(`{"metadata": { "labels": {"kubevirt.io/schedulable": "false"}}}`))
				return true, nil, nil
			})

			sanityExecute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
		})
		DescribeTable("should set a vmi without a pod to failed state if the vmi is in ", func(phase v1.VirtualMachineInstancePhase) {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi1", node)
			vmi.Status.Phase = phase
			addVMI(vmi)
			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{}, nil
			})

			Expect(controller.checkVirtLauncherPodsAndUpdateVMIStatus(node.Name, []*v1.VirtualMachineInstance{vmi}, log.DefaultLogger())).To(Succeed())
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			expectVMIToFailedStatus(vmi.Name)
		},
			Entry("running state", v1.Running),
			Entry("scheduled state", v1.Scheduled),
		)
		It("should set multiple vmis to failed in one go, even if some updates fail", func() {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi", node)
			addVMI(vmi)
			vmi1 := watchtesting.NewRunningVirtualMachine("vmi1", node)
			addVMI(vmi1)
			vmi2 := watchtesting.NewRunningVirtualMachine("vmi2", node)
			addVMI(vmi2)

			fakeVirtClient.Fake.PrependReactor("patch", "virtualmachineinstances", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				patchAction := action.(k8stesting.PatchAction)
				if patchAction.GetName() == vmi1.Name {
					return true, &v1.VirtualMachineInstance{}, fmt.Errorf("some error")
				}
				return
			})

			Expect(controller.updateVMIWithFailedStatus([]*v1.VirtualMachineInstance{vmi, vmi1, vmi2}, log.DefaultLogger())).To(HaveOccurred())
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "patch", "virtualmachineinstances")).To(HaveLen(3))
		})
		It("should set a vmi without a pod to failed state, triggered by vmi add event", func() {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi1", node)
			addVMI(vmi)

			vmiFeeder.Add(vmi)
			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				a, _ := action.(k8stesting.ListAction)
				if strings.Contains(a.GetListRestrictions().Labels.String(), "virt-handler") {
					return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewVirtHandlerPod(node.Name)}}, nil
				}

				return true, &k8sv1.PodList{}, nil
			})

			sanityExecute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			expectVMIToFailedStatus(vmi.Name)
		})
		It("should set a vmi without a pod containing all terminated containers in a failed state", func() {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi1", node)
			addVMI(vmi)

			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewUnhealthyStuckTerminatingPodForVirtualMachine("whatever", vmi)}}, nil
			})

			Expect(controller.checkVirtLauncherPodsAndUpdateVMIStatus("testnode", []*v1.VirtualMachineInstance{vmi}, log.DefaultLogger())).To(Succeed())
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			expectVMIToFailedStatus(vmi.Name)
		})
		It("should set a vmi without a pod to failed state, triggered by node update", func() {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi1", node)
			addVMI(vmi)

			Expect(nodeInformer.GetStore().Add(node)).To(Succeed())
			modifyNode(node.DeepCopy())
			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				a, _ := action.(k8stesting.ListAction)
				if strings.Contains(a.GetListRestrictions().Labels.String(), "virt-handler") {
					return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewVirtHandlerPod(node.Name)}}, nil
				}

				return true, &k8sv1.PodList{}, nil
			})

			sanityExecute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			expectVMIToFailedStatus(vmi.Name)
		})
		It("should set a vmi without a pod to failed state, triggered by node delete", func() {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi1", node)
			addVMI(vmi)

			Expect(nodeInformer.GetStore().Add(node)).To(Succeed())
			deleteNode(node.DeepCopy())
			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				a, _ := action.(k8stesting.ListAction)
				if strings.Contains(a.GetListRestrictions().Labels.String(), "virt-handler") {
					return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewVirtHandlerPod(node.Name)}}, nil
				}

				return true, &k8sv1.PodList{}, nil
			})

			sanityExecute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			expectVMIToFailedStatus(vmi.Name)
		})
		It("should set a vmi without a pod to failed state, triggered by vmi modify event", func() {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi1", node)
			addVMI(vmi)

			Expect(vmiInformer.GetStore().Add(vmi)).To(Succeed())
			vmiFeeder.Modify(vmi)
			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				a, _ := action.(k8stesting.ListAction)
				if strings.Contains(a.GetListRestrictions().Labels.String(), "virt-handler") {
					return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewVirtHandlerPod(node.Name)}}, nil
				}

				return true, &k8sv1.PodList{}, nil
			})

			sanityExecute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			expectVMIToFailedStatus(vmi.Name)
		})
		It("should set a vmi with an unhealthy pod to failed state, triggered by vmi modify event", func() {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi1", node)
			addVMI(vmi)

			Expect(vmiInformer.GetStore().Add(vmi)).To(Succeed())
			vmiFeeder.Modify(vmi)
			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				a, _ := action.(k8stesting.ListAction)
				if strings.Contains(a.GetListRestrictions().Labels.String(), "virt-handler") {
					return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewVirtHandlerPod(node.Name)}}, nil
				}

				return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewUnhealthyPodForVirtualMachine("whatever", vmi)}}, nil
			})

			sanityExecute()
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			expectVMIToFailedStatus(vmi.Name)
		})

		DescribeTable("should ignore a vmi which still has a healthy pod in", func(phase v1.VirtualMachineInstancePhase) {
			node := NewUnhealthyNode("testnode")
			vmi := watchtesting.NewRunningVirtualMachine("vmi", node)
			vmi.Status.Phase = phase
			addVMI(vmi)
			vmi1 := watchtesting.NewRunningVirtualMachine("vmi1", node)
			vmi1.Status.Phase = phase
			addVMI(vmi1)

			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewHealthyPodForVirtualMachine("whatever", vmi1)}}, nil
			})

			Expect(controller.checkVirtLauncherPodsAndUpdateVMIStatus(node.Name, []*v1.VirtualMachineInstance{vmi}, log.DefaultLogger())).To(Succeed())
			testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			By("checking that only a vmi with a pod gets removed")
			expectVMIToFailedStatus(vmi.Name)
			updatedVMI, err := fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi1.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVMI.Status.Phase).To(Equal(phase))
		},
			Entry("running state", v1.Running),
			Entry("scheduled state", v1.Scheduled),
		)
	})

	Context("check for orphaned vmis", func() {

		var node *k8sv1.Node
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			node = NewHealthyNode("testnode")
			vmi = watchtesting.NewRunningVirtualMachine("vmi", node)
		})

		DescribeTable("testing orpahned event", func(returnVirtHandler bool, ds *appv1.DaemonSet, hasrunningvmi bool, expectEvent bool) {

			kubeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				if returnVirtHandler {
					return true, &k8sv1.PodList{Items: []k8sv1.Pod{*NewVirtHandlerPod(node.Name)}}, nil
				}

				return true, &k8sv1.PodList{}, nil
			})

			kubeClient.Fake.PrependReactor("list", "daemonsets", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, &appv1.DaemonSetList{Items: []appv1.DaemonSet{*ds}}, nil
			})

			vmis := []*v1.VirtualMachineInstance{}
			if hasrunningvmi {
				vmis = []*v1.VirtualMachineInstance{vmi}
			}

			err := controller.createEventIfNodeHasOrphanedVMIs(node, vmis)
			Expect(err).ToNot(HaveOccurred())
			if expectEvent {
				testutils.ExpectEvent(recorder, NodeUnresponsiveReason)
			}
		},
			Entry("is not created when no vmis", true, &appv1.DaemonSet{}, false, false),
			Entry("is not created when virt-handler is running", true, &appv1.DaemonSet{}, true, false),
			Entry("is not created when daemonSet is not stable", false, UnHealthVirtHandlerDS(), true, false),
			Entry("is created when virt-handler is missing on node with vmis", false, HealthVirtHandlerDS(), true, true),
		)
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
	})

})

func NewHealthyNode(nodeName string) *k8sv1.Node {
	return &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Annotations: map[string]string{
				v1.VirtHandlerHeartbeat: nowAsJSONWithOffset(0),
			},
			Labels: map[string]string{
				v1.NodeSchedulable: "true",
			},
		},
	}
}

func HealthVirtHandlerDS() *appv1.DaemonSet {
	ds := newVirtHanderDS()
	ds.Status = appv1.DaemonSetStatus{
		CurrentNumberScheduled: 1,
		DesiredNumberScheduled: 1,
		NumberReady:            1,
	}

	return ds
}

func UnHealthVirtHandlerDS() *appv1.DaemonSet {
	ds := newVirtHanderDS()
	ds.Status = appv1.DaemonSetStatus{
		CurrentNumberScheduled: 0,
		DesiredNumberScheduled: 1,
		NumberReady:            0,
	}

	return ds
}

func newVirtHanderDS() *appv1.DaemonSet {
	return &appv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-handler",
			Labels: map[string]string{
				"kubevirt.io": "virt-handler",
			},
		},
	}
}

func NewUnhealthyNode(nodeName string) *k8sv1.Node {
	node := NewHealthyNode(nodeName)
	node.Annotations[v1.VirtHandlerHeartbeat] = nowAsJSONWithOffset(-10 * time.Minute)
	node.Labels[v1.NodeSchedulable] = "false"
	return node
}

func nowAsJSONWithOffset(offset time.Duration) string {
	now := metav1.Now()
	now = metav1.NewTime(now.Add(offset))

	data, err := json.Marshal(now)
	Expect(err).ToNot(HaveOccurred())
	return strings.Trim(string(data), `"`)
}

func NewVirtHandlerPod(nodeName string) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-handler",
			Labels: map[string]string{
				"kubevirt.io": "virt-handler",
			},
		},
		Spec: k8sv1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func NewUnhealthyStuckTerminatingPodForVirtualMachine(podName string, vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	pod := NewHealthyPodForVirtualMachine(podName, vmi)
	pod.Status.Phase = k8sv1.PodPending
	pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{
		{
			State: k8sv1.ContainerState{
				Terminated: &k8sv1.ContainerStateTerminated{},
			},
		},
	}
	return pod
}

func NewUnhealthyPodForVirtualMachine(podName string, vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	pod := NewHealthyPodForVirtualMachine(podName, vmi)
	pod.Status.Phase = k8sv1.PodFailed
	return pod
}

func NewHealthyPodForVirtualMachine(podName string, vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: k8sv1.NamespaceDefault,
			Labels: map[string]string{
				v1.CreatedByLabel: string(vmi.UID),
				v1.AppLabel:       "virt-launcher",
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind)},
		},
		Spec: k8sv1.PodSpec{NodeName: vmi.Status.NodeName},
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
		},
	}
}
