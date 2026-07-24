package framework

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	instancetypevmcontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	mutating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook"
	validating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vm"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vmi"
	workloadupdater "kubevirt.io/kubevirt/pkg/virt-controller/watch/workload-updater"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virtoperator "kubevirt.io/kubevirt/pkg/virt-operator"
	operatorinstall "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

const (
	testNodeName         = "functional-node"
	testNamespace        = "kubevirt"
	defaultLauncherImage = "virt-launcher:latest"
)

type Option func(*Framework)

func WithWebhooks() Option {
	return func(f *Framework) {
		f.webhooksEnabled = true
	}
}

func WithFakeLibvirt() Option {
	return func(f *Framework) {
		f.fakeLibvirtEnabled = true
	}
}

func WithWorkloadUpdateController() Option {
	return func(f *Framework) {
		f.workloadUpdateEnabled = true
	}
}

func WithVirtOperator() Option {
	return func(f *Framework) {
		f.virtOperatorEnabled = true
	}
}

type Framework struct {
	env        *envtest.Environment
	virtClient kubecli.KubevirtClient
	k8sClient  kubernetes.Interface

	vmController  *vm.Controller
	vmiController *vmi.Controller

	podSimulator             *PodSimulator
	resourceSimulator        *ResourceSimulator
	webhookServer            *http.Server
	fakeLibvirt              *FakeLibvirt
	workloadUpdateController *workloadupdater.WorkloadUpdateController
	virtOperatorController   *virtoperator.KubeVirtController

	webhooksEnabled       bool
	fakeLibvirtEnabled    bool
	workloadUpdateEnabled bool
	virtOperatorEnabled   bool

	stopCh chan struct{}
}

func New(opts ...Option) *Framework {
	crds, err := loadCRDs()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to load CRDs")
	f := &Framework{
		env: &envtest.Environment{
			CRDs: crds,
		},
	}
	for _, opt := range opts {
		opt(f)
	}
	if f.webhooksEnabled {
		f.env.WebhookInstallOptions = envtest.WebhookInstallOptions{
			MutatingWebhooks:  filterMutatingWebhooks(components.NewVirtAPIMutatingWebhookConfiguration(testNamespace)),
			ValidatingWebhooks: filterValidatingWebhooks(components.NewVirtAPIValidatingWebhookConfiguration(testNamespace)),
		}
	}
	return f
}

func (f *Framework) Start() {
	ctx := context.Background()

	cfg, err := f.env.Start()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to start envtest")

	f.virtClient, err = kubecli.GetKubevirtClientFromRESTConfig(cfg)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create kubevirt client")
	f.k8sClient = f.virtClient

	matcher.SetClient(func() kubecli.KubevirtClient { return f.virtClient })

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
	instancetypeInformer := informerFactory.VirtualMachineInstancetype()
	clusterInstancetypeInformer := informerFactory.VirtualMachineClusterInstancetype()
	preferenceInformer := informerFactory.VirtualMachinePreference()
	clusterPreferenceInformer := informerFactory.VirtualMachineClusterPreference()

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

	recorder := record.NewFakeRecorder(1000)

	templateService := services.NewTemplateService(
		defaultLauncherImage,
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

	vmiCtrl, err := vmi.NewController(
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
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create VMI controller")
	f.vmiController = vmiCtrl

	vmCtrl, err := vm.NewController(
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
		instancetypevmcontroller.New(
			instancetypeInformer.GetStore(),
			clusterInstancetypeInformer.GetStore(),
			preferenceInformer.GetStore(),
			clusterPreferenceInformer.GetStore(),
			crInformer.GetStore(),
			f.virtClient,
			clusterConfig,
			recorder,
		),
		nil,
		nil,
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create VM controller")
	f.vmController = vmCtrl

	f.podSimulator = NewPodSimulator(f.k8sClient, podInformer, testNodeName)

	informerFactory.Start(f.stopCh)
	informerFactory.WaitForCacheSync(f.stopCh)

	go f.vmiController.Run(3, f.stopCh)
	go f.vmController.Run(3, f.stopCh)

	f.podSimulator.Start()

	if f.workloadUpdateEnabled {
		wuc, err := workloadupdater.NewWorkloadUpdateController(
			defaultLauncherImage,
			vmiInformer,
			podInformer,
			migrationInformer,
			kubeVirtInformer,
			recorder,
			f.virtClient,
			clusterConfig,
		)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create workload update controller")
		f.workloadUpdateController = wuc
		go f.workloadUpdateController.Run(f.stopCh)
	}

	if f.virtOperatorEnabled {
		operatorInformers := operatorutil.Informers{
			KubeVirt:                         kubeVirtInformer,
			CRD:                              informerFactory.CRD(),
			ServiceAccount:                   informerFactory.OperatorServiceAccount(),
			ClusterRole:                      informerFactory.OperatorClusterRole(),
			ClusterRoleBinding:               informerFactory.OperatorClusterRoleBinding(),
			Role:                             informerFactory.OperatorRole(),
			RoleBinding:                      informerFactory.OperatorRoleBinding(),
			OperatorCrd:                      informerFactory.OperatorCRD(),
			Service:                          informerFactory.OperatorService(),
			Deployment:                       informerFactory.OperatorDeployment(),
			DaemonSet:                        informerFactory.OperatorDaemonSet(),
			ValidationWebhook:                informerFactory.OperatorValidationWebhook(),
			MutatingWebhook:                  informerFactory.OperatorMutatingWebhook(),
			InstallStrategyConfigMap:         informerFactory.OperatorInstallStrategyConfigMaps(),
			InstallStrategyJob:               informerFactory.OperatorInstallStrategyJob(),
			InfrastructurePod:                informerFactory.OperatorPod(),
			PodDisruptionBudget:              informerFactory.OperatorPodDisruptionBudget(),
			Namespace:                        informerFactory.Namespace(),
			Secrets:                          informerFactory.Secrets(),
			ConfigMap:                        informerFactory.OperatorConfigMap(),
			ClusterInstancetype:              informerFactory.VirtualMachineClusterInstancetype(),
			ClusterPreference:                informerFactory.VirtualMachineClusterPreference(),
			Leases:                           informerFactory.Leases(),
			SCC:                              informerFactory.DummyOperatorSCC(),
			Route:                            informerFactory.DummyOperatorRoute(),
			ServiceMonitor:                   informerFactory.DummyOperatorServiceMonitor(),
			PrometheusRule:                    informerFactory.DummyOperatorPrometheusRule(),
			APIService:                       dummyInformer(),
			ValidatingAdmissionPolicyBinding: informerFactory.DummyOperatorValidatingAdmissionPolicyBinding(),
			ValidatingAdmissionPolicy:        informerFactory.DummyOperatorValidatingAdmissionPolicy(),
		}

		// Label existing CRDs so the operator's OperatorCrd informer picks them up
		// instead of trying to create them (they were already loaded by envtest)
		crdClient := f.virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions()
		crdList, err := crdClient.List(ctx, metav1.ListOptions{})
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		for i := range crdList.Items {
			crd := &crdList.Items[i]
			if crd.Labels == nil {
				crd.Labels = make(map[string]string)
			}
			crd.Labels[virtv1.ManagedByLabel] = virtv1.ManagedByLabelOperatorValue
			_, err = crdClient.Update(ctx, crd, metav1.UpdateOptions{})
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
		}

		// Restart informer factory to pick up the new operator informers
		informerFactory.Start(f.stopCh)
		informerFactory.WaitForCacheSync(f.stopCh)

		// Set up env var manager so GetTargetConfigFromKV can resolve image names
		envVarManager := &operatorutil.EnvVarManagerMock{}
		ExpectWithOffset(1, envVarManager.Setenv(operatorutil.OldOperatorImageEnvName, "registry/virt-operator:v1.0.0")).To(Succeed())
		operatorutil.DefaultEnvVarManager = envVarManager

		// Seed install strategy ConfigMap
		seedKV := &virtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
		}
		deploymentConfig := operatorutil.GetTargetConfigFromKV(seedKV)
		strategyCM, err := operatorinstall.NewInstallStrategyConfigMap(deploymentConfig, "", testNamespace)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to generate install strategy ConfigMap")
		_, err = f.k8sClient.CoreV1().ConfigMaps(testNamespace).Create(ctx, strategyCM, metav1.CreateOptions{})
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create install strategy ConfigMap")

		// Start resource simulator for Deployment/DaemonSet readiness
		f.resourceSimulator = NewResourceSimulator(f.k8sClient, operatorInformers.Deployment, operatorInformers.DaemonSet, operatorInformers.ValidationWebhook, operatorInformers.MutatingWebhook)
		f.resourceSimulator.Start()

		operatorConfig := operatorutil.OperatorConfig{}
		opCtrl, err := virtoperator.NewKubeVirtController(
			f.virtClient,
			f.k8sClient,
			&noopAPIServiceClient{},
			recorder,
			operatorConfig,
			operatorInformers,
			testNamespace,
		)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create virt-operator controller")
		f.virtOperatorController = opCtrl
		go f.virtOperatorController.Run(1, f.stopCh)
	}

	if f.fakeLibvirtEnabled {
		tmpDir, err := os.MkdirTemp("", "envtest-fakelibvirt-")
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create temp dir for fake libvirt")
		f.fakeLibvirt = newFakeLibvirt(tmpDir, vmiInformer, f.virtClient)
		ExpectWithOffset(1, f.fakeLibvirt.Start()).To(Succeed(), "failed to start fake libvirt gRPC server")
	}

	if f.webhooksEnabled {
		f.startWebhookServer(clusterConfig)
	}
}

func (f *Framework) Stop() {
	if f.fakeLibvirt != nil {
		f.fakeLibvirt.Stop()
	}
	if f.webhookServer != nil {
		f.webhookServer.Close()
	}
	if f.resourceSimulator != nil {
		f.resourceSimulator.Stop()
	}
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

func (f *Framework) FakeLibvirt() *FakeLibvirt {
	return f.fakeLibvirt
}

func (f *Framework) LauncherImage() string {
	return defaultLauncherImage
}

func dummyInformer() cache.SharedIndexInformer {
	informer, _ := testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
	return informer
}

func (f *Framework) createSeedData(ctx context.Context) {
	for _, ns := range []string{"default", testNamespace} {
		_, err := f.k8sClient.CoreV1().Namespaces().Create(ctx, &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			ExpectWithOffset(2, err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create namespace %s", ns))
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
	ExpectWithOffset(2, err).NotTo(HaveOccurred(), "failed to create node")
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

func (f *Framework) startWebhookServer(clusterConfig *virtconfig.ClusterConfig) {
	opts := f.env.WebhookInstallOptions
	certPath := filepath.Join(opts.LocalServingCertDir, "tls.crt")
	keyPath := filepath.Join(opts.LocalServingCertDir, "tls.key")

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to load webhook TLS certs")

	webhookInformers := &webhooks.Informers{}

	mux := http.NewServeMux()
	mux.HandleFunc(components.VMMutatePath, func(w http.ResponseWriter, r *http.Request) {
		mutating_webhook.ServeVMs(w, r, clusterConfig, f.virtClient)
	})
	mux.HandleFunc(components.VMValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeVMs(w, r, clusterConfig, f.virtClient, webhookInformers, nil)
	})

	f.webhookServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", opts.LocalServingHost, opts.LocalServingPort),
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	go f.webhookServer.ListenAndServeTLS("", "")
}

func filterMutatingWebhooks(cfg *admissionregistrationv1.MutatingWebhookConfiguration) []*admissionregistrationv1.MutatingWebhookConfiguration {
	filtered := cfg.DeepCopy()
	var kept []admissionregistrationv1.MutatingWebhook
	for _, wh := range filtered.Webhooks {
		for _, rule := range wh.Rules {
			for _, res := range rule.Resources {
				if res == "virtualmachines" {
					kept = append(kept, wh)
				}
			}
		}
	}
	filtered.Webhooks = kept
	return []*admissionregistrationv1.MutatingWebhookConfiguration{filtered}
}

func filterValidatingWebhooks(cfg *admissionregistrationv1.ValidatingWebhookConfiguration) []*admissionregistrationv1.ValidatingWebhookConfiguration {
	filtered := cfg.DeepCopy()
	var kept []admissionregistrationv1.ValidatingWebhook
	for _, wh := range filtered.Webhooks {
		for _, rule := range wh.Rules {
			for _, res := range rule.Resources {
				if res == "virtualmachines" {
					kept = append(kept, wh)
				}
			}
		}
	}
	filtered.Webhooks = kept
	return []*admissionregistrationv1.ValidatingWebhookConfiguration{filtered}
}

