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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/netbinding"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func GenerateMultusCNIAnnotation(namespace string, interfaces []v1.Interface, networks []v1.Network, config *virtconfig.ClusterConfig) (string, error) {
	return GenerateMultusCNIAnnotationFromNameScheme(namespace, interfaces, networks, namescheme.CreateHashedNetworkNameScheme(networks), config)
}

func GenerateMultusCNIAnnotationFromNameScheme(namespace string, interfaces []v1.Interface, networks []v1.Network, networkNameScheme map[string]string, config *virtconfig.ClusterConfig) (string, error) {
	multusNetworkAnnotationPool := multus.NetworkAnnotationPool{}

	for _, network := range networks {
		if vmispec.IsSecondaryMultusNetwork(network) {
			podInterfaceName := networkNameScheme[network.Name]
			multusNetworkAnnotationPool.Add(
				multus.NewAnnotationData(namespace, interfaces, network, podInterfaceName))
		}

		if config != nil && config.NetworkBindingPlugingsEnabled() {
			if iface := vmispec.LookupInterfaceByName(interfaces, network.Name); iface.Binding != nil {
				bindingPluginAnnotationData, err := newBindingPluginMultusAnnotationData(
					config.GetConfig(), iface.Binding.Name, namespace, network.Name)
				if err != nil {
					return "", err
				}
				if bindingPluginAnnotationData != nil {
					multusNetworkAnnotationPool.Add(*bindingPluginAnnotationData)
				}
			}
		}
	}

	if !multusNetworkAnnotationPool.IsEmpty() {
		return multusNetworkAnnotationPool.ToString()
	}
	return "", nil
}

func newBindingPluginMultusAnnotationData(kvConfig *v1.KubeVirtConfiguration, pluginName, namespace, networkName string) (*networkv1.NetworkSelectionElement, error) {
	plugin := netbinding.ReadNetBindingPluginConfiguration(kvConfig, pluginName)
	if plugin == nil {
		return nil, fmt.Errorf("unable to find the network binding plugin '%s' in Kubevirt configuration", pluginName)
	}

	if plugin.NetworkAttachmentDefinition == "" {
		return nil, nil
	}
	netAttachDefNamespace, netAttachDefName := multus.GetNamespaceAndNetworkName(namespace, plugin.NetworkAttachmentDefinition)

	// cniArgNetworkName is the CNI arg name for the VM spec network logical name.
	// The binding plugin CNI should read this arg and realize which logical network it should modify.
	const cniArgNetworkName = "logicNetworkName"

	return &networkv1.NetworkSelectionElement{
		Namespace: netAttachDefNamespace,
		Name:      netAttachDefName,
		CNIArgs: &map[string]interface{}{
			cniArgNetworkName: networkName,
		},
	}, nil
}

func NonDefaultMultusNetworksIndexedByIfaceName(pod *k8sv1.Pod) map[string]networkv1.NetworkStatus {
	indexedNetworkStatus := map[string]networkv1.NetworkStatus{}
	podNetworkStatus, found := pod.Annotations[networkv1.NetworkStatusAnnot]

	if !found {
		return indexedNetworkStatus
	}

	var networkStatus []networkv1.NetworkStatus
	if err := json.Unmarshal([]byte(podNetworkStatus), &networkStatus); err != nil {
		log.Log.Errorf("failed to unmarshall pod network status: %v", err)
		return indexedNetworkStatus
	}

	for _, ns := range networkStatus {
		if ns.Default {
			continue
		}
		indexedNetworkStatus[ns.Interface] = ns
	}

	return indexedNetworkStatus
}
