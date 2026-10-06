package framework

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	instancetypevmcontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vm"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vmi"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

const (
	testNodeName  = "functional-node"
	testNamespace = "kubevirt"

	launcherQemuTimeout = 240
	launcherSubGID      = 107
	controllerWorkers   = 3
	namespaceSuffixLen  = 8
	// Allow time for cold CRD watch caches to initialize on slower CI workers.
	namespaceCleanupTimeout  = time.Minute
	namespaceCleanupInterval = 100 * time.Millisecond

	testNodeCPU     = "8"
	testNodeMemory  = "16Gi"
	testNodeMaxPods = "110"
)

type Framework struct {
	env           *envtest.Environment
	virtClient    kubecli.KubevirtClient
	k8sClient     kubernetes.Interface
	dynamicClient dynamic.Interface

	vmController  *vm.Controller
	vmiController *vmi.Controller

	podSimulator *PodSimulator

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// informerSet groups all KubeVirt informers used by the framework controllers.
type informerSet struct {
	vmi           cache.SharedIndexInformer
	vm            cache.SharedIndexInformer
	pod           cache.SharedIndexInformer
	pvc           cache.SharedIndexInformer
	migration     cache.SharedIndexInformer
	storageClass  cache.SharedIndexInformer
	namespace     cache.SharedIndexInformer
	cr            cache.SharedIndexInformer
	kubeVirt      cache.SharedIndexInformer
	resourceQuota cache.SharedIndexInformer
}

// cdiInformerSet groups the fake CDI informers used in place of real CDI controllers.
type cdiInformerSet struct {
	dataVolume     cache.SharedIndexInformer
	dataSource     cache.SharedIndexInformer
	storageProfile cache.SharedIndexInformer
	cdi            cache.SharedIndexInformer
	cdiConfig      cache.SharedIndexInformer
}

func (i informerSet) all() []cache.SharedIndexInformer {
	return []cache.SharedIndexInformer{
		i.vmi, i.vm, i.pod, i.pvc, i.migration, i.storageClass,
		i.namespace, i.cr, i.kubeVirt, i.resourceQuota,
	}
}

func (c cdiInformerSet) all() []cache.SharedIndexInformer {
	return []cache.SharedIndexInformer{c.dataVolume, c.dataSource, c.storageProfile, c.cdi, c.cdiConfig}
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

func (f *Framework) Start(ctx context.Context) {
	GinkgoHelper()
	f.ctx, f.cancel = context.WithCancel(ctx)
	DeferCleanup(f.Stop)

	cfg, err := f.env.Start()
	Expect(err).NotTo(HaveOccurred(), "failed to start envtest")

	f.initClients(cfg)
	f.createSeedData()

	inf := f.buildInformers()
	cdi := newFakeCDIInformers()

	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})
	// A nil Events channel discards events, including during controller shutdown.
	recorder := new(record.FakeRecorder)

	f.vmiController = buildVMIController(f.virtClient, inf, cdi, clusterConfig, recorder)
	f.vmController = buildVMController(f.virtClient, inf, cdi, clusterConfig, recorder)
	f.podSimulator = NewPodSimulator(f.k8sClient, inf.pod, testNodeName)

	var syncs []cache.InformerSynced
	for _, informer := range append(inf.all(), cdi.all()...) {
		syncs = append(syncs, informer.HasSynced)
		f.wg.Go(func() { informer.RunWithContext(f.ctx) })
	}
	Expect(cache.WaitForCacheSync(f.ctx.Done(), syncs...)).To(BeTrue(), "failed to sync informer caches")

	f.wg.Go(func() { f.vmiController.Run(controllerWorkers, f.ctx.Done()) })
	f.wg.Go(func() { f.vmController.Run(controllerWorkers, f.ctx.Done()) })
	f.podSimulator.Start(f.ctx)
}

func (f *Framework) Stop() {
	GinkgoHelper()
	if f.cancel != nil {
		f.cancel()
	}
	if f.podSimulator != nil {
		f.podSimulator.Stop()
	}
	f.wg.Wait()
	if f.env != nil {
		Expect(f.env.Stop()).To(Succeed())
	}
}

func (f *Framework) VirtClient() kubecli.KubevirtClient {
	return f.virtClient
}

func (f *Framework) CreateNamespace(ctx context.Context) string {
	GinkgoHelper()
	ns := fmt.Sprintf("test-%s", rand.String(namespaceSuffixLen))
	_, err := f.virtClient.CoreV1().Namespaces().Create(ctx, &k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create namespace %s", ns))
	DeferCleanup(func(ctx SpecContext) {
		Expect(f.cleanupNamespace(ns)).To(Succeed())
		_, err := f.virtClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
		vms, err := f.virtClient.VirtualMachine(ns).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vms.Items).To(BeEmpty())
		vmis, err := f.virtClient.VirtualMachineInstance(ns).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vmis.Items).To(BeEmpty())
		pods, err := f.virtClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).To(BeEmpty())
	})
	return ns
}

// envtest has no namespace or garbage collection controllers. Delete all
// namespaced resources and clear their finalizers before finalizing the namespace.
func (f *Framework) cleanupNamespace(ns string) error {
	ctx, cancel := context.WithTimeout(f.ctx, namespaceCleanupTimeout)
	defer cancel()
	if err := f.virtClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("deleting namespace %s: %w", ns, err)
	}
	resources, err := f.k8sClient.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return fmt.Errorf("discovering namespaced resources: %w", err)
	}
	err = wait.PollUntilContextCancel(ctx, namespaceCleanupInterval, true, func(ctx context.Context) (bool, error) {
		return cleanupResources(ctx, f.dynamicClient, resources, ns)
	})
	if err != nil {
		return fmt.Errorf("cleaning resources in namespace %s: %w", ns, err)
	}
	nsObj, err := f.virtClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting namespace %s after deletion: %w", ns, err)
	}
	nsObj.Spec.Finalizers = nil
	if _, err = f.virtClient.CoreV1().Namespaces().Finalize(ctx, nsObj, metav1.UpdateOptions{}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("finalizing namespace %s: %w", ns, err)
	}
	return nil
}

func cleanupResources(ctx context.Context, client dynamic.Interface, resources []*metav1.APIResourceList, ns string) (bool, error) {
	empty := true
	for _, resourceList := range resources {
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return false, err
		}
		for _, resource := range resourceList.APIResources {
			if !resource.Namespaced || !slices.Contains(resource.Verbs, "list") || !slices.Contains(resource.Verbs, "delete") {
				continue
			}
			resourceEmpty, cleanupErr := cleanupResource(ctx, client.Resource(gv.WithResource(resource.Name)).Namespace(ns))
			if cleanupErr != nil {
				return false, cleanupErr
			}
			empty = empty && resourceEmpty
		}
	}
	return empty, nil
}

func cleanupResource(ctx context.Context, api dynamic.ResourceInterface) (bool, error) {
	objects, err := api.List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, object := range objects.Items {
		err = api.Delete(ctx, object.GetName(), metav1.DeleteOptions{GracePeriodSeconds: new(int64)})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if len(object.GetFinalizers()) > 0 {
			payload, err := patch.New(patch.WithAdd("/metadata/finalizers", nil)).GeneratePayload()
			if err != nil {
				return false, err
			}
			_, err = api.Patch(ctx, object.GetName(), types.JSONPatchType, payload, metav1.PatchOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			}
		}
	}
	return len(objects.Items) == 0, nil
}

func (f *Framework) initClients(cfg *rest.Config) {
	GinkgoHelper()
	var err error
	f.virtClient, err = kubecli.GetKubevirtClientFromRESTConfig(cfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create kubevirt client")
	f.k8sClient, err = kubecli.GetK8sClientFromRESTConfig(cfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create k8s client")
	f.dynamicClient, err = dynamic.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create dynamic client")
	matcher.SetClient(func() kubecli.KubevirtClient { return f.virtClient })
}

func (f *Framework) buildInformers() informerSet {
	factory := controller.NewKubeInformerFactory(f.virtClient.RestClient(), f.virtClient, f.virtClient, nil, testNamespace)
	return informerSet{
		vmi:           factory.VMI(),
		vm:            factory.VirtualMachine(),
		pod:           factory.KubeVirtPod(),
		pvc:           factory.PersistentVolumeClaim(),
		migration:     factory.VirtualMachineInstanceMigration(),
		storageClass:  factory.StorageClass(),
		namespace:     factory.Namespace(),
		cr:            factory.ControllerRevision(),
		kubeVirt:      factory.KubeVirt(),
		resourceQuota: factory.ResourceQuota(),
	}
}

func newFakeCDIInformers() cdiInformerSet {
	dv, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
	ds, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
	sp, _ := testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
	cdi, _ := testutils.NewFakeInformerFor(&cdiv1.CDI{})
	cdiCfg, _ := testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
	return cdiInformerSet{
		dataVolume:     dv,
		dataSource:     ds,
		storageProfile: sp,
		cdi:            cdi,
		cdiConfig:      cdiCfg,
	}
}

func buildVMIController(
	virtClient kubecli.KubevirtClient,
	inf informerSet,
	cdi cdiInformerSet,
	clusterConfig *virtconfig.ClusterConfig,
	recorder record.EventRecorder,
) *vmi.Controller {
	GinkgoHelper()
	templateService := services.NewTemplateService(
		"virt-launcher:latest",
		launcherQemuTimeout,
		"/var/run/kubevirt",
		"/var/run/kubevirt-ephemeral-disks",
		"/container-disks",
		"/hotplug-disks",
		"",
		inf.pvc.GetStore(),
		virtClient,
		clusterConfig,
		launcherSubGID,
		"virt-exportserver:latest",
		inf.resourceQuota.GetStore(),
		inf.namespace.GetStore(),
	)
	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "envtest-vmi"},
	)
	ctrl, err := vmi.NewController(
		queue,
		templateService,
		inf.vmi,
		inf.vm,
		inf.pod,
		inf.pvc,
		inf.migration,
		inf.storageClass,
		recorder,
		virtClient,
		cdi.dataVolume,
		cdi.storageProfile,
		cdi.cdi,
		cdi.cdiConfig,
		inf.kubeVirt,
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
	return ctrl
}

func buildVMController(
	virtClient kubecli.KubevirtClient,
	inf informerSet,
	cdi cdiInformerSet,
	clusterConfig *virtconfig.ClusterConfig,
	recorder record.EventRecorder,
) *vm.Controller {
	GinkgoHelper()
	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "envtest-vm"},
	)
	ctrl, err := vm.NewController(
		queue,
		inf.vmi,
		inf.vm,
		cdi.dataVolume,
		cdi.dataSource,
		inf.kubeVirt,
		inf.namespace,
		inf.pvc,
		inf.cr,
		recorder,
		virtClient,
		clusterConfig,
		&noopSynchronizer{},
		&noopSynchronizer{},
		instancetypevmcontroller.NewControllerStub(),
		nil,
		nil,
	)
	Expect(err).NotTo(HaveOccurred(), "failed to create VM controller")
	return ctrl
}

func (f *Framework) createSeedData() {
	GinkgoHelper()
	for _, ns := range []string{"default", testNamespace} {
		_, err := f.virtClient.CoreV1().Namespaces().Create(f.ctx, &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create namespace %s", ns))
		}
	}

	_, err := f.virtClient.CoreV1().Nodes().Create(f.ctx, &k8sv1.Node{
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
