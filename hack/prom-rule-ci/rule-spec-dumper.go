package main

import (
	"encoding/json"

	"fmt"
	"io/ioutil"
	"os"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
)

func verifyArgs(args []string) error {
	numOfArgs := len(os.Args[1:])
	if numOfArgs != 1 {
		return fmt.Errorf("expected exactly 1 argument, got: %d", numOfArgs)
	}
	return nil
}

func main() {
	if err := verifyArgs(os.Args); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	targetFile := os.Args[1]

	promRuleSpec := operands.NewPrometheusRuleSpec()
	b, err := json.Marshal(promRuleSpec)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(targetFile, b, 0644)
	if err != nil {
		panic(err)
	}
}
