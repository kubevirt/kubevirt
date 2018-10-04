package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const ServiceAccountNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
const namespaceKubevirt = "kubevirt"

func GetNamespace() (string, error) {
	if data, err := ioutil.ReadFile(ServiceAccountNamespaceFile); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to determine namespace from %s: %v", ServiceAccountNamespaceFile, err)
	}
	return namespaceKubevirt, nil
}
