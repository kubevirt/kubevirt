package main

import (
	"context"
	"fmt"
	"os"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/api"
	"github.com/kubevirt/hyperconverged-cluster-operator/cmd/cmdcommon"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/webhooks"
	ttov1alpha1 "github.com/kubevirt/tekton-tasks-operator/api/v1alpha1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
)

// Change below variables to serve metrics on different host or port.
var (
	logger               = logf.Log.WithName("hyperconverged-webhook-cmd")
	cmdHelper            = cmdcommon.NewHelper(logger, "webhook")
	resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
		api.AddToScheme,
		corev1.AddToScheme,
		cdiv1beta1.AddToScheme,
		networkaddonsv1.AddToScheme,
		sspv1beta1.AddToScheme,
		ttov1alpha1.AddToScheme,
		admissionregistrationv1.AddToScheme,
		openshiftconfigv1.Install,
		kubevirtcorev1.AddToScheme,
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

	// apiclient.New() returns a client without cache.
	// cache is not initialized before mgr.Start()
	// we need this because we need to interact with OperatorCondition
	apiClient, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	cmdHelper.ExitOnError(err, "Cannot create a new API client")

	// register pprof instrumentation if HCO_PPROF_ADDR is set
	cmdHelper.ExitOnError(cmdHelper.RegisterPPROFServer(mgr), "can't register pprof server")

	logger.Info("Registering Components.")

	// Detect OpenShift version
	ci := hcoutil.GetClusterInfo()
	ctx := context.TODO()
	err = ci.Init(ctx, apiClient, logger)
	cmdHelper.ExitOnError(err, "Cannot detect cluster type")

	eventEmitter := hcoutil.GetEventEmitter()
	eventEmitter.Init(ci.GetPod(), ci.GetCSV(), mgr.GetEventRecorderFor(hcoutil.HyperConvergedName))

	err = mgr.AddHealthzCheck("ping", healthz.Ping)
	cmdHelper.ExitOnError(err, "unable to add health check")

	err = mgr.AddReadyzCheck("ready", healthz.Ping)
	cmdHelper.ExitOnError(err, "unable to add ready check")

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	if err = webhooks.SetupWebhookWithManager(ctx, mgr, ci.IsOpenshift()); err != nil {
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
