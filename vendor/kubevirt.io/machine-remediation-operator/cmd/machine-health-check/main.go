package main

import (
	"flag"
	"runtime"

	"github.com/golang/glog"
	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/controllers"
	"kubevirt.io/machine-remediation-operator/pkg/controllers/machinehealthcheck"
	"kubevirt.io/machine-remediation-operator/pkg/version"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func printVersion() {
	glog.Infof("Go Version: %s", runtime.Version())
	glog.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	glog.Infof("Component version: %s", version.Get())
}

func main() {
	namespace := flag.String("namespace", "", "Namespace that the controller watches to reconcile objects. If unspecified, the controller watches for machine-health-check objects across all namespaces.")
	flag.Parse()

	printVersion()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}

	opts := manager.Options{
		LeaderElection:   true,
		LeaderElectionID: "machine-health-check",
	}
	if *namespace != "" {
		opts.LeaderElectionNamespace = *namespace
		opts.Namespace = *namespace
		glog.Infof("Watching MHC objects only in namespace %q for reconciliation.", opts.Namespace)
	}
	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, opts)
	if err != nil {
		glog.Fatal(err)
	}

	glog.Infof("Registering Components.")

	// Setup Scheme for all resources
	if err := mrv1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}
	if err := mapiv1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	// Setup all Controllers
	if err := controllers.AddToManager(mgr, opts, machinehealthcheck.Add); err != nil {
		glog.Fatal(err)
	}

	glog.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Fatal(err)
	}
}
