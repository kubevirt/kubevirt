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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"
	"strings"

	"kubevirt.io/client-go/precond"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	k8scnicncfiov1 "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/typed/k8s.cni.cncf.io/v1"
)

const MULTUS_RESOURCE_NAME_ANNOTATION = "k8s.v1.cni.cncf.io/resourceName"
const MULTUS_DEFAULT_NETWORK_CNI_ANNOTATION = "v1.multus-cni.io/default-network"

func GetNetworkAttachmentDefinitionByName(networkClient k8scnicncfiov1.K8sCniCncfIoV1Interface, namespace string, multusNetworks []virtv1.Network) (map[string]*networkv1.NetworkAttachmentDefinition, error) {
	nadMap := map[string]*networkv1.NetworkAttachmentDefinition{}
	for _, network := range multusNetworks {
		if !vmispec.IsMultusNetwork(network) {
			return nil, fmt.Errorf("failed asserting network (%s) is multus", network.Name)
		}
		ns, networkName := getNamespaceAndNetworkName(namespace, network.Multus.NetworkName)
		nad, err := networkClient.NetworkAttachmentDefinitions(ns).Get(context.Background(), networkName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to locate network attachment definition %s/%s", ns, networkName)
		}
		nadMap[network.Name] = nad
	}
	return nadMap, nil
}

func ExtractNetworkToResourceMap(nadMap map[string]*networkv1.NetworkAttachmentDefinition) map[string]string {
	networkToResourceMap := map[string]string{}
	for networkName, nad := range nadMap {
		networkToResourceMap[networkName] = getResourceNameForNetwork(nad)
	}
	return networkToResourceMap
}

func getResourceNameForNetwork(network *networkv1.NetworkAttachmentDefinition) string {
	resourceName, ok := network.Annotations[MULTUS_RESOURCE_NAME_ANNOTATION]
	if ok {
		return resourceName
	}
	return "" // meaning the network is not served by resources
}

func getNamespaceAndNetworkName(namespace string, fullNetworkName string) (string, string) {
	if strings.Contains(fullNetworkName, "/") {
		res := strings.SplitN(fullNetworkName, "/", 2)
		return res[0], res[1]
	}
	return precond.MustNotBeEmpty(namespace), fullNetworkName
}
