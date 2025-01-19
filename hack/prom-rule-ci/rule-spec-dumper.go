package main

import (
	"encoding/json"
	"fmt"
	"os"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hcoRules "github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/rules"
	observabilityRules "github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/observability/rules"
)

func verifyArgs() error {
	numOfArgs := len(os.Args[1:])
	if numOfArgs != 2 {
		return fmt.Errorf("got %d arguments instead of 2, expected usage: %s <rules source ('hyperconverged' or 'observability')> <output file>", numOfArgs, os.Args[0])
	}
	return nil
}

func main() {
	err := verifyArgs()
	checkErrorAndExit(err)

	targetRules := os.Args[1]
	targetFile := os.Args[2]

	var promRule *promv1.PrometheusRule

	switch targetRules {
	case "hyperconverged":
		promRule = buildHyperconvergedRule()
	case "observability":
		promRule = buildObservabilityRule()
	default:
		checkErrorAndExit(fmt.Errorf("invalid target rules: %s", targetRules))
	}

	b, err := json.Marshal(promRule.Spec)
	checkErrorAndExit(err)

	err = os.WriteFile(targetFile, b, 0644)
	checkErrorAndExit(err)
}

func buildHyperconvergedRule() *promv1.PrometheusRule {
	err := hcoRules.SetupRules()
	checkErrorAndExit(err)

	rule, err := hcoRules.BuildPrometheusRule("default", metav1.OwnerReference{})
	checkErrorAndExit(err)

	return rule
}

func buildObservabilityRule() *promv1.PrometheusRule {
	err := observabilityRules.SetupRules()
	checkErrorAndExit(err)

	rule, err := observabilityRules.BuildPrometheusRule("default", &metav1.OwnerReference{})
	checkErrorAndExit(err)

	return rule
}

func checkErrorAndExit(err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stdout, "ERROR: %v\n", err)
	os.Exit(1)
}
