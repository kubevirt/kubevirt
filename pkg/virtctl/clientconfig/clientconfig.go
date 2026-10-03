package clientconfig

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
)

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// clientConfigKey is the key for clientcmd.ClientConfig values in Contexts.
// It is unexported; clients use clientconfig.NewContext and clientconfig.FromContext
// instead of using this key directly.
var clientConfigKey key

// NewContext returns a new Context that stores a clientConfig as value.
func NewContext(ctx context.Context, clientConfig clientcmd.ClientConfig) context.Context {
	return context.WithValue(ctx, clientConfigKey, clientConfig)
}

// clientConfigFromContext retrieves the clientcmd.ClientConfig stored in ctx, if any.
// Otherwise, it returns an error.
func clientConfigFromContext(ctx context.Context) (clientcmd.ClientConfig, error) {
	clientConfig, ok := ctx.Value(clientConfigKey).(clientcmd.ClientConfig)
	if !ok {
		return nil, fmt.Errorf("unable to get client config from context")
	}
	return clientConfig, nil
}

// ClientAndNamespaceFromContext tries to retrieve a clientcmd.Clientconfig value stored in ctx, if any.
// It then creates a kubecli.KubevirtClient and gets the namespace from the client config and returns them.
// Otherwise, it returns an error.
func ClientAndNamespaceFromContext(ctx context.Context) (virtClient kubecli.KubevirtClient, namespace string, overridden bool, err error) {
	clientConfig, err := clientConfigFromContext(ctx)
	if err != nil {
		return nil, "", false, err
	}
	virtClient, err = kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, "", false, err
	}
	namespace, overridden, err = clientConfig.Namespace()
	if err != nil {
		return nil, "", false, err
	}
	return virtClient, namespace, overridden, nil
}

// K8sClientFromContext tries to retrieve a clientcmd.ClientConfig value stored in ctx, if any.
// It then creates a kubernetes.Interface and returns it.
// Otherwise, it returns an error.
func K8sClientFromContext(ctx context.Context) (kubernetes.Interface, error) {
	clientConfig, err := clientConfigFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return kubecli.GetK8sClientFromClientConfig(clientConfig)
}
