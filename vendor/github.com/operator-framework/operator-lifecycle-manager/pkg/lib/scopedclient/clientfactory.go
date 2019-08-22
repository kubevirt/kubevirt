package scopedclient

import (
	"k8s.io/client-go/rest"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
)

// NewFactory returns a new instance of Factory.
func NewFactory(config *rest.Config) *Factory {
	return &Factory{
		config: config,
	}
}

// Factory holds on to a default config and can create a new client instance(s)
// bound to any bearer token specified.
type Factory struct {
	// default config object from which new client instance(s) will be created.
	config *rest.Config
}

// NewOperatorClient return a new instance of operator client from the bearer
// token specified.
func (f *Factory) NewOperatorClient(token string) (client operatorclient.ClientInterface, err error) {
	scoped := copy(f.config, token)
	client, err = operatorclient.NewClientFromRestConfig(scoped)

	return
}

// NewKubernetesClient return a new instance of CR client from the bearer
// token specified.
func (f *Factory) NewKubernetesClient(token string) (client versioned.Interface, err error) {
	scoped := copy(f.config, token)
	client, err = versioned.NewForConfig(scoped)

	return
}

func copy(config *rest.Config, token string) *rest.Config {
	copied := rest.AnonymousClientConfig(config)
	copied.BearerToken = token

	return copied
}
