package main

import (
	"flag"
	"runtime"

	"github.com/golang/glog"

	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/consts"
	"kubevirt.io/machine-remediation-operator/pkg/controllers"
	"kubevirt.io/machine-remediation-operator/pkg/operator"
	"kubevirt.io/machine-remediation-operator/pkg/version"

	"sigs.k8s.io/controller-runtime/pkg/cache"
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
	namespace := flag.String("namespace", "", "Namespace that the controller watches to reconcile objects. If unspecified, the controller watches for machine-remediation-operator objects across all namespaces.")
	flag.Parse()

	printVersion()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}

	namespaces := []string{consts.NamespaceOpenshiftMachineAPI, metav1.NamespaceNone}
	if *namespace != "" {
		namespaces = append(namespaces, *namespace)
	}
	opts := manager.Options{
		NewCache: cache.MultiNamespacedCacheBuilder(namespaces),
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, opts)
	if err != nil {
		glog.Fatal(err)
	}

	glog.Infof("Registering Components.")

	// Setup Scheme for all resources
	if err := extv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	if err := mrv1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	// Setup all Controllers
	if err := controllers.AddToManager(mgr, opts, operator.Add); err != nil {
		glog.Fatal(err)
	}

	glog.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Fatal(err)
	}
}
