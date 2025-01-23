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

package multus

import (
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/netbinding"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	// DefaultNetworkCNIAnnotation is used when one wants to instruct Multus to connect the pod's primary interface
	// to a network other than Multus's `clusterNetwork` field under /etc/cni/net.d
	// The value of this annotation should be a NetworkAttachmentDefinition's name
	DefaultNetworkCNIAnnotation = "v1.multus-cni.io/default-network"

	// ResourceNameAnnotation represents a resource name that is associated with the network.
	// It could be found on NetworkAttachmentDefinition objects.
	ResourceNameAnnotation = "k8s.v1.cni.cncf.io/resourceName"
)

func GenerateCNIAnnotation(
	namespace string,
	interfaces []v1.Interface,
	networks []v1.Network,
	config *virtconfig.ClusterConfig,
) (string, error) {
	return GenerateCNIAnnotationFromNameScheme(namespace, interfaces, networks, namescheme.CreateHashedNetworkNameScheme(networks), config)
}

func GenerateCNIAnnotationFromNameScheme(
	namespace string,
	interfaces []v1.Interface,
	networks []v1.Network,
	networkNameScheme map[string]string,
	config *virtconfig.ClusterConfig,
) (string, error) {
	multusNetworkAnnotationPool := networkAnnotationPool{}

	for _, network := range networks {
		if vmispec.IsSecondaryMultusNetwork(network) {
			podInterfaceName := networkNameScheme[network.Name]
			multusNetworkAnnotationPool.Add(
				newAnnotationData(namespace, interfaces, network, podInterfaceName))
		}

		if config != nil {
			if iface := vmispec.LookupInterfaceByName(interfaces, network.Name); iface.Binding != nil {
				bindingPluginAnnotationData, err := newBindingPluginAnnotationData(
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

type networkAnnotationPool struct {
	pool []networkv1.NetworkSelectionElement
}

func (nap *networkAnnotationPool) Add(multusNetworkAnnotation networkv1.NetworkSelectionElement) {
	nap.pool = append(nap.pool, multusNetworkAnnotation)
}

func (nap *networkAnnotationPool) IsEmpty() bool {
	return len(nap.pool) == 0
}

func (nap *networkAnnotationPool) ToString() (string, error) {
	multusNetworksAnnotation, err := json.Marshal(nap.pool)
	if err != nil {
		return "", fmt.Errorf("failed to create JSON list from multus interface pool %v", nap.pool)
	}
	return string(multusNetworksAnnotation), nil
}

func newAnnotationData(
	namespace string,
	interfaces []v1.Interface,
	network v1.Network,
	podInterfaceName string,
) networkv1.NetworkSelectionElement {
	multusIface := vmispec.LookupInterfaceByName(interfaces, network.Name)
	nadNamespacedName := NetAttachDefNamespacedName(namespace, network.Multus.NetworkName)
	var multusIfaceMac string
	if multusIface != nil {
		multusIfaceMac = multusIface.MacAddress
	}
	return networkv1.NetworkSelectionElement{
		InterfaceRequest: podInterfaceName,
		MacRequest:       multusIfaceMac,
		Namespace:        nadNamespacedName.Namespace,
		Name:             nadNamespacedName.Name,
	}
}

func newBindingPluginAnnotationData(
	kvConfig *v1.KubeVirtConfiguration,
	pluginName, namespace, networkName string,
) (*networkv1.NetworkSelectionElement, error) {
	plugin := netbinding.ReadNetBindingPluginConfiguration(kvConfig, pluginName)
	if plugin == nil {
		return nil, fmt.Errorf("unable to find the network binding plugin '%s' in Kubevirt configuration", pluginName)
	}

	if plugin.NetworkAttachmentDefinition == "" {
		return nil, nil
	}
	nadNamespacedName := NetAttachDefNamespacedName(namespace, plugin.NetworkAttachmentDefinition)

	// cniArgNetworkName is the CNI arg name for the VM spec network logical name.
	// The binding plugin CNI should read this arg and realize which logical network it should modify.
	const cniArgNetworkName = "logicNetworkName"

	return &networkv1.NetworkSelectionElement{
		Namespace: nadNamespacedName.Namespace,
		Name:      nadNamespacedName.Name,
		CNIArgs: &map[string]interface{}{
			cniArgNetworkName: networkName,
		},
	}, nil
}
