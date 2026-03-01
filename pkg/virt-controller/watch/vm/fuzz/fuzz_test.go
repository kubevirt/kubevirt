package fuzz

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	stdruntime "runtime"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"

	fuzz "github.com/google/gofuzz"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	KVv1 "kubevirt.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vm"

	framework "k8s.io/client-go/tools/cache/testing"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	instancetypecontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	maxResources      = 3
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
		if len(data) < 100 {
			return
		}
		currentName := 1
		fuzzConsumer := fuzz.NewFromGoFuzz(data)

		namespaceInformer, nsCs := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		vms := make([]*v1.VirtualMachine, 0)
		for range int(numberOfVMs) % maxResources {
			virtualMachine := &v1.VirtualMachine{}
			fuzzConsumer.Fuzz(virtualMachine)
			virtualMachine.TypeMeta = metav1.TypeMeta{
				Kind:       "VirtualMachine",
				APIVersion: k8sv1.SchemeGroupVersion.String(),
			}
			if virtualMachine.GetObjectMeta().GetName() == "" {
				name := fmt.Sprintf("name-%d", currentName)
				currentName += 1
				virtualMachine.Name = name
			}
			if virtualMachine.GetObjectMeta().GetNamespace() == "" {
				virtualMachine.Namespace = k8sv1.NamespaceDefault
			}

			if virtualMachine.Spec.Template == nil {
				virtualMachine.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:   virtualMachine.ObjectMeta.Name,
						Labels: virtualMachine.ObjectMeta.Labels,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							CPU: &v1.CPU{
								Cores: 4,
							},
						},
					},
				}
			}

			if virtualMachine.Status.StartFailure != nil {
				oldRetry := time.Now().Add(1 * time.Millisecond)
				virtualMachine.Status.StartFailure = &v1.VirtualMachineStartFailure{
					LastFailedVMIUID:     "123",
					ConsecutiveFailCount: 1,
					RetryAfterTimestamp: &metav1.Time{
						Time: oldRetry,
					},
				}
			}
			volRequests := make([]v1.VirtualMachineVolumeRequest, 0)
			for _, request := range virtualMachine.Status.VolumeRequests {
				hpSource := v1.HotplugVolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name:         fmt.Sprintf("hotplug-dv-%d", currentName),
						Hotpluggable: true,
					},
				}
				currentName += 1
				disk := v1.Disk{
					Name: fmt.Sprintf("hotplug-dv-%d", currentName),
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{Bus: v1.DiskBusSCSI},
					},
				}
				currentName += 1

				request = v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         fmt.Sprintf("hotplug-dv-%d", currentName),
						VolumeSource: &hpSource,
						Disk:         &disk,
					},
				}
				currentName += 1

				if request.AddVolumeOptions == nil {
					request.AddVolumeOptions = &v1.AddVolumeOptions{
						Name:         fmt.Sprintf("hotplug-dv-%d", currentName),
						VolumeSource: &hpSource,
						Disk:         &disk,
					}
					currentName += 1
					volRequests = append(volRequests, request)
					continue
				}
				if request.AddVolumeOptions.VolumeSource == nil {
					request.AddVolumeOptions.VolumeSource = &hpSource
					volRequests = append(volRequests, request)
					continue
				}
			}
			virtualMachine.Status.VolumeRequests = volRequests

			var setLatestAnnotation bool
			fuzzConsumer.Fuzz(&setLatestAnnotation)
			// This helps the vm overcome some checks early in the callgraph
			if setLatestAnnotation {
				virtcontroller.SetLatestApiVersionAnnotation(virtualMachine)
			}
			ns := &k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: virtualMachine.Namespace,
				},
			}
			if err := namespaceInformer.GetStore().Add(ns); err != nil {
				panic(err)
				continue
			}
			defer namespaceInformer.GetStore().Delete(virtualMachine.Namespace)
			vms = append(vms, virtualMachine)
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
			vm.UpdateReactor(vm.SubresourceHandle, virtFakeClient.Tracker(), vm.ModifyStatusOnlyVM))
		virtFakeClient.PrependReactor("update", "virtualmachines",
			vm.UpdateReactor(vm.Handle, virtFakeClient.Tracker(), vm.ModifyVM))

		virtFakeClient.PrependReactor("patch", "virtualmachines",
			vm.PatchReactor(vm.SubresourceHandle, virtFakeClient.Tracker(), vm.ModifyStatusOnlyVM))
		virtFakeClient.PrependReactor("patch", "virtualmachines",
			vm.PatchReactor(vm.Handle, virtFakeClient.Tracker(), vm.ModifyVM))

		for _, virtualMachine := range vms {
			virtClient.EXPECT().VirtualMachineInstance(virtualMachine.Namespace).Return(
				virtFakeClient.KubevirtV1().VirtualMachineInstances(virtualMachine.Namespace),
			).AnyTimes()
		}

		dataVolumeInformer, dvCs := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		dataSourceInformer, dsCs := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
		kvInformer, unused := testutils.NewFakeInformerFor(&v1.KubeVirt{})
		vmiInformer, viCs := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, virtcontroller.GetVMIInformerIndexers())
		vmInformer, vCs := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
		pvcInformer, piCs := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})

		for _, virtualMachine := range vms {
			err := vmInformer.GetIndexer().Add(virtualMachine)
			if err != nil {
				return
			}
		}

		// When running this fuzzer on OSS-Fuzz, we need to shut down
		// the controller sources to avoid consuming too much memory.
		defer dvCs.Shutdown()
		defer dsCs.Shutdown()
		defer viCs.Shutdown()
		defer vCs.Shutdown()
		defer piCs.Shutdown()
		defer nsCs.Shutdown()
		defer unused.Shutdown()

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
			t.Fatal("Should not happen")
		}
		if namespaceInformer.GetStore().Add(ns2) != nil {
			t.Fatal("Should not happen")
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
		defer func() {
			for _, obj := range crdInformer.GetStore().List() {
				err := crdInformer.GetStore().Delete(obj)
				if err != nil {
					panic(err)
				}
			}
		}()

		controller, err := vm.NewController(vmiInformer,
			vmInformer,
			dataVolumeInformer,
			dataSourceInformer,
			kvInformer,
			namespaceInformer,
			pvcInformer,
			crInformer,
			recorder,
			virtClient,
			config,
			nil,
			nil,
			instancetypecontroller.NewControllerStub(),
			[]string{},
			[]string{},
		)
		if err != nil {
			return
		}
		// Shutdown default queue to avoid memory creep
		defer controller.Queue.ShutDown()

		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue := testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

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
		for _, virtualMachine := range vms {
			// We can create the VM by way of the client or put the vm in the queue.
			var create bool
			fuzzConsumer.Fuzz(&create)
			if create {
				_, err := virtFakeClient.KubevirtV1().VirtualMachines(virtualMachine.Namespace).Create(context.Background(), virtualMachine, metav1.CreateOptions{})
				if err != nil {
					return
				}
			}
			key, err := virtcontroller.KeyFunc(virtualMachine)
			if err != nil {
				return
			}
			controller.Queue.Add(key)
			virtClient.EXPECT().VirtualMachine(virtualMachine.Namespace).Return(
				virtFakeClient.KubevirtV1().VirtualMachines(virtualMachine.Namespace),
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
		for i := controller.Queue.Len(); i > 0; i-- {
			controller.Execute()
		}
	})
}
