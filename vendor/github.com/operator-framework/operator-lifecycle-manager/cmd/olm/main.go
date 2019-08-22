package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	configclientset "github.com/openshift/client-go/config/clientset/versioned"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/operators/olm"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorstatus"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/profile"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/signals"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/metrics"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/version"
)

const (
	defaultWakeupInterval          = 5 * time.Minute
	defaultOperatorName            = ""
	defaultPackageServerStatusName = ""
)

// config flags defined globally so that they appear on the test binary as well
var (
	kubeConfigPath = flag.String(
		"kubeconfig", "", "absolute path to the kubeconfig file")

	wakeupInterval = flag.Duration(
		"interval", defaultWakeupInterval, "wake up interval")

	watchedNamespaces = flag.String(
		"watchedNamespaces", "", "comma separated list of namespaces for olm operator to watch. "+
			"If not set, or set to the empty string (e.g. `-watchedNamespaces=\"\"`), "+
			"olm operator will watch all namespaces in the cluster.")

	writeStatusName = flag.String(
		"writeStatusName", defaultOperatorName, "ClusterOperator name in which to write status, set to \"\" to disable.")

	writePackageServerStatusName = flag.String(
		"writePackageServerStatusName", defaultPackageServerStatusName, "ClusterOperator name in which to write status for package API server, set to \"\" to disable.")

	debug = flag.Bool(
		"debug", false, "use debug log level")

	version = flag.Bool("version", false, "displays olm version")

	tlsKeyPath = flag.String(
		"tls-key", "", "Path to use for private key (requires tls-cert)")

	tlsCertPath = flag.String(
		"tls-cert", "", "Path to use for certificate key (requires tls-key)")

	profiling = flag.Bool(
		"profiling", false, "serve profiling data (on port 8080)")

	namespace = flag.String(
		"namespace", "", "namespace where cleanup runs")
)

func init() {
	metrics.RegisterOLM()
}

// main function - entrypoint to OLM operator
func main() {
	// Get exit signal context
	ctx, cancel := context.WithCancel(signals.Context())
	defer cancel()

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

	// Set log level to debug if `debug` flag set
	logger := log.New()
	if *debug {
		logger.SetLevel(log.DebugLevel)
	}
	logger.Infof("log level %s", logger.Level)

	var useTLS bool
	if *tlsCertPath != "" && *tlsKeyPath == "" || *tlsCertPath == "" && *tlsKeyPath != "" {
		logger.Warn("both --tls-key and --tls-crt must be provided for TLS to be enabled, falling back to non-https")
	} else if *tlsCertPath == "" && *tlsKeyPath == "" {
		logger.Info("TLS keys not set, using non-https for metrics")
	} else {
		logger.Info("TLS keys set, using https for metrics")
		useTLS = true
	}

	// Serve a health check.
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Serve profiling if enabled
	if *profiling {
		logger.Infof("profiling enabled")
		profile.RegisterHandlers(healthMux)
	}

	go func() {
		err := http.ListenAndServe(":8080", healthMux)
		if err != nil {
			logger.Errorf("Health serving failed: %v", err)
		}
	}()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	if useTLS {
		go func() {
			err := http.ListenAndServeTLS(":8081", *tlsCertPath, *tlsKeyPath, metricsMux)
			if err != nil {
				logger.Errorf("Metrics (https) serving failed: %v", err)
			}
		}()
	} else {
		go func() {
			err := http.ListenAndServe(":8081", metricsMux)
			if err != nil {
				logger.Errorf("Metrics (http) serving failed: %v", err)
			}
		}()
	}

	// create a config client for operator status
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		log.Fatalf("error configuring client: %s", err.Error())
	}
	versionedConfigClient, err := configclientset.NewForConfig(config)
	if err != nil {
		err = fmt.Errorf("error configuring OpenShift Proxy client: %v", err)
		return
	}
	configClient, err := configv1client.NewForConfig(config)
	if err != nil {
		log.Fatalf("error configuring client: %s", err.Error())
	}
	opClient := operatorclient.NewClientFromConfig(*kubeConfigPath, logger)
	crClient, err := client.NewClient(*kubeConfigPath)
	if err != nil {
		log.Fatalf("error configuring client: %s", err.Error())
	}

	cleanup(logger, opClient, crClient)

	// Create a new instance of the operator.
	op, err := olm.NewOperator(
		ctx,
		olm.WithLogger(logger),
		olm.WithWatchedNamespaces(namespaces...),
		olm.WithResyncPeriod(*wakeupInterval),
		olm.WithExternalClient(crClient),
		olm.WithOperatorClient(opClient),
		olm.WithRestConfig(config),
		olm.WithConfigClient(versionedConfigClient),
	)
	if err != nil {
		log.WithError(err).Fatalf("error configuring operator")
		return
	}

	op.Run(ctx)
	<-op.Ready()

	if *writeStatusName != "" {
		operatorstatus.MonitorClusterStatus(*writeStatusName, op.AtLevel(), ctx.Done(), opClient, configClient)
	}

	if *writePackageServerStatusName != "" {
		logger.Info("Initializing cluster operator monitor for package server")

		names := *writePackageServerStatusName
		discovery := opClient.KubernetesInterface().Discovery()
		monitor, sender := operatorstatus.NewMonitor(names, logger, discovery, configClient)

		handler := operatorstatus.NewCSVWatchNotificationHandler(logger, op.GetCSVSetGenerator(), op.GetReplaceFinder(), sender)
		op.RegisterCSVWatchNotification(handler)

		go monitor.Run(op.Done())
	}

	<-op.Done()
}
