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

// ClientAndNamespaceFromContext tries to retrieve a clientcmd.Clientconfig value stored in ctx, if any.
// It then creates a kubecli.KubevirtClient, a kubernetes.Interface and gets the namespace from the client config and returns them.
// Otherwise, it returns an error.
func ClientAndNamespaceFromContext(ctx context.Context) (virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, namespace string, overridden bool, err error) {
	clientConfig, ok := ctx.Value(clientConfigKey).(clientcmd.ClientConfig)
	if !ok {
		return nil, nil, "", false, fmt.Errorf("unable to get client config from context")
	}
	virtClient, err = kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, nil, "", false, err
	}
	k8sClient, err = kubecli.GetK8sClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, nil, "", false, err
	}
	namespace, overridden, err = clientConfig.Namespace()
	if err != nil {
		return nil, nil, "", false, err
	}
	return virtClient, k8sClient, namespace, overridden, nil
}
