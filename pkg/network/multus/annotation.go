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

// MergeMultusAnnotations merges existing Multus annotations with new ones, preserving CNI arguments
// that may have been added by CNI plugins after the initial annotation was set.
func MergeMultusAnnotations(existingAnnotation, newAnnotation string) (string, error) {
	if existingAnnotation == "" {
		return newAnnotation, nil
	}
	if newAnnotation == "" {
		return existingAnnotation, nil
	}

	var existingNetworks, newNetworks []networkv1.NetworkSelectionElement

	// Parse existing annotation
	if err := json.Unmarshal([]byte(existingAnnotation), &existingNetworks); err != nil {
		return "", fmt.Errorf("failed to parse existing multus annotation: %w", err)
	}

	// Parse new annotation
	if err := json.Unmarshal([]byte(newAnnotation), &newNetworks); err != nil {
		return "", fmt.Errorf("failed to parse new multus annotation: %w", err)
	}

	// Create a map of existing networks by name for quick lookup
	existingNetworksMap := make(map[string]networkv1.NetworkSelectionElement)
	for _, network := range existingNetworks {
		key := fmt.Sprintf("%s/%s", network.Namespace, network.Name)
		existingNetworksMap[key] = network
	}

	// Merge networks, preserving existing CNI arguments
	mergedNetworks := make([]networkv1.NetworkSelectionElement, 0, len(newNetworks))
	for _, newNetwork := range newNetworks {
		key := fmt.Sprintf("%s/%s", newNetwork.Namespace, newNetwork.Name)
		if existingNetwork, exists := existingNetworksMap[key]; exists {
			// Merge the networks, preserving existing CNI arguments
			mergedNetwork := mergeNetworkSelectionElements(existingNetwork, newNetwork)
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

// mergeNetworkSelectionElements merges two NetworkSelectionElement objects,
// preserving CNI arguments from the existing element while updating other fields from the new element.
func mergeNetworkSelectionElements(existing, new networkv1.NetworkSelectionElement) networkv1.NetworkSelectionElement {
	merged := new // Start with the new network as base

	// Preserve existing CNI arguments if they exist
	if existing.CNIArgs != nil {
		if merged.CNIArgs == nil {
			merged.CNIArgs = &map[string]interface{}{}
		}
		// Merge CNI arguments, preserving existing ones
		for key, value := range *existing.CNIArgs {
			if _, exists := (*merged.CNIArgs)[key]; !exists {
				(*merged.CNIArgs)[key] = value
			}
		}
	}

	// Preserve other fields that might have been set by CNI plugins
	if len(existing.IPRequest) > 0 {
		if merged.IPRequest == nil {
			merged.IPRequest = existing.IPRequest
		}
	}

	if existing.MacRequest != "" && merged.MacRequest == "" {
		merged.MacRequest = existing.MacRequest
	}

	if existing.InfinibandGUIDRequest != "" && merged.InfinibandGUIDRequest == "" {
		merged.InfinibandGUIDRequest = existing.InfinibandGUIDRequest
	}

	if existing.InterfaceRequest != "" && merged.InterfaceRequest == "" {
		merged.InterfaceRequest = existing.InterfaceRequest
	}

	if len(existing.PortMappingsRequest) > 0 {
		if merged.PortMappingsRequest == nil {
			merged.PortMappingsRequest = existing.PortMappingsRequest
		}
	}

	if existing.BandwidthRequest != nil && merged.BandwidthRequest == nil {
		merged.BandwidthRequest = existing.BandwidthRequest
	}

	if len(existing.GatewayRequest) > 0 {
		if merged.GatewayRequest == nil {
			merged.GatewayRequest = existing.GatewayRequest
		}
	}

	return merged
}

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
