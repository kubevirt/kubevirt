package util

import (
	"fmt"
	"os"
)

const OperatorNamespaceEnv = "OPERATOR_NAMESPACE"

func GetOperatorNamespaceFromEnv() (string, error) {
	if namespace, ok := os.LookupEnv(OperatorNamespaceEnv); ok {
		return namespace, nil
	}

	return "", fmt.Errorf("%s unset or empty in environment", OperatorNamespaceEnv)
}
