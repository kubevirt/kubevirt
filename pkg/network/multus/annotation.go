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
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
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
	registeredBindingPlugins map[string]v1.InterfaceBindingPlugin,
) (string, error) {
	return GenerateCNIAnnotationFromNameScheme(
		namespace,
		interfaces,
		networks,
		namescheme.CreateHashedNetworkNameScheme(networks),
		registeredBindingPlugins,
	)
}

func GenerateCNIAnnotationFromNameScheme(
	namespace string,
	interfaces []v1.Interface,
	networks []v1.Network,
	networkNameScheme map[string]string,
	registeredBindingPlugins map[string]v1.InterfaceBindingPlugin,
) (string, error) {
	var networkSelectionElements []networkv1.NetworkSelectionElement

	for _, network := range networks {
		if vmispec.IsSecondaryMultusNetwork(network) {
			podInterfaceName := networkNameScheme[network.Name]
			networkSelectionElements = append(networkSelectionElements, newAnnotationData(namespace, interfaces, network, podInterfaceName))
		}

		if iface := vmispec.LookupInterfaceByName(interfaces, network.Name); iface.Binding != nil {
			bindingPluginAnnotationData, err := newBindingPluginAnnotationData(
				registeredBindingPlugins,
				iface.Binding.Name,
				namespace,
				network.Name,
			)
			if err != nil {
				return "", err
			}
			if bindingPluginAnnotationData != nil {
				networkSelectionElements = append(networkSelectionElements, *bindingPluginAnnotationData)
			}
		}
	}

	if len(networkSelectionElements) == 0 {
		return "", nil
	}

	multusNetworksAnnotation, err := json.Marshal(networkSelectionElements)
	if err != nil {
		return "", fmt.Errorf("failed to create JSON list from networkSelectionElements: %v", networkSelectionElements)
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
	registeredBindingPlugins map[string]v1.InterfaceBindingPlugin,
	pluginName,
	namespace,
	networkName string,
) (*networkv1.NetworkSelectionElement, error) {
	plugin, exists := registeredBindingPlugins[pluginName]
	if !exists {
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

// MergeMultusAnnotations merges existing Multus annotations with new ones, preserving all fields
// that may have been added by CNI plugins after the initial annotation was set.
func MergeMultusAnnotations(existingAnnotation, newAnnotation string) (string, error) {
	if existingAnnotation == "" {
		return newAnnotation, nil
	}
	if newAnnotation == "" {
		return existingAnnotation, nil
	}

	var existingNetworks, newNetworks []map[string]interface{}

	// Parse existing annotation as generic JSON
	if err := json.Unmarshal([]byte(existingAnnotation), &existingNetworks); err != nil {
		return "", fmt.Errorf("failed to parse existing multus annotation: %w", err)
	}

	// Parse new annotation as generic JSON
	if err := json.Unmarshal([]byte(newAnnotation), &newNetworks); err != nil {
		return "", fmt.Errorf("failed to parse new multus annotation: %w", err)
	}

	// Create a map of existing networks by name for quick lookup
	existingNetworksMap := make(map[string]map[string]interface{})
	for _, network := range existingNetworks {
		name, _ := network["name"].(string)
		namespace, _ := network["namespace"].(string)
		if name != "" {
			key := fmt.Sprintf("%s/%s", namespace, name)
			existingNetworksMap[key] = network
		}
	}

	// Merge networks, preserving existing fields
	mergedNetworks := make([]map[string]interface{}, 0, len(newNetworks))
	for _, newNetwork := range newNetworks {
		name, _ := newNetwork["name"].(string)
		namespace, _ := newNetwork["namespace"].(string)
		if name == "" {
			continue
		}

		key := fmt.Sprintf("%s/%s", namespace, name)
		if existingNetwork, exists := existingNetworksMap[key]; exists {
			// Merge the networks, preserving existing fields
			mergedNetwork := mergeNetworkMaps(existingNetwork, newNetwork)
			mergedNetworks = append(mergedNetworks, mergedNetwork)
		} else {
			// New network, add as-is
			mergedNetworks = append(mergedNetworks, newNetwork)
		}
	}

	// Marshal the merged result
	mergedAnnotation, err := json.Marshal(mergedNetworks)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged multus annotation: %w", err)
	}

	return string(mergedAnnotation), nil
}

// mergeNetworkMaps merges two network maps, preserving fields from existing while updating with new
// This approach treats the network as a generic JSON object and preserves all fields that KubeVirt doesn't explicitly manage
func mergeNetworkMaps(existing, newMap map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Start with the new network as base
	for key, value := range newMap {
		merged[key] = value
	}

	// Preserve existing fields that aren't in new, or merge nested objects
	for key, existingValue := range existing {
		if newValue, exists := merged[key]; exists {
			// Field exists in both, check if we need to deep merge
			if existingMap, existingIsMap := existingValue.(map[string]interface{}); existingIsMap {
				if newMap, newIsMap := newValue.(map[string]interface{}); newIsMap {
					// Both are maps, deep merge them
					merged[key] = mergeNetworkMaps(existingMap, newMap)
				}
			}
			// If not maps, new value takes precedence (already set above)
		} else {
			// Field only exists in existing, preserve it
			merged[key] = existingValue
		}
	}

	return merged
}
