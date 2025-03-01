package node

import (
	"bufio"
	"bytes"
	"context"
	"testing"
	"time"

	gfh "github.com/AdaLogics/go-fuzz-headers"
	"github.com/golang/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var (
	maxResources = 3
)

// FuzzExecute add up to 3 nodes and vmis
// to the context and then runs the controller.
func FuzzExecute(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, numberOfVMs, numberOfVMI uint8) {
		fdp := gfh.NewConsumer(data)

		// Create nodes and vmis
		nodes := make([]*k8sv1.Node, 0)
		for _ = range int(numberOfVMs) % maxResources {
			node := &k8sv1.Node{}
			err := fdp.GenerateStruct(node)
			if err != nil {
				return
			}
			nodes = append(nodes, node)
		}

		vmis := make([]*virtv1.VirtualMachineInstance, 0)
		for _ = range int(numberOfVMI) % maxResources {
			vmi := &virtv1.VirtualMachineInstance{}
			err := fdp.GenerateStruct(vmi)
			if err != nil {
				return
			}
			vmis = append(vmis, vmi)
		}
		// There is no point in continuing
		// if we have not created any resources.
		if len(nodes) == 0 || len(vmis) == 0 {
			return
		}
		// ignore logs
		var b bytes.Buffer
		log.Log.SetIOWriter(bufio.NewWriter(&b))

		// Done creating resources. These are not yet in
		// the queue or any caches.

		// Create the controller
		ctrl := gomock.NewController(t)
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		fakeVirtClient := kubevirtfake.NewSimpleClientset()

		nodeInformer, nodeCs := testutils.NewFakeInformerFor(&k8sv1.Node{})
		vmiInformer, vmiCs := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})

		// We need to shut down the controller sources to avoid excessive memory usage
		defer nodeCs.Shutdown()
		defer vmiCs.Shutdown()
		recorder := record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		controller, _ := NewController(virtClient, nodeInformer, vmiInformer, recorder)

		// We need to shut down the queue to avoid excessive memory usage
		controller.Queue.ShutDown()
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue := testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		controller.recheckInterval = 10 * time.Millisecond

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceAll).Return(fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceAll)).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		kubeClient := fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().AppsV1().Return(kubeClient.AppsV1()).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action k8sTesting.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, nil
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Add the resources to the context
		for _, node := range nodes {
			controller.enqueueNode(node)
		}
		for _, vmi := range vmis {
			// Either add a VMI to the queue or create it
			addToQueue, err := fdp.GetBool()
			if err != nil {
				return
			}
			if addToQueue {
				controller.addVirtualMachine(vmi)
			} else {
				_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(ctx, vmi, metav1.CreateOptions{})
				if err != nil {
					return
				}
			}
		}

		// If the queue is empty, we won't proceed
		if controller.Queue.Len() == 0 {
			return
		}

		// Run the controller
		controller.Execute()
	})
}
