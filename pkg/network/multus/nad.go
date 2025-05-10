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
 *
 */

package multus

import (
	"context"
	"fmt"
	"strings"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/precond"
)

func NetAttachDefNamespacedName(namespace, fullNetworkName string) types.NamespacedName {
	if strings.Contains(fullNetworkName, "/") {
		const twoParts = 2
		res := strings.SplitN(fullNetworkName, "/", twoParts)
		return types.NamespacedName{
			Namespace: res[0],
			Name:      res[1],
		}
	}

	return types.NamespacedName{
		Namespace: precond.MustNotBeEmpty(namespace),
		Name:      fullNetworkName,
	}
}

func getNetworkFromClient(virtClient kubecli.KubevirtClient, nadNamespacedName types.NamespacedName) (*networkv1.NetworkAttachmentDefinition, error) {
	return virtClient.NetworkClient().
		K8sCniCncfIoV1().
		NetworkAttachmentDefinitions(nadNamespacedName.Namespace).
		Get(context.Background(), nadNamespacedName.Name, metav1.GetOptions{})
}

func fetchNetWorkAttachmentDefinition(virtClient kubecli.KubevirtClient, cacheStore cache.Store, nadNamespacedName types.NamespacedName) (*networkv1.NetworkAttachmentDefinition, error) {
	obj, exists, err := cacheStore.GetByKey(nadNamespacedName.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch network attachment definition from informer cache for %s: %w", nadNamespacedName.String(), err)
	}
	if exists {
		return obj.(*networkv1.NetworkAttachmentDefinition), nil
	}
	fmt.Printf("Cache miss: fetching network attachment definition from API server for %s\n", nadNamespacedName.String())
	return getNetworkFromClient(virtClient, nadNamespacedName)
}

func NetworkToResource(virtClient kubecli.KubevirtClient, networkAttachmentDefinitionCache cache.Store, vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	networkToResourceMap := map[string]string{}

	for _, network := range vmi.Spec.Networks {
		if network.Multus == nil {
			continue
		}

		nadNamespacedName := NetAttachDefNamespacedName(vmi.Namespace, network.Multus.NetworkName)
		netAttachDef, err := fetchNetWorkAttachmentDefinition(virtClient, networkAttachmentDefinitionCache, nadNamespacedName)
		if err != nil {
			return nil, fmt.Errorf("failed to locate network attachment definition for %s: %w", nadNamespacedName.String(), err)
		}

		networkToResourceMap[network.Name] = netAttachDef.Annotations[ResourceNameAnnotation]
	}

	return networkToResourceMap, nil
}
