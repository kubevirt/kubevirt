package main

import (
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/rules"
)

func verifyArgs() error {
	numOfArgs := len(os.Args[1:])
	if numOfArgs != 1 {
		return fmt.Errorf("expected exactly 1 argument, got: %d", numOfArgs)
	}
	return nil
}

func main() {
	if err := verifyArgs(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	targetFile := os.Args[1]

	err := rules.SetupRules()
	if err != nil {
		panic(err)
	}

	promRule, err := rules.BuildPrometheusRule(
		"kubevirt-hyperconverged",
		metav1.OwnerReference{
			APIVersion: "v1",
			Kind:       "Namespace",
			Name:       "kubevirt-hyperconverged",
		},
	)
	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(promRule.Spec)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(targetFile, b, 0644)
	if err != nil {
		panic(err)
	}
}
