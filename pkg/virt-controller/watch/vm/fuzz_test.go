package vm

import (
	"bufio"
	"bytes"
	"context"
	"k8s.io/apimachinery/pkg/util/rand"
	stdruntime "runtime"
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
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	KVv1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/client-go/log"

	framework "k8s.io/client-go/tools/cache/testing"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	instancetypecontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	maxResources = 3
	kvObjectNamespace = "kubevirt"
	kvObjectName      = "kubevirt"
)

func NewFakeClusterConfigUsingKV(kv *KVv1.KubeVirt) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.Store, *framework.FakeControllerSource, *framework.FakeControllerSource) {
	return NewFakeClusterConfigUsingKVWithCPUArch(kv, stdruntime.GOARCH)
}

func NewFakeClusterConfigUsingKVWithCPUArch(kv *KVv1.KubeVirt, CPUArch string) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.Store, *framework.FakeControllerSource, *framework.FakeControllerSource) {
	kv.ResourceVersion = rand.String(10)
	kv.Status.Phase = "Deployed"
	crdInformer, cs1 := testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
	kubeVirtInformer, cs2 := testutils.NewFakeInformerFor(&KVv1.KubeVirt{})

	kubeVirtInformer.GetStore().Add(kv)

	AddDataVolumeAPI(crdInformer)
	cfg, _ := virtconfig.NewClusterConfigWithCPUArch(crdInformer, kubeVirtInformer, kvObjectNamespace, CPUArch)
	return cfg, crdInformer, kubeVirtInformer.GetStore(), cs1, cs2
}

func AddDataVolumeAPI(crdInformer cache.SharedIndexInformer) {
	crdInformer.GetStore().Add(&extv1.CustomResourceDefinition{
		Spec: extv1.CustomResourceDefinitionSpec{
			Names: extv1.CustomResourceDefinitionNames{
				Kind: "DataVolume",
			},
		},
	})
}

func NewFakeClusterConfigUsingKVConfig(kv *KVv1.KubeVirt) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, cache.Store, *framework.FakeControllerSource, *framework.FakeControllerSource) {
	return NewFakeClusterConfigUsingKV(kv)
}

// FuzzExecute add up to 3 virtual machines
// to the context and then runs the controller.
func FuzzExecute(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, numberOfVMs uint8) {
		fdp := gfh.NewConsumer(data)

		//vm *v1.VirtualMachine
		vms := make([]*v1.VirtualMachine)
		for range int(numberOfVMs) % maxResources {
			vm := &v1.VirtualMachine{}
			err := fdp.GenerateStruct(vm)
			if err != nil {
				return
			}
			vm.Namespace = metav1.NamespaceDefault

			setLatestAnnotation, err := fdp.GetBool()
			if err != nil {
				return
			}
			// This helps the vm overcome some checks early in the callgraph
			if setLatestAnnotation {
				virtcontroller.SetLatestApiVersionAnnotation(vm)
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

		dataVolumeInformer, dvCs := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		dataSourceInformer, dsCs := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
		vmiInformer, viCs := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, virtcontroller.GetVMIInformerIndexers())
		vmInformer, vCs := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
		pvcInformer, piCs := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		namespaceInformer, nsCs := testutils.NewFakeInformerFor(&k8sv1.Namespace{})

		// When running this fuzzer on OSS-Fuzz, we need to shut down
		// the controller sources to avoid consuming too much memory.
		defer dvCs.Shutdown()
		defer dsCs.Shutdown()
		defer viCs.Shutdown()
		defer vCs.Shutdown()
		defer piCs.Shutdown()
		defer nsCs.Shutdown()

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
		defer namespaceInformer.GetStore().Delete(ns1)
		defer namespaceInformer.GetStore().Delete(ns2)

		crInformer, crCs := testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
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
		defer crCs.Shutdown()

		recorder := record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		kv := &KVv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kvObjectName,
				Namespace: kvObjectNamespace,
			},
			Spec: KVv1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
			Status: KVv1.KubeVirtStatus{
				DefaultArchitecture: stdruntime.GOARCH,
				Phase:               "Deployed",
			},
		}

		config, crdInformer, kubeVirtInformerStore, cs1, cs2 := NewFakeClusterConfigUsingKVConfig(kv)
		defer cs1.Shutdown()
		defer cs2.Shutdown()
		defer kubeVirtInformerStore.Delete(kv)
		defer func(){
				for _, obj := range crdInformer.GetStore().List() {
				err := crdInformer.GetStore().Delete(obj)
				if err != nil {
					panic(err)
				}
			}
		}()

		controller, err := NewController(vmiInformer,
			vmInformer,
			dataVolumeInformer,
			dataSourceInformer,
			namespaceInformer.GetStore(),
			pvcInformer,
			crInformer,
			recorder,
			virtClient,
			config,
			nil,
			instancetypecontroller.NewMockController(),
		)
		if err != nil {
			return
		}
		// Shutdown default queue to avoid memory creep
		controller.Queue.ShutDown()

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
			// We can create the VM by way of the client or put the vm in the queue.
			create, err := fdp.GetBool() 
			if err != nil {
				return
			}
			if create {
				_, err := virtFakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				if err != nil {
					return
				}
				continue
			}
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
		// An empty queue will block the fuzzer and it will timeout
		if controller.Queue.Len() == 0 {
			return
		}
		// ignore logs
		var b bytes.Buffer
		log.Log.SetIOWriter(bufio.NewWriter(&b))

		// Run the controller
		controller.Execute()
	})
}
