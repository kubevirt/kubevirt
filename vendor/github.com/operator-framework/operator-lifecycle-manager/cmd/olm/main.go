package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/operators/olm"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/signals"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/version"
)

const (
	envOperatorName         = "OPERATOR_NAME"
	envOperatorNamespace    = "OPERATOR_NAMESPACE"
	ALMManagedAnnotationKey = "alm-manager"

	defaultWakeupInterval = 5 * time.Minute
)

// helper function for required env vars
func envOrDie(varname, description string) string {
	val := os.Getenv(varname)
	if len(val) == 0 {
		log.Fatalf("must set env %s - %s", varname, description)
	}
	return val
}

// config flags defined globally so that they appear on the test binary as well
var (
	kubeConfigPath = flag.String(
		"kubeconfig", "", "absolute path to the kubeconfig file")

	wakeupInterval = flag.Duration(
		"interval", defaultWakeupInterval, "wake up interval")

	watchedNamespaces = flag.String(
		"watchedNamespaces", "", "comma separated list of namespaces for alm operator to watch. "+
			"If not set, or set to the empty string (e.g. `-watchedNamespaces=\"\"`), "+
			"alm operator will watch all namespaces in the cluster.")

	debug = flag.Bool(
		"debug", false, "use debug log level")

	version = flag.Bool("version", false, "displays olm version")
)

// main function - entrypoint to ALM operator
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

	// Env Vars
	operatorNamespace := envOrDie(
		envOperatorNamespace, "used to set annotation indicating which ALM operator manages a namespace")

	operatorName := envOrDie(
		envOperatorName, "used to distinguish ALM operators of the same name")

	annotation := map[string]string{
		ALMManagedAnnotationKey: fmt.Sprintf("%s.%s", operatorNamespace, operatorName),
	}

	// Set log level to debug if `debug` flag set
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	// `namespaces` will always contain at least one entry: if `*watchedNamespaces` is
	// the empty string, the resulting array will be `[]string{""}`.
	namespaces := strings.Split(*watchedNamespaces, ",")

	// Create a client for OLM
	crClient, err := client.NewClient(*kubeConfigPath)
	if err != nil {
		log.Fatalf("error configuring client: %s", err.Error())
	}

	opClient := operatorclient.NewClientFromConfig(*kubeConfigPath)

	// Create a new instance of the operator.
	operator, err := olm.NewOperator(crClient, opClient, &install.StrategyResolver{}, *wakeupInterval, annotation, namespaces)

	if err != nil {
		log.Fatalf("error configuring operator: %s", err.Error())
	}
	defer operator.Cleanup()

	// Serve a health check.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	operator.Run(stopCh)
}
