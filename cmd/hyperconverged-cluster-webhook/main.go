package main

import (
	"context"
	"fmt"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"os"

	"github.com/kubevirt/hyperconverged-cluster-operator/cmd/cmdcommon"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/webhooks"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Change below variables to serve metrics on different host or port.
var (
	logger               = logf.Log.WithName("hyperconverged-webhook-cmd")
	cmdHelper            = cmdcommon.NewHelper(logger, "webhook")
	resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
		apis.AddToScheme,
		corev1.AddToScheme,
		cdiv1beta1.AddToScheme,
		networkaddons.AddToScheme,
		sspv1beta1.AddToScheme,
		vmimportv1beta1.AddToScheme,
		admissionregistrationv1.AddToScheme,
		openshiftconfigv1.AddToScheme,
		kubevirtv1.AddToScheme,
	}
)

func main() {

	cmdHelper.InitiateCommand()

	watchNamespace := cmdHelper.GetWatchNS()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error(err, "")
		os.Exit(1)
	}

	// Setup Scheme for all resources
	scheme := apiruntime.NewScheme()
	cmdHelper.AddToScheme(scheme, resourcesSchemeFuncs)

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:              watchNamespace,
		MetricsBindAddress:     fmt.Sprintf("%s:%d", hcoutil.MetricsHost, hcoutil.MetricsPort),
		HealthProbeBindAddress: fmt.Sprintf("%s:%d", hcoutil.HealthProbeHost, hcoutil.HealthProbePort),
		ReadinessEndpointName:  hcoutil.ReadinessEndpointName,
		LivenessEndpointName:   hcoutil.LivenessEndpointName,
		LeaderElection:         false,
		Scheme:                 scheme,
	})
	cmdHelper.ExitOnError(err, "failed to create manager")

	// register pprof instrumentation if HCO_PPROF_ADDR is set
	cmdHelper.ExitOnError(cmdHelper.RegisterPPROFServer(mgr), "can't register pprof server")

	logger.Info("Registering Components.")

	// Detect OpenShift version
	ci := hcoutil.GetClusterInfo()
	ctx := context.TODO()
	err = ci.CheckRunningInOpenshift(mgr.GetAPIReader(), ctx, logger, cmdHelper.IsRunInLocal())
	cmdHelper.ExitOnError(err, "Cannot detect cluster type")

	eventEmitter := hcoutil.GetEventEmitter()
	// Set temporary configuration, until the regular client is ready
	eventEmitter.Init(ctx, mgr, ci, logger)

	err = mgr.AddHealthzCheck("ping", healthz.Ping)
	cmdHelper.ExitOnError(err, "unable to add health check")

	err = mgr.AddReadyzCheck("ready", healthz.Ping)
	cmdHelper.ExitOnError(err, "unable to add ready check")

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	hwHandler := &webhooks.WebhookHandler{}
	if err = (&hcov1beta1.HyperConverged{}).SetupWebhookWithManager(ctx, mgr, hwHandler, ci.IsOpenshift()); err != nil {
		logger.Error(err, "unable to create webhook", "webhook", "HyperConverged")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to create webhook")
		os.Exit(1)
	}

	logger.Info("Starting the Cmd.")
	eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Init", "Starting the HyperConverged webhook Pod")
	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Error(err, "Manager exited non-zero")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", "HyperConverged crashed; "+err.Error())
		os.Exit(1)
	}
}
