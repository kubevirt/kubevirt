package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"os"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
	log                       = logf.Log.WithName("cmd")
)

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	// Get the namespace the operator is currently deployed in.
	depOperatorNs, err := k8sutil.GetOperatorNamespace()
	runInLocal := false
	if err != nil {
		if err == k8sutil.ErrRunLocal {
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
		watchNamespace, err = k8sutil.GetWatchNamespace()
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

	// TODO: remove this once we will move to OLM operator conditions
	// Get the webhook mode
	operatorWebhookMode, err := hcoutil.GetWebhookModeFromEnv()
	if err != nil {
		log.Error(err, "Failed to get operator webhook mode from the environment")
		os.Exit(1)
	}
	if operatorWebhookMode {
		log.Info("operatorWebhookMode: running only the validating webhook")
	} else {
		log.Info("operatorWebhookMode: running only the real operator")
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

	// Create a new file supporting readiness probe
	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	defer r.Unset()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()

	// a lock is not needed in webhook mode
	// TODO: remove this once we will move to OLM operator conditions
	if !operatorWebhookMode {
		// Become the leader before proceeding
		err = leader.Become(ctx, "hyperconverged-cluster-operator-lock")
		if err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          watchNamespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
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
	if err != nil {
		log.Error(err, "failed to initiate event emitter")
		os.Exit(1)
	}

	// TODO: remove this once we will move to OLM operator conditions
	if !operatorWebhookMode {
		// Setup all Controllers
		if err := controller.AddToManager(mgr, ci); err != nil {
			log.Error(err, "")
			eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to register component; "+err.Error())
			os.Exit(1)
		}
	}

	if !runInLocal {
		if err = serveCRMetrics(cfg); err != nil {
			log.Error(err, "Could not generate and serve custom resource metrics")
		}

		// Add to the below struct any other metrics ports you want to expose.
		servicePorts := []corev1.ServicePort{
			{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
			{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
		}

		// Create Service object to expose the metrics port(s).
		service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
		if err != nil {
			log.Error(err, "Could not create metrics Service")
		}
		services := []*corev1.Service{service}
		_, err = metrics.CreateServiceMonitors(cfg, depOperatorNs, services)
		if err != nil {
			log.Error(err, "Could not create ServiceMonitor object")
			// If this operator is deployed to a cluster without the prometheus-operator running, it will return
			// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
			if err == metrics.ErrServiceMonitorNotPresent {
				log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects")
			}
		}
	}

	// TODO: remove this once we will move to OLM operator conditions
	if operatorWebhookMode {
		// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
		// necessary to configure Prometheus to scrape metrics from this operator.
		if err = (&hcov1beta1.HyperConverged{}).SetupWebhookWithManager(ctx, mgr); err != nil {
			log.Error(err, "unable to create webhook", "webhook", "HyperConverged")
			eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "InitError", "Unable to create webhook")
			os.Exit(1)
		}
	} else {
		err = createPriorityClass(ctx, mgr)
		if err != nil {
			log.Error(err, "Failed creating PriorityClass")
			os.Exit(1)
		}
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
	pc := (&hcov1beta1.HyperConverged{}).NewKubeVirtPriorityClass()

	key, err := client.ObjectKeyFromObject(pc)
	if err != nil {
		log.Error(err, "Failed to get object key for KubeVirt PriorityClass")
		return err
	}

	err = mgr.GetAPIReader().Get(ctx, key, pc)

	if err != nil && apierrors.IsNotFound(err) {
		log.Info("Creating KubeVirt PriorityClass")
		return mgr.GetClient().Create(ctx, pc, &client.CreateOptions{})
	}

	return err
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
