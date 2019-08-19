package utils

import (
	"flag"
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var kubeconfig string

// Init will initialize the kubeconfig variable for command line parameters
func Init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig file")
}

// LoadConfig builds config from kubernetes config
func LoadConfig() (*rest.Config, error) {
	if kubeconfig == "" {
		return rest.InClusterConfig()
	}

	c, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error loading KubeConfig: %v", err.Error())
	}

	return clientcmd.NewDefaultClientConfig(*c, &clientcmd.ConfigOverrides{}).ClientConfig()
}

// LoadClient builds controller runtime client that accepts any registered type
func LoadClient() (client.Client, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err.Error())
	}
	return client.New(config, client.Options{})
}
