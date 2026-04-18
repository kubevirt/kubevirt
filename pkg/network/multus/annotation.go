/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
