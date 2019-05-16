package main

import (
	"flag"
	"net/http"
	"os"
	"runtime"
	"time"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/signals"
	"github.com/operator-framework/operator-marketplace/pkg/apis"
	"github.com/operator-framework/operator-marketplace/pkg/catalogsourceconfig"
	"github.com/operator-framework/operator-marketplace/pkg/controller"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/status"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// TODO: resyncInterval is hardcoded to 1 hour now, it would have to be
	// configurable on a per OperatorSource level.
	resyncInterval = time.Duration(60) * time.Minute

	initialWait                = time.Duration(1) * time.Minute
	updateNotificationSendWait = time.Duration(10) * time.Minute
)

func printVersion() {
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	// Parse the command line arguments for the registry server image
	flag.StringVar(&catalogsourceconfig.RegistryServerImage, "registryServerImage",
		catalogsourceconfig.DefaultRegistryServerImage, "the image to use for creating the operator registry pod")
	flag.Parse()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Fatalf("failed to get watch namespace: %v", err)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Registering Components.")

	catalogsourceconfig.InitializeStaticSyncer(mgr.GetClient(), initialWait)
	registrySyncer := operatorsource.NewRegistrySyncer(mgr.GetClient(), initialWait, resyncInterval, updateNotificationSendWait, catalogsourceconfig.Syncer, catalogsourceconfig.Syncer)

	// Setup Scheme for all defined resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		exit(err)
	}

	// Add external resource to scheme
	if err := olm.AddToScheme(mgr.GetScheme()); err != nil {
		exit(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		exit(err)
	}

	// Serve a health check.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	log.Print("Starting the Cmd.")
	stopCh := signals.SetupSignalHandler()

	// statusReportingDoneCh will be closed after the operator has successfully stopped reporting ClusterOperator status.
	statusReportingDoneCh := status.StartReporting(cfg, mgr, namespace, os.Getenv("RELEASE_VERSION"), stopCh)

	go registrySyncer.Sync(stopCh)
	go catalogsourceconfig.Syncer.Sync(stopCh)

	// Start the Cmd
	err = mgr.Start(stopCh)

	// Wait for ClusterOperator status reporting routine to close the statusReportingDoneCh channel.
	<-statusReportingDoneCh

	exit(err)
}

// exit stops the reporting of ClusterOperator status and exits with the proper exit code.
func exit(err error) {
	// If an error exists then exit with status set to 1
	if err != nil {
		log.Fatalf("The operator encountered an error, exit code 1: %v", err)
	}

	// No error, graceful termination
	log.Info("The operator exited gracefully, exit code 0")
	os.Exit(0)
}
