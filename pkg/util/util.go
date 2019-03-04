package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
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

func GetMultusOrNpwgNetworkName(network v1.Network) string {
	if network.Npwg != nil {
		return network.Npwg.NetworkName
	}
	if network.Multus != nil {
		return network.Multus.NetworkName
	}
	return ""
}
