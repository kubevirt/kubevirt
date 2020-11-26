package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/webhooks"
	"github.com/spf13/pflag"
	"os"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	sspopv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Change below variables to serve metrics on different host or port.
var (
	log = logf.Log.WithName("cmd")
)

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	zapfs := flag.NewFlagSet("zap", flag.ExitOnError)
	zopts := &zap.Options{}
	zopts.BindFlags(zapfs)
	pflag.CommandLine.AddGoFlagSet(zapfs)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	logf.SetLogger(zap.New(zap.UseFlagOptions(zopts)))

	printVersion()

	// Get the namespace the webhook is currently deployed in.
	depWebhookNs, err := hcoutil.GetOperatorNamespace(log)
	runInLocal := false
	if err != nil {
		if err == hcoutil.ErrRunLocal {
			runInLocal = true
		} else {
			log.Error(err, "Failed to get webhook namespace")
			os.Exit(1)
		}
	}

	if runInLocal {
		log.Info("running locally")
	}

	watchNamespace := ""

	if !runInLocal {
		watchNamespace, err = hcoutil.GetWatchNamespace()
		if err != nil {
			log.Error(err, "Failed to get watch namespace")
			os.Exit(1)
		}
	}

	// Get the namespace the webhook should be deployed in.
	webhookNsEnv, err := hcoutil.GetOperatorNamespaceFromEnv()
	if err != nil {
		log.Error(err, "Failed to get webhook namespace from the environment")
		os.Exit(1)
	}

	if runInLocal {
		depWebhookNs = webhookNsEnv
	}

	if depWebhookNs != webhookNsEnv {
		log.Error(
			fmt.Errorf("webhook running in different namespace than expected"),
			fmt.Sprintf("Please re-deploy this webhook into %v namespace", webhookNsEnv),
			"Expected.Namespace", webhookNsEnv,
			"Deployed.Namespace", depWebhookNs,
		)
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:              watchNamespace,
		MetricsBindAddress:     fmt.Sprintf("%s:%d", hcoutil.MetricsHost, hcoutil.MetricsPort),
		HealthProbeBindAddress: fmt.Sprintf("%s:%d", hcoutil.HealthProbeHost, hcoutil.HealthProbePort),
		ReadinessEndpointName:  hcoutil.ReadinessEndpointName,
		LivenessEndpointName:   hcoutil.LivenessEndpointName,
		LeaderElection:         false,
		LeaderElectionID:       "hyperconverged-cluster-webhook-lock",
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	for _, f := range []func(*apiruntime.Scheme) error{
		apis.AddToScheme,
		cdiv1beta1.AddToScheme,
		networkaddons.AddToScheme,
		sspopv1.AddToScheme,
		csvv1alpha1.AddToScheme,
		vmimportv1beta1.AddToScheme,
		admissionregistrationv1.AddToScheme,
		consolev1.AddToScheme,
		openshiftconfigv1.AddToScheme,
	} {
		if err := f(mgr.GetScheme()); err != nil {
			log.Error(err, "Failed to add to scheme")
			os.Exit(1)
		}
	}

	// Detect OpenShift version
	ci := hcoutil.GetClusterInfo()
	err = ci.CheckRunningInOpenshift(mgr.GetAPIReader(), ctx, log, runInLocal)
	if err != nil {
		log.Error(err, "Cannot detect cluster type")
		os.Exit(1)
	}

	eventEmitter := hcoutil.GetEventEmitter()
	// Set temporary configuration, until the regular client is ready
	eventEmitter.Init(ctx, mgr, ci, log)

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		log.Error(err, "unable to add health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("ready", healthz.Ping); err != nil {
		log.Error(err, "unable to add ready check")
		os.Exit(1)
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	hwHandler := &webhooks.WebhookHandler{}
	if err = (&hcov1beta1.HyperConverged{}).SetupWebhookWithManager(ctx, mgr, hwHandler); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "HyperConverged")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to create webhook")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")
	eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Init", "Starting the HyperConverged webhook Pod")
	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", "HyperConverged crashed; "+err.Error())
		os.Exit(1)
	}
}
