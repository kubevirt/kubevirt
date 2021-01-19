package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
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

	// Get the namespace the operator is currently deployed in.
	depOperatorNs, err := hcoutil.GetOperatorNamespace(log)
	runInLocal := false
	if err != nil {
		if err == hcoutil.ErrRunLocal {
			runInLocal = true
		} else {
			log.Error(err, "Failed to get operator namespace")
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

	// Get the namespace the operator should be deployed in.
	operatorNsEnv, err := hcoutil.GetOperatorNamespaceFromEnv()
	if err != nil {
		log.Error(err, "Failed to get operator namespace from the environment")
		os.Exit(1)
	}

	if runInLocal {
		depOperatorNs = operatorNsEnv
	}

	if depOperatorNs != operatorNsEnv {
		log.Error(
			fmt.Errorf("operator running in different namespace than expected"),
			fmt.Sprintf("Please re-deploy this operator into %v namespace", operatorNsEnv),
			"Expected.Namespace", operatorNsEnv,
			"Deployed.Namespace", depOperatorNs,
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

	// a lock is not needed in webhook mode
	// TODO: remove this once we will move to OLM operator conditions
	needLeaderElection := !runInLocal

	// Create a new Cmd to provide shared dependencies and start components
	// TODO: consider changing LeaderElectionResourceLock to new default "configmapsleases".
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:                  watchNamespace,
		MetricsBindAddress:         fmt.Sprintf("%s:%d", hcoutil.MetricsHost, hcoutil.MetricsPort),
		HealthProbeBindAddress:     fmt.Sprintf("%s:%d", hcoutil.HealthProbeHost, hcoutil.HealthProbePort),
		ReadinessEndpointName:      hcoutil.ReadinessEndpointName,
		LivenessEndpointName:       hcoutil.LivenessEndpointName,
		LeaderElection:             needLeaderElection,
		LeaderElectionResourceLock: "configmaps",
		LeaderElectionID:           "hyperconverged-cluster-operator-lock",
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
		sspv1beta1.AddToScheme,
		csvv1alpha1.AddToScheme,
		vmimportv1beta1.AddToScheme,
		admissionregistrationv1.AddToScheme,
		consolev1.AddToScheme,
		openshiftconfigv1.AddToScheme,
		monitoringv1.AddToScheme,
		consolev1.AddToScheme,
		apiextensionsv1.AddToScheme,
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

	readyCheck := hcoutil.GetHcoPing()

	if err := mgr.AddReadyzCheck("ready", readyCheck); err != nil {
		log.Error(err, "unable to add ready check")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr, ci); err != nil {
		log.Error(err, "")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register component; "+err.Error())
		os.Exit(1)
	}

	err = createPriorityClass(ctx, mgr)
	if err != nil {
		log.Error(err, "Failed creating PriorityClass")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")
	eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Init", "Starting the HyperConverged Pod")
	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", "HyperConverged crashed; "+err.Error())
		os.Exit(1)
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
		log.Info("Creating KubeVirt PriorityClass")
		return mgr.GetClient().Create(ctx, pc, &client.CreateOptions{})
	}

	return err
}
