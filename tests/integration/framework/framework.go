package framework

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	instancetypevmcontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vm"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vmi"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

const (
	testNodeName  = "functional-node"
	testNamespace = "kubevirt"

	recorderBufferSize  = 1000
	launcherQemuTimeout = 240
	launcherSubGID      = 107
	controllerWorkers   = 3

	testNodeCPU     = "8"
	testNodeMemory  = "16Gi"
	testNodeMaxPods = "110"
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
	GinkgoHelper()
	crds, err := loadCRDs()
	Expect(err).NotTo(HaveOccurred(), "failed to load CRDs")
	return &Framework{
		env: &envtest.Environment{
			CRDs: crds,
		},
	}
}

func (f *Framework) Start() { //nolint:funlen
	GinkgoHelper()
	ctx := context.Background()

	cfg, err := f.env.Start()
	Expect(err).NotTo(HaveOccurred(), "failed to start envtest")

	f.virtClient, err = kubecli.GetKubevirtClientFromRESTConfig(cfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create kubevirt client")

	f.k8sClient, err = kubecli.GetK8sClientFromRESTConfig(cfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create k8s client")

	matcher.SetClient(func() kubecli.KubevirtClient { return f.virtClient })

	f.createSeedData(ctx)

	f.stopCh = make(chan struct{})

	restClient := f.virtClient.RestClient()
	informerFactory := controller.NewKubeInformerFactory(restClient, f.virtClient, f.virtClient, nil, testNamespace)

	vmiInformer := informerFactory.VMI()
	vmInformer := informerFactory.VirtualMachine()
	podInformer := informerFactory.KubeVirtPod()
	pvcInformer := informerFactory.PersistentVolumeClaim()
	migrationInformer := informerFactory.VirtualMachineInstanceMigration()
	storageClassInformer := informerFactory.StorageClass()
	namespaceInformer := informerFactory.Namespace()
	crInformer := informerFactory.ControllerRevision()
	kubeVirtInformer := informerFactory.KubeVirt()

	dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
	dataSourceInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
	storageProfileInformer, _ := testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
	cdiInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
	cdiConfigInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})

	go dataVolumeInformer.Run(f.stopCh)
	go dataSourceInformer.Run(f.stopCh)
	go storageProfileInformer.Run(f.stopCh)
	go cdiInformer.Run(f.stopCh)
	go cdiConfigInformer.Run(f.stopCh)

	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})

	recorder := record.NewFakeRecorder(recorderBufferSize)

	templateService := services.NewTemplateService(
		"virt-launcher:latest",
		launcherQemuTimeout,
		"/var/run/kubevirt",
		"/var/run/kubevirt-ephemeral-disks",
		"/container-disks",
		"/hotplug-disks",
		"",
		pvcInformer.GetStore(),
		f.virtClient,
		clusterConfig,
		launcherSubGID,
		"virt-exportserver:latest",
		informerFactory.ResourceQuota().GetStore(),
		namespaceInformer.GetStore(),
	)

	vmiQueue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "envtest-vmi"},
	)
	vmiCtrl, err := vmi.NewController(
		vmiQueue,
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
		&noopVsockAllocator{},
		nil,
		nil,
	)
	Expect(err).NotTo(HaveOccurred(), "failed to create VMI controller")
	f.vmiController = vmiCtrl

	vmQueue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "envtest-vm"},
	)
	vmCtrl, err := vm.NewController(
		vmQueue,
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
	Expect(err).NotTo(HaveOccurred(), "failed to create VM controller")
	f.vmController = vmCtrl

	f.podSimulator = NewPodSimulator(f.k8sClient, podInformer, testNodeName)

	informerFactory.Start(f.stopCh)
	informerFactory.WaitForCacheSync(f.stopCh)

	go f.vmiController.Run(controllerWorkers, f.stopCh)
	go f.vmController.Run(controllerWorkers, f.stopCh)

	f.podSimulator.Start()
}

func (f *Framework) Stop() {
	GinkgoHelper()
	if f.podSimulator != nil {
		f.podSimulator.Stop()
	}
	if f.stopCh != nil {
		close(f.stopCh)
	}
	if f.env != nil {
		Expect(f.env.Stop()).To(Succeed())
	}
}

func (f *Framework) VirtClient() kubecli.KubevirtClient {
	return f.virtClient
}

func (f *Framework) createSeedData(ctx context.Context) {
	GinkgoHelper()
	for _, ns := range []string{"default", testNamespace} {
		_, err := f.virtClient.CoreV1().Namespaces().Create(ctx, &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create namespace %s", ns))
		}
	}

	_, err := f.virtClient.CoreV1().Nodes().Create(ctx, &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		Status: k8sv1.NodeStatus{
			Conditions: []k8sv1.NodeCondition{
				{
					Type:   k8sv1.NodeReady,
					Status: k8sv1.ConditionTrue,
				},
			},
			Allocatable: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse(testNodeCPU),
				k8sv1.ResourceMemory: resource.MustParse(testNodeMemory),
				k8sv1.ResourcePods:   resource.MustParse(testNodeMaxPods),
			},
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to create node")
}

func loadCRDs() ([]*extv1.CustomResourceDefinition, error) {
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
			return nil, fmt.Errorf("failed to generate CRD: %w", err)
		}
		crds = append(crds, crd)
	}
	return crds, nil
}
