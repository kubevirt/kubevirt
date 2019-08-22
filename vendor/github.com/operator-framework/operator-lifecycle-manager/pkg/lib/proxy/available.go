package proxy

import (
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	apidiscovery "k8s.io/client-go/discovery"
)

const (
	// This is the error message thrown by ServerSupportsVersion function
	// when an API version is not supported by the server.
	notSupportedErrorMessage = "server does not support API version"
)

// IsAPIAvailable return true if OpenShift config API is present on the cluster.
// Otherwise, supported is set to false.
func IsAPIAvailable(discovery apidiscovery.DiscoveryInterface) (supported bool, err error) {
	if discovery == nil {
		err = errors.New("discovery interface can not be <nil>")
		return
	}

	opStatusGV := schema.GroupVersion{
		Group:   "config.openshift.io",
		Version: "v1",
	}
	if discoveryErr := apidiscovery.ServerSupportsVersion(discovery, opStatusGV); discoveryErr != nil {
		if strings.Contains(discoveryErr.Error(), notSupportedErrorMessage) {
			return
		}

		err = discoveryErr
		return
	}

	supported = true
	return
}
