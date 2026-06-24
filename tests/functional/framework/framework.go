package framework

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	instancetypevmcontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vm"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vmi"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	testNodeName  = "functional-node"
	testNamespace = "kubevirt"
)

type Framework struct {
	env        *envtest.Environment
	virtClient kubecli.KubevirtClient
	k8sClient  kubernetes.Interface

	vmController  *vm.Controller
	vmiController *vmi.Controller

	podSimulator *PodSimulator

	stopCh chan struct{}
}

func New() *Framework {
	crds := mustLoadCRDs()
	return &Framework{
		env: &envtest.Environment{
			CRDs: crds,
		},
	}
}

func (f *Framework) Start() {
	ctx := context.Background()

	cfg, err := f.env.Start()
	if err != nil {
		panic(fmt.Sprintf("failed to start envtest: %v", err))
	}

	f.virtClient, err = kubecli.GetKubevirtClientFromRESTConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create kubevirt client: %v", err))
	}
	f.k8sClient = f.virtClient

	f.createSeedData(ctx)

	f.stopCh = make(chan struct{})

	restClient := f.virtClient.RestClient()
	informerFactory := controller.NewKubeInformerFactory(restClient, f.virtClient, f.k8sClient, nil, testNamespace)

	vmiInformer := informerFactory.VMI()
	vmInformer := informerFactory.VirtualMachine()
	podInformer := informerFactory.KubeVirtPod()
	pvcInformer := informerFactory.PersistentVolumeClaim()
	migrationInformer := informerFactory.VirtualMachineInstanceMigration()
	storageClassInformer := informerFactory.StorageClass()
	namespaceInformer := informerFactory.Namespace()
	crInformer := informerFactory.ControllerRevision()
	kubeVirtInformer := informerFactory.KubeVirt()

	dataVolumeInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
	dataSourceInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
	storageProfileInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
	cdiInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
	cdiConfigInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})

	go dataVolumeInformer.Run(f.stopCh)
	go dataSourceInformer.Run(f.stopCh)
	go storageProfileInformer.Run(f.stopCh)
	go cdiInformer.Run(f.stopCh)
	go cdiConfigInformer.Run(f.stopCh)

	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})

	recorder := record.NewFakeRecorder(1000)

	templateService := services.NewTemplateService(
		"virt-launcher:latest",
		240,
		"/var/run/kubevirt",
		"/var/run/kubevirt-ephemeral-disks",
		"/container-disks",
		"/hotplug-disks",
		"",
		pvcInformer.GetStore(),
		f.virtClient,
		clusterConfig,
		107,
		"virt-exportserver:latest",
		informerFactory.ResourceQuota().GetStore(),
		namespaceInformer.GetStore(),
	)

	var vmiCtrl *vmi.Controller
	vmiCtrl, err = vmi.NewController(
		templateService,
		vmiInformer,
		vmInformer,
		podInformer,
		pvcInformer,
		migrationInformer,
		storageClassInformer,
		recorder,
		f.virtClient,
		dataVolumeInformer,
		storageProfileInformer,
		cdiInformer,
		cdiConfigInformer,
		kubeVirtInformer,
		clusterConfig,
		&noopTopologyHinter{},
		&noopAnnotationsGenerator{},
		&noopStorageAnnotationsGenerator{},
		noopStatusUpdater,
		noopSpecValidator,
		&noopMigrationEvaluator{},
		nil,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create VMI controller: %v", err))
	}
	f.vmiController = vmiCtrl

	var vmCtrl *vm.Controller
	vmCtrl, err = vm.NewController(
		vmiInformer,
		vmInformer,
		dataVolumeInformer,
		dataSourceInformer,
		kubeVirtInformer,
		namespaceInformer,
		pvcInformer,
		crInformer,
		recorder,
		f.virtClient,
		clusterConfig,
		&noopSynchronizer{},
		&noopSynchronizer{},
		instancetypevmcontroller.NewControllerStub(),
		nil,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create VM controller: %v", err))
	}
	f.vmController = vmCtrl

	f.podSimulator = NewPodSimulator(f.k8sClient, podInformer, testNodeName)

	informerFactory.Start(f.stopCh)
	informerFactory.WaitForCacheSync(f.stopCh)

	go f.vmiController.Run(3, f.stopCh)
	go f.vmController.Run(3, f.stopCh)

	f.podSimulator.Start()
}

func (f *Framework) Stop() {
	if f.podSimulator != nil {
		f.podSimulator.Stop()
	}
	if f.stopCh != nil {
		close(f.stopCh)
	}
	if f.env != nil {
		f.env.Stop()
	}
}

func (f *Framework) VirtClient() kubecli.KubevirtClient {
	return f.virtClient
}

func (f *Framework) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *Framework) createSeedData(ctx context.Context) {
	for _, ns := range []string{"default", testNamespace} {
		_, err := f.k8sClient.CoreV1().Namespaces().Create(ctx, &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			panic(fmt.Sprintf("failed to create namespace %s: %v", ns, err))
		}
	}

	_, err := f.k8sClient.CoreV1().Nodes().Create(ctx, &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		Status: k8sv1.NodeStatus{
			Conditions: []k8sv1.NodeCondition{
				{
					Type:   k8sv1.NodeReady,
					Status: k8sv1.ConditionTrue,
				},
			},
			Allocatable: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("8"),
				k8sv1.ResourceMemory: resource.MustParse("16Gi"),
				k8sv1.ResourcePods:   resource.MustParse("110"),
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to create node: %v", err))
	}
}

func mustLoadCRDs() []*extv1.CustomResourceDefinition {
	generators := []func() (*extv1.CustomResourceDefinition, error){
		components.NewVirtualMachineInstanceCrd,
		components.NewVirtualMachineCrd,
		components.NewVirtualMachineInstanceMigrationCrd,
		components.NewReplicaSetCrd,
		components.NewKubeVirtCrd,
		components.NewVirtualMachinePoolCrd,
		components.NewVirtualMachineSnapshotCrd,
		components.NewVirtualMachineSnapshotContentCrd,
		components.NewVirtualMachineRestoreCrd,
		components.NewVirtualMachineExportCrd,
		components.NewVirtualMachineInstancetypeCrd,
		components.NewVirtualMachineClusterInstancetypeCrd,
		components.NewVirtualMachinePreferenceCrd,
		components.NewVirtualMachineClusterPreferenceCrd,
		components.NewMigrationPolicyCrd,
		components.NewVirtualMachineCloneCrd,
	}

	crds := make([]*extv1.CustomResourceDefinition, 0, len(generators))
	for _, gen := range generators {
		crd, err := gen()
		if err != nil {
			panic(fmt.Sprintf("failed to generate CRD: %v", err))
		}
		crds = append(crds, crd)
	}
	return crds
}

