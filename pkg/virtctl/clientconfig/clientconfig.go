package clientconfig

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
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

// FromContext returns a clientcmd.Clientconfig value stored in ctx, if any.
// Otherwise, it returns an error.
func FromContext(ctx context.Context) (clientcmd.ClientConfig, error) {
	clientConfig, ok := ctx.Value(clientConfigKey).(clientcmd.ClientConfig)
	if !ok {
		return nil, fmt.Errorf("unable to get client config from context")
	}
	return clientConfig, nil
}
