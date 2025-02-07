package vm

import (
	"testing"

	gfh "github.com/AdaLogics/go-fuzz-headers"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	instancetypecontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var (
	maxResources = 3
)

// FuzzExecute add up to 3 virtual machines
// to the context and then runs the controller.
func FuzzExecute(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, numberOfVMs uint8) {
		fdp := gfh.NewConsumer(data)

		//vm *v1.VirtualMachine
		vms := make([]*v1.VirtualMachine, 0)
		for _ = range int(numberOfVMs) % maxResources {
			vm := &v1.VirtualMachine{}
			err := fdp.GenerateStruct(vm)
			if err != nil {
				return
			}
			vms = append(vms, vm)
		}
		// There is no point in continuing
		// if we have not created any resources.
		if len(vms) == 0 {
			return
		}

		virtClient := kubecli.NewMockKubevirtClient(gomock.NewController(t))
		virtFakeClient := fake.NewSimpleClientset()
		// enable /status, this assumes that no other reactor will be prepend.
		// if you need to prepend reactor it need to not handle the object or use the
		// modify function
		virtFakeClient.PrependReactor("update", "virtualmachines",
			UpdateReactor(SubresourceHandle, virtFakeClient.Tracker(), ModifyStatusOnlyVM))
		virtFakeClient.PrependReactor("update", "virtualmachines",
			UpdateReactor(Handle, virtFakeClient.Tracker(), ModifyVM))

		virtFakeClient.PrependReactor("patch", "virtualmachines",
			PatchReactor(SubresourceHandle, virtFakeClient.Tracker(), ModifyStatusOnlyVM))
		virtFakeClient.PrependReactor("patch", "virtualmachines",
			PatchReactor(Handle, virtFakeClient.Tracker(), ModifyVM))

		dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		dataSourceInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
		vmiInformer, _ := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, virtcontroller.GetVMIInformerIndexers())
		vmInformer, _ := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		namespaceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})

		ns1 := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns1",
			},
		}
		ns2 := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		}
		if namespaceInformer.GetStore().Add(ns1) != nil {
			panic("Should not happen")
		}
		if namespaceInformer.GetStore().Add(ns2) != nil {
			panic("Should not happen")
		}

		crInformer, _ := testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
			"vm": func(obj interface{}) ([]string, error) {
				cr := obj.(*appsv1.ControllerRevision)
				for _, ref := range cr.OwnerReferences {
					if ref.Kind == "VirtualMachine" {
						return []string{string(ref.UID)}, nil
					}
				}
				return nil, nil
			},
		})
		podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})

		recorder := record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		controller, err := NewController(vmiInformer,
			vmInformer,
			dataVolumeInformer,
			dataSourceInformer,
			namespaceInformer.GetStore(),
			pvcInformer,
			crInformer,
			podInformer,
			recorder,
			virtClient,
			config,
			nil,
			instancetypecontroller.NewMockController(),
		)
		if err != nil {
			return
		}

		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue := testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault),
		).AnyTimes()

		
		// TODO Remove GeneratedKubeVirtClient (like in vm_test.go)
		virtClient.EXPECT().GeneratedKubeVirtClient().Return(virtFakeClient).AnyTimes()

		cdiClient := cdifake.NewSimpleClientset()
		virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
		cdiClient.Fake.PrependReactor("*", "*", func(action k8sTesting.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, nil
		})

		k8sClient := k8sfake.NewSimpleClientset()
		virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().AuthorizationV1().Return(k8sClient.AuthorizationV1()).AnyTimes()

		// Add the resources to the context
		for _, vm := range vms {
			key, err := virtcontroller.KeyFunc(vm)
			if err != nil {
				return
			}
			controller.Queue.Add(key)
			err = controller.vmIndexer.Add(vm)
			if err != nil {
				return
			}
			virtClient.EXPECT().VirtualMachine(vm.Namespace).Return(
				virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace),
			).AnyTimes()
		}

		// Run the controller
		controller.Execute()
	})
}
