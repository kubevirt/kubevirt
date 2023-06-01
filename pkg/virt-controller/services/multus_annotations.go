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

package services

import (
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type multusNetworkAnnotation struct {
	InterfaceName string `json:"interface"`
	Mac           string `json:"mac,omitempty"`
	NetworkName   string `json:"name"`
	Namespace     string `json:"namespace"`
}

type multusNetworkAnnotationPool struct {
	pool []multusNetworkAnnotation
}

func (mnap *multusNetworkAnnotationPool) add(multusNetworkAnnotation multusNetworkAnnotation) {
	mnap.pool = append(mnap.pool, multusNetworkAnnotation)
}

func (mnap multusNetworkAnnotationPool) isEmpty() bool {
	return len(mnap.pool) == 0
}

func (mnap multusNetworkAnnotationPool) toString() (string, error) {
	multusNetworksAnnotation, err := json.Marshal(mnap.pool)
	if err != nil {
		return "", fmt.Errorf("failed to create JSON list from multus interface pool %v", mnap.pool)
	}
	return string(multusNetworksAnnotation), nil
}

func GenerateMultusCNIAnnotation(namespace string, interfaces []v1.Interface, networks []v1.Network) (string, error) {
	return GenerateMultusCNIAnnotationFromNameScheme(namespace, interfaces, networks, namescheme.CreateHashedNetworkNameScheme(networks))
}

func GenerateMultusCNIAnnotationFromNameScheme(namespace string, interfaces []v1.Interface, networks []v1.Network, networkNameScheme map[string]string) (string, error) {
	multusNetworkAnnotationPool := multusNetworkAnnotationPool{}

	for _, network := range networks {
		if vmispec.IsSecondaryMultusNetwork(network) {
			podInterfaceName := networkNameScheme[network.Name]
			multusNetworkAnnotationPool.add(
				newMultusAnnotationData(namespace, interfaces, network, podInterfaceName))
		}
	}

	if !multusNetworkAnnotationPool.isEmpty() {
		return multusNetworkAnnotationPool.toString()
	}
	return "", nil
}

func newMultusAnnotationData(namespace string, interfaces []v1.Interface, network v1.Network, podInterfaceName string) multusNetworkAnnotation {
	multusIface := vmispec.LookupInterfaceByName(interfaces, network.Name)
	namespace, networkName := getNamespaceAndNetworkName(namespace, network.Multus.NetworkName)
	var multusIfaceMac string
	if multusIface != nil {
		multusIfaceMac = multusIface.MacAddress
	}
	return multusNetworkAnnotation{
		InterfaceName: podInterfaceName,
		Mac:           multusIfaceMac,
		Namespace:     namespace,
		NetworkName:   networkName,
	}
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
