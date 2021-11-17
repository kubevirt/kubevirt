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

	v1 "kubevirt.io/api/core/v1"
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

func generateMultusCNIAnnotation(vmi *v1.VirtualMachineInstance) (string, error) {
	multusNetworkAnnotationPool := multusNetworkAnnotationPool{}

	multusNonDefaultNetworks := filterMultusNonDefaultNetworks(vmi.Spec.Networks)
	for i, network := range multusNonDefaultNetworks {
		multusNetworkAnnotationPool.add(
			newMultusAnnotationData(vmi, network, fmt.Sprintf("net%d", i+1)))
	}

	if !multusNetworkAnnotationPool.isEmpty() {
		return multusNetworkAnnotationPool.toString()
	}
	return "", nil
}

func filterMultusNonDefaultNetworks(networks []v1.Network) []v1.Network {
	var multusNetworks []v1.Network
	for _, network := range networks {
		if network.Multus != nil {
			if network.Multus.Default {
				continue
			}
			multusNetworks = append(multusNetworks, network)
		}
	}
	return multusNetworks
}

func newMultusAnnotationData(vmi *v1.VirtualMachineInstance, network v1.Network, podInterfaceName string) multusNetworkAnnotation {
	multusIface := getIfaceByName(vmi, network.Name)
	namespace, networkName := getNamespaceAndNetworkName(vmi, network.Multus.NetworkName)
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

func getIfaceByName(vmi *v1.VirtualMachineInstance, name string) *v1.Interface {
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Name == name {
			return &vmi.Spec.Domain.Devices.Interfaces[i]
		}
	}
	return nil
}
