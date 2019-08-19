package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorstatus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/operators/catalog"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/signals"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/metrics"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/version"
)

const (
	defaultWakeupInterval       = 15 * time.Minute
	defaultCatalogNamespace     = "openshift-operator-lifecycle-manager"
	defaultConfigMapServerImage = "quay.io/operatorframework/configmap-operator-registry:latest"
	defaultOperatorName         = ""
)

// config flags defined globally so that they appear on the test binary as well
var (
	kubeConfigPath = flag.String(
		"kubeconfig", "", "absolute path to the kubeconfig file")

	wakeupInterval = flag.Duration(
		"interval", defaultWakeupInterval, "wakeup interval")

	watchedNamespaces = flag.String(
		"watchedNamespaces", "", "comma separated list of namespaces that catalog watches, leave empty to watch all namespaces")

	catalogNamespace = flag.String(
		"namespace", defaultCatalogNamespace, "namespace where catalog will run and install catalog resources")

	configmapServerImage = flag.String(
		"configmapServerImage", defaultConfigMapServerImage, "the image to use for serving the operator registry api for a configmap")

	writeStatusName = flag.String(
		"writeStatusName", defaultOperatorName, "ClusterOperator name in which to write status, set to \"\" to disable.")

	debug = flag.Bool(
		"debug", false, "use debug log level")

	version = flag.Bool("version", false, "displays olm version")

	tlsKeyPath = flag.String(
		"tls-key", "", "Path to use for private key (requires tls-cert)")

	tlsCertPath = flag.String(
		"tls-cert", "", "Path to use for certificate key (requires tls-key)")
)

func init() {
	metrics.RegisterCatalog()
}

func main() {
	stopCh := signals.SetupSignalHandler()

	// Parse the command-line flags.
	flag.Parse()

	// Check if version flag was set
	if *version {
		fmt.Print(olmversion.String())

		// Exit early
		os.Exit(0)
	}

	// `namespaces` will always contain at least one entry: if `*watchedNamespaces` is
	// the empty string, the resulting array will be `[]string{""}`.
	namespaces := strings.Split(*watchedNamespaces, ",")
	for _, ns := range namespaces {
		if ns == v1.NamespaceAll {
			namespaces = []string{v1.NamespaceAll}
			break
		}
	}

	logger := log.New()
	if *debug {
		logger.SetLevel(log.DebugLevel)
	}
	logger.Infof("log level %s", logger.Level)

	var useTLS bool
	if *tlsCertPath != "" && *tlsKeyPath == "" || *tlsCertPath == "" && *tlsKeyPath != "" {
		logger.Warn("both --tls-key and --tls-crt must be provided for TLS to be enabled, falling back to non-https")
	} else if *tlsCertPath == "" && *tlsKeyPath == "" {
		logger.Info("TLS keys not set, using non-https")
	} else {
		useTLS = true
	}

	// Serve a health check.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	http.Handle("/metrics", promhttp.Handler())
	if useTLS {
		go http.ListenAndServeTLS(":8081", *tlsCertPath, *tlsKeyPath, nil)
	} else {
		go http.ListenAndServe(":8081", nil)
	}

	// create a config client for operator status
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		log.Fatalf("error configuring client: %s", err.Error())
	}
	configClient, err := configv1client.NewForConfig(config)
	if err != nil {
		log.Fatalf("error configuring client: %s", err.Error())
	}
	opClient := operatorclient.NewClientFromConfig(*kubeConfigPath, logger)

	// Create a new instance of the operator.
	catalogOperator, err := catalog.NewOperator(*kubeConfigPath, logger, *wakeupInterval, *configmapServerImage, *catalogNamespace, namespaces...)
	if err != nil {
		log.Panicf("error configuring operator: %s", err.Error())
	}

	ready, done, sync := catalogOperator.Run(stopCh)
	<-ready

	if *writeStatusName != "" {
		operatorstatus.MonitorClusterStatus(*writeStatusName, sync, stopCh, opClient, configClient)
	}

	<-done
}
