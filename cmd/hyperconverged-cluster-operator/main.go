package main

import (
	"context"
	"fmt"
	"maps"
	"os"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	imagev1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	openshiftroutev1 "github.com/openshift/api/route/v1"
	deschedulerv1 "github.com/openshift/cluster-kube-descheduler-operator/pkg/apis/descheduler/v1"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsapiv2 "github.com/operator-framework/api/pkg/operators/v2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	controllerruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	aaqv1alpha1 "kubevirt.io/application-aware-quota/staging/src/kubevirt.io/application-aware-quota-api/pkg/apis/core/v1alpha1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta2 "kubevirt.io/ssp-operator/api/v1beta2"

	"github.com/kubevirt/hyperconverged-cluster-operator/api"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/cmd/cmdcommon"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/crd"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/descheduler"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/hyperconverged"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/ingresscluster"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/nodes"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/observability"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/authorization"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/metrics"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/upgradepatch"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const openshiftMonitoringNamespace = "openshift-monitoring"

// Change below variables to serve metrics on different host or port.
var (
	logger               = logf.Log.WithName("hyperconverged-operator-cmd")
	cmdHelper            = cmdcommon.NewHelper(logger, "operator")
	resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
		api.AddToScheme,
		schedulingv1.AddToScheme,
		corev1.AddToScheme,
		appsv1.AddToScheme,
		rbacv1.AddToScheme,
		cdiv1beta1.AddToScheme,
		networkaddonsv1.AddToScheme,
		sspv1beta2.AddToScheme,
		csvv1alpha1.AddToScheme,
		admissionregistrationv1.AddToScheme,
		consolev1.Install,
		consolev1.Install,
		operatorv1.Install,
		openshiftconfigv1.Install,
		openshiftroutev1.Install,
		monitoringv1.AddToScheme,
		apiextensionsv1.AddToScheme,
		kubevirtcorev1.AddToScheme,
		coordinationv1.AddToScheme,
		operatorsapiv2.AddToScheme,
		imagev1.Install,
		aaqv1alpha1.AddToScheme,
		deschedulerv1.AddToScheme,
	}
)

func main() {
	cmdHelper.InitiateCommand()

	operatorNamespace := hcoutil.GetOperatorNamespaceFromEnv()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	cmdHelper.ExitOnError(err, "can't load configuration")

	// Setup Scheme for all resources
	scheme := apiruntime.NewScheme()
	cmdHelper.AddToScheme(scheme, resourcesSchemeFuncs)

	ci := hcoutil.GetClusterInfo()

	// apiclient.New() returns a client without cache.
	// cache is not initialized before mgr.Start()
	// we need this because we need to interact with OperatorCondition
	apiClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	cmdHelper.ExitOnError(err, "Cannot create a new API client")

	// Detect OpenShift version
	ctx := context.TODO()
	err = ci.Init(ctx, apiClient, logger)
	cmdHelper.ExitOnError(err, "Cannot detect cluster type")

	needLeaderElection := !ci.IsRunningLocally()

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, getManagerOptions(operatorNamespace, needLeaderElection, ci, scheme))
	cmdHelper.ExitOnError(err, "can't initiate manager")

	// register pprof instrumentation if HCO_PPROF_ADDR is set
	cmdHelper.ExitOnError(cmdHelper.RegisterPPROFServer(mgr), "can't register pprof server")

	logger.Info("Registering Components.")

	eventEmitter := hcoutil.GetEventEmitter()
	eventEmitter.Init(ci.GetPod(), ci.GetCSV(), mgr.GetEventRecorderFor(hcoutil.HyperConvergedName))

	err = mgr.AddHealthzCheck("ping", healthz.Ping)
	cmdHelper.ExitOnError(err, "unable to add health check")

	readyCheck := hcoutil.GetHcoPing()

	err = mgr.AddReadyzCheck("ready", readyCheck)
	cmdHelper.ExitOnError(err, "unable to add ready check")

	// Force OperatorCondition Upgradeable to False
	//
	// We have to at least default the condition to False or
	// OLM will use the Readiness condition via our readiness probe instead:
	// https://olm.operatorframework.io/docs/advanced-tasks/communicating-operator-conditions-to-olm/#setting-defaults
	//
	// We want to force it to False to ensure that the final decision about whether
	// the operator can be upgraded stays within the hyperconverged controller.
	logger.Info("Setting OperatorCondition.")
	upgradeableCondition, err := hcoutil.NewOperatorCondition(ci, apiClient, operatorsapiv2.Upgradeable)
	cmdHelper.ExitOnError(err, "Cannot create the Upgradeable Operator Condition")

	err = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		err := upgradeableCondition.Set(ctx, metav1.ConditionFalse, hcoutil.UpgradeableInitReason, hcoutil.UpgradeableInitMessage)
		if err != nil {
			logger.Error(err, "Cannot set the status of the Upgradeable Operator Condition; "+err.Error())
		}
		return err == nil, nil
	})
	cmdHelper.ExitOnError(err, "Failed to set the status of the Upgradeable Operator Condition")

	if err = upgradepatch.Init(logger); err != nil {
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Failed validating upgrade patches file")
		cmdHelper.ExitOnError(err, "Failed validating upgrade patches file")
	}

	// re-create the condition, this time with the final client
	upgradeableCondition, err = hcoutil.NewOperatorCondition(ci, mgr.GetClient(), operatorsapiv2.Upgradeable)
	cmdHelper.ExitOnError(err, "Cannot create Upgradeable Operator Condition")

	ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 10)
	defer close(ingressEventCh)

	// Create a new reconciler
	if err := hyperconverged.RegisterReconciler(mgr, ci, upgradeableCondition, ingressEventCh); err != nil {
		logger.Error(err, "failed to register the HyperConverged controller")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register HyperConverged controller; "+err.Error())
		os.Exit(1)
	}

	// a channel to trigger a restart of the operator
	// via a clean cancel of the manager
	restartCh := make(chan struct{})
	defer close(restartCh)

	// Create a new CRD reconciler
	if err := crd.RegisterReconciler(mgr, restartCh); err != nil {
		logger.Error(err, "failed to register the CRD controller")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register CRD controller; "+err.Error())
		os.Exit(1)
	}

	if ci.IsOpenshift() {
		if err = observability.SetupWithManager(mgr, ci.GetDeployment()); err != nil {
			logger.Error(err, "unable to create controller", "controller", "Observability")
			os.Exit(1)
		}
	}

	if ci.IsDeschedulerAvailable() {
		// Create a new reconciler for KubeDescheduler
		if err := descheduler.RegisterReconciler(mgr); err != nil {
			logger.Error(err, "failed to register the KubeDescheduler controller")
			eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register KubeDescheduler controller; "+err.Error())
			os.Exit(1)
		}
	}

	// Create a new Nodes reconciler
	if err := nodes.RegisterReconciler(mgr); err != nil {
		logger.Error(err, "failed to register the Nodes controller")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register Nodes controller; "+err.Error())
		os.Exit(1)
	}

	if ci.IsOpenshift() {
		if err = ingresscluster.RegisterReconciler(mgr, ingressEventCh); err != nil {
			logger.Error(err, "failed to register the IngressCluster controller")
			eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register Ingress controller; "+err.Error())
			os.Exit(1)
		}
	}

	err = createPriorityClass(ctx, mgr)
	cmdHelper.ExitOnError(err, "Failed creating PriorityClass")

	// Setup Monitoring
	operatormetrics.Register = controllerruntimemetrics.Registry.Register
	err = metrics.SetupMetrics()
	cmdHelper.ExitOnError(err, "failed to setup metrics: %v")

	logger.Info("Starting the Cmd.")
	eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Init", "Starting the HyperConverged Pod")

	// create context with cancel for the manager
	mgrCtx, mgrCancel := context.WithCancel(signals.SetupSignalHandler())

	defer mgrCancel()
	go func() {
		<-restartCh
		mgrCancel()
	}()

	// Start the Cmd
	if err := mgr.Start(mgrCtx); err != nil {
		logger.Error(err, "Manager exited non-zero")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", "HyperConverged crashed; "+err.Error())
		os.Exit(1)
	}
}

// Restricts the cache's ListWatch to specific fields/labels per GVK at the specified object to control the memory impact
// this is used to completely overwrite the NewCache function so all the interesting objects should be explicitly listed here
func getCacheOption(operatorNamespace string, ci hcoutil.ClusterInfo) cache.Options {
	namespaceSelector := fields.Set{"metadata.namespace": operatorNamespace}.AsSelector()
	labelSelector := labels.Set{hcoutil.AppLabel: hcoutil.HyperConvergedName}.AsSelector()
	labelSelectorForNamespace := labels.Set{hcoutil.KubernetesMetadataName: operatorNamespace}.AsSelector()

	cacheOptions := cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&hcov1beta1.HyperConverged{}:           {},
			&kubevirtcorev1.KubeVirt{}:             {},
			&cdiv1beta1.CDI{}:                      {},
			&networkaddonsv1.NetworkAddonsConfig{}: {},
			&sspv1beta2.SSP{}:                      {},
			&schedulingv1.PriorityClass{}: {
				Label: labels.SelectorFromSet(labels.Set{hcoutil.AppLabel: hcoutil.HyperConvergedName}),
			},
			&corev1.ConfigMap{}: {
				Label: labelSelector,
			},
			&corev1.Service{}: {
				Field: namespaceSelector,
			},
			&corev1.Endpoints{}: {
				Field: namespaceSelector,
			},
			&rbacv1.Role{}: {
				Label: labelSelector,
				Field: namespaceSelector,
			},
			&rbacv1.RoleBinding{}: {
				Label: labelSelector,
				Field: namespaceSelector,
			},
			&corev1.Namespace{}: {
				Label: labelSelectorForNamespace,
			},
			&appsv1.Deployment{}: {
				Label: labelSelector,
				Field: namespaceSelector,
			},
			&apiextensionsv1.CustomResourceDefinition{}: {},
		},
	}

	cacheOptionsByObjectForMonitoring := map[client.Object]cache.ByObject{
		&monitoringv1.ServiceMonitor{}: {
			Label: labelSelector,
			Field: namespaceSelector,
		},
		&monitoringv1.PrometheusRule{}: {
			Label: labelSelector,
			Field: namespaceSelector,
		},
	}

	cacheOptionsByObjectForDescheduler := map[client.Object]cache.ByObject{
		&deschedulerv1.KubeDescheduler{}: {},
	}

	cacheOptionsByObjectForOpenshift := map[client.Object]cache.ByObject{
		&openshiftroutev1.Route{}: {
			Namespaces: map[string]cache.Config{
				operatorNamespace:            {},
				openshiftMonitoringNamespace: {},
			},
		},
		&imagev1.ImageStream{}: {
			Label: labelSelector,
		},
		&openshiftconfigv1.APIServer{}: {},
		&consolev1.ConsoleCLIDownload{}: {
			Label: labelSelector,
		},
		&consolev1.ConsoleQuickStart{}: {
			Label: labelSelector,
		},
		&consolev1.ConsolePlugin{}: {
			Label: labelSelector,
		},
	}

	if ci.IsMonitoringAvailable() {
		maps.Copy(cacheOptions.ByObject, cacheOptionsByObjectForMonitoring)
	}
	if ci.IsDeschedulerAvailable() {
		maps.Copy(cacheOptions.ByObject, cacheOptionsByObjectForDescheduler)
	}
	if ci.IsOpenshift() {
		maps.Copy(cacheOptions.ByObject, cacheOptionsByObjectForOpenshift)
	}

	return cacheOptions

}

func getManagerOptions(operatorNamespace string, needLeaderElection bool, ci hcoutil.ClusterInfo, scheme *apiruntime.Scheme) manager.Options {
	return manager.Options{
		Metrics: server.Options{
			BindAddress:    fmt.Sprintf("%s:%d", hcoutil.MetricsHost, hcoutil.MetricsPort),
			FilterProvider: authorization.HttpWithBearerToken,
		},
		HealthProbeBindAddress: fmt.Sprintf("%s:%d", hcoutil.HealthProbeHost, hcoutil.HealthProbePort),
		ReadinessEndpointName:  hcoutil.ReadinessEndpointName,
		LivenessEndpointName:   hcoutil.LivenessEndpointName,
		LeaderElection:         needLeaderElection,
		// We set ConfigMapsLeasesResourceLock already in release-1.5 to migrate from configmaps to leases.
		// Since we used "configmapsleases" for over two years, spanning five minor releases,
		// any actively maintained operators are very likely to have a released version that uses
		// "configmapsleases". Therefore, having only "leases" should be safe now.
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaderElectionID:           "hyperconverged-cluster-operator-lock",
		Cache:                      getCacheOption(operatorNamespace, ci),
		Scheme:                     scheme,
	}
}

// KubeVirtPriorityClass is needed by virt-operator but OLM is not able to
// create it so we have to create it ASAP.
// When the user deletes HCO CR virt-operator should continue running
// so we are never supposed to delete it: because the priority class
// is completely opaque to OLM it will remain as a leftover on the cluster
func createPriorityClass(ctx context.Context, mgr manager.Manager) error {
	pc := operands.NewKubeVirtPriorityClass(&hcov1beta1.HyperConverged{})

	err := mgr.GetAPIReader().Get(ctx, client.ObjectKeyFromObject(pc), pc)
	if err != nil && apierrors.IsNotFound(err) {
		logger.Info("Creating KubeVirt PriorityClass")
		return mgr.GetClient().Create(ctx, pc, &client.CreateOptions{})
	}

	return err
}
