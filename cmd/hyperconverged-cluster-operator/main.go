package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kubevirt/hyperconverged-cluster-operator/cmd/cmdcommon"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/hyperconverged"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Change below variables to serve metrics on different host or port.
var (
	logger               = logf.Log.WithName("hyperconverged-operator-cmd")
	cmdHelper            = cmdcommon.NewHelper(logger, "operator")
	resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
		apis.AddToScheme,
		cdiv1beta1.AddToScheme,
		networkaddons.AddToScheme,
		sspv1beta1.AddToScheme,
		csvv1alpha1.AddToScheme,
		vmimportv1beta1.AddToScheme,
		admissionregistrationv1.AddToScheme,
		consolev1.AddToScheme,
		openshiftconfigv1.AddToScheme,
		monitoringv1.AddToScheme,
		consolev1.AddToScheme,
		apiextensionsv1.AddToScheme,
	}
)

func main() {
	cmdHelper.InitiateCommand()

	watchNamespace := cmdHelper.GetWatchNS()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	cmdHelper.ExitOnError(err, "can't load configuration")

	// a lock is not needed in webhook mode
	// TODO: remove this once we will move to OLM operator conditions
	needLeaderElection := !cmdHelper.IsRunInLocal()

	// Create a new Cmd to provide shared dependencies and start components
	// TODO: consider changing LeaderElectionResourceLock to new default "configmapsleases".
	mgr, err := manager.New(cfg, getManagerOptions(watchNamespace, needLeaderElection))
	cmdHelper.ExitOnError(err, "can't initiate manager")

	// register pprof instrumentation if HCO_PPROF_ADDR is set
	cmdHelper.ExitOnError(cmdHelper.RegisterPPROFServer(mgr), "can't register pprof server")

	logger.Info("Registering Components.")

	// Setup Scheme for all resources
	cmdHelper.AddToScheme(mgr, resourcesSchemeFuncs)

	// Detect OpenShift version
	ctx := context.TODO()
	ci := hcoutil.GetClusterInfo()
	err = ci.CheckRunningInOpenshift(mgr.GetAPIReader(), ctx, logger, cmdHelper.IsRunInLocal())
	cmdHelper.ExitOnError(err, "Cannot detect cluster type")

	eventEmitter := hcoutil.GetEventEmitter()
	// Set temporary configuration, until the regular client is ready
	eventEmitter.Init(ctx, mgr, ci, logger)

	err = mgr.AddHealthzCheck("ping", healthz.Ping)
	cmdHelper.ExitOnError(err, "unable to add health check")

	readyCheck := hcoutil.GetHcoPing()

	err = mgr.AddReadyzCheck("ready", readyCheck)
	cmdHelper.ExitOnError(err, "unable to add ready check")

	// Setup all Controllers
	if err := hyperconverged.RegisterReconciler(mgr, ci); err != nil {
		logger.Error(err, "")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register HyperConverged controller; "+err.Error())
		os.Exit(1)
	}

	err = createPriorityClass(ctx, mgr)
	cmdHelper.ExitOnError(err, "Failed creating PriorityClass")

	logger.Info("Starting the Cmd.")
	eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Init", "Starting the HyperConverged Pod")
	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Error(err, "Manager exited non-zero")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", "HyperConverged crashed; "+err.Error())
		os.Exit(1)
	}
}

func getManagerOptions(watchNamespace string, needLeaderElection bool) manager.Options {
	return manager.Options{
		Namespace:                  watchNamespace,
		MetricsBindAddress:         fmt.Sprintf("%s:%d", hcoutil.MetricsHost, hcoutil.MetricsPort),
		HealthProbeBindAddress:     fmt.Sprintf("%s:%d", hcoutil.HealthProbeHost, hcoutil.HealthProbePort),
		ReadinessEndpointName:      hcoutil.ReadinessEndpointName,
		LivenessEndpointName:       hcoutil.LivenessEndpointName,
		LeaderElection:             needLeaderElection,
		LeaderElectionResourceLock: "configmaps",
		LeaderElectionID:           "hyperconverged-cluster-operator-lock",
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
