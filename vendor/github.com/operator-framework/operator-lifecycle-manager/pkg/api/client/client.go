package client

import (
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/internalversion"
	"k8s.io/client-go/rest"
)

// NewClient creates a client that can interact with OLM resources in k8s api
func NewClient(kubeconfig string) (client versioned.Interface, err error) {
	var config *rest.Config
	config, err = getConfig(kubeconfig)
	if err != nil {
		return
	}
	return versioned.NewForConfig(config)
}

// NewInternalClient creates a client that can interact with OLM resources in the k8s api using internal versions.
func NewInternalClient(kubeconfig string) (client internalversion.Interface, err error) {
	var config *rest.Config
	config, err = getConfig(kubeconfig)
	if err != nil {
		return
	}
	return internalversion.NewForConfig(config)
}
