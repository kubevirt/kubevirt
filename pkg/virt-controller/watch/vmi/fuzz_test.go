package vmi

import (
	"context"
	"testing"

	gfh "github.com/AdaLogics/go-fuzz-headers"
	"github.com/golang/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	kvcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

var (
	maxResources = 3
)

// FuzzExecute add up to 3 virtual machine instances,
// pods, persistent volume claims and data volumes
// to the context and then runs the controller.
func FuzzExecute(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, numberOfVMI, numberOfPods, numberOfPVC, numberOfDataVolumes uint8) {
		fdp := gfh.NewConsumer(data)
		VMIs := make([]*virtv1.VirtualMachineInstance, 0)
		for _ = range int(numberOfVMI) % maxResources {
			vmi := &virtv1.VirtualMachineInstance{}
			err := fdp.GenerateStruct(vmi)
			if err != nil {
				return
			}
			VMIs = append(VMIs, vmi)
		}
		pods := make([]*k8sv1.Pod, 0)
		for _ = range int(numberOfPods) % maxResources {
			pod := &k8sv1.Pod{}
			err := fdp.GenerateStruct(pod)
			if err != nil {
				return
			}
			pods = append(pods, pod)
		}
		PVCs := make([]*k8sv1.PersistentVolumeClaim, 0)
		for _ = range int(numberOfPVC) % maxResources {
			pvc := &k8sv1.PersistentVolumeClaim{}
			err := fdp.GenerateStruct(pvc)
			if err != nil {
				return
			}
			PVCs = append(PVCs, pvc)
		}
		dataVolumes := make([]*cdiv1.DataVolume, 0)
		for _ = range int(numberOfDataVolumes) % maxResources {
			dataVolume := &cdiv1.DataVolume{}
			err := fdp.GenerateStruct(dataVolume)
			if err != nil {
				return
			}
			dataVolumes = append(dataVolumes, dataVolume)
		}
		// There is no point in continuing
		// if we have not created any resources.
		if len(VMIs) == 0 &&
			len(pods) == 0 &&
			len(PVCs) == 0 &&
			len(dataVolumes) == 0 {
			return
		}

		// Create the controller
		var recorder *record.FakeRecorder
		var fuzzVirtClientset *kubevirtfake.Clientset
		var config *virtconfig.ClusterConfig
		var controller *Controller
		var mockQueue *testutils.MockWorkQueue[string]
		var kubeClient *fake.Clientset

		fuzzVirtClient := kubecli.NewMockKubevirtClient(gomock.NewController(t))
		fuzzVirtClientset = kubevirtfake.NewSimpleClientset()

		vmiInformer, _ := testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachineInstance{}, kvcontroller.GetVMIInformerIndexers())

		vmInformer, _ := testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachine{}, kvcontroller.GetVirtualMachineInformerIndexers())
		podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
		dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		storageProfileInformer, _ := testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		kubevirtFakeConfig := &virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{
				MinimumClusterTSCFrequency: pointer.P(int64(12345)),
			},
		}

		config, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(kubevirtFakeConfig)
		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		storageClassInformer, _ := testutils.NewFakeInformerFor(&storagev1.StorageClass{})
		cdiInformer, _ := testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
		cdiConfigInformer, _ := testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
		rqInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		nsInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		var qemuGid int64 = 107

		stubNetStatusUpdate := func(vmi *virtv1.VirtualMachineInstance, _ *k8sv1.Pod) error {
			vmi.Status.Interfaces = append(vmi.Status.Interfaces, virtv1.VirtualMachineInstanceNetworkInterface{Name: "stubNetStatusUpdate"})
			return nil
		}

		controller, err := NewController(
			services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", pvcInformer.GetStore(), fuzzVirtClient, config, qemuGid, "g", rqInformer.GetStore(), nsInformer.GetStore()),
			vmiInformer,
			vmInformer,
			podInformer,
			pvcInformer,
			storageClassInformer,
			recorder,
			fuzzVirtClient,
			dataVolumeInformer,
			storageProfileInformer,
			cdiInformer,
			cdiConfigInformer,
			config,
			topology.NewTopologyHinter(&cache.FakeCustomStore{}, &cache.FakeCustomStore{}, config),
			stubNetworkAnnotationsGenerator{},
			stubNetStatusUpdate,
			validateNetVMISpecStub(),
		)
		if err != nil {
			return
		}
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		// Set up mock client
		kubeClient = fake.NewSimpleClientset()
		fuzzVirtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		// Add the resources to the context
		for _, vmi := range VMIs {
			err := controller.vmiIndexer.Add(vmi)
			if err != nil {
				return
			}
			key, err := kvcontroller.KeyFunc(vmi)
			if err != nil {
				return
			}
			mockQueue.Add(key)
			_, err = fuzzVirtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			if err != nil {
				return
			}
			fuzzVirtClient.EXPECT().VirtualMachineInstance(vmi.ObjectMeta.Namespace).Return(
				fuzzVirtClientset.KubevirtV1().VirtualMachineInstances(vmi.ObjectMeta.Namespace),
			).AnyTimes()
		}
		for _, pod := range pods {
			err := controller.podIndexer.Add(pod)
			if err != nil {
				return
			}
			key, err := kvcontroller.KeyFunc(pod)
			if err != nil {
				return
			}
			mockQueue.Add(key)
			_, err = kubeClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
			if err != nil {
				return
			}
		}
		for _, pvc := range PVCs {
			err := controller.pvcIndexer.Add(pvc)
			if err != nil {
				return
			}
			key, err := kvcontroller.KeyFunc(pvc)
			if err != nil {
				return
			}
			mockQueue.Add(key)
			_, err = kubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			if err != nil {
				return
			}
		}
		for _, dataVolume := range dataVolumes {
			err := controller.dataVolumeIndexer.Add(dataVolume)
			if err != nil {
				return
			}
			key, err := kvcontroller.KeyFunc(dataVolume)
			if err != nil {
				return
			}
			mockQueue.Add(key)
		}
		if mockQueue.Len() == 0 {
			return
		}

		// Run the controller
		controller.Execute()
	})
}