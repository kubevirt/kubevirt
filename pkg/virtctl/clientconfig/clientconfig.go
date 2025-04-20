/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package clientconfig

import (
	"context"
	"fmt"

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
// It then creates a kubecli.KubevirtClient and gets the namespace from the client config and returns them.
// Otherwise, it returns an error.
func ClientAndNamespaceFromContext(ctx context.Context) (virtClient kubecli.KubevirtClient, namespace string, overridden bool, err error) {
	clientConfig, ok := ctx.Value(clientConfigKey).(clientcmd.ClientConfig)
	if !ok {
		return nil, "", false, fmt.Errorf("unable to get client config from context")
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
