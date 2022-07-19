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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package sriov

import (
	"encoding/json"
	"errors"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"
)

type PCIAddressPoolWithNetworkStatus struct {
	networkNameToPciAddress map[string]string
}

// NewPCIAddressPoolWithNetworkStatus creates a PCI address pool based on the network status multus annotation
func NewPCIAddressPoolWithNetworkStatus(sriovIfaces []v1.Interface, multusNetworks []v1.Network, vmiAnnotations map[string]string) (*PCIAddressPoolWithNetworkStatus, error) {
	pool := &PCIAddressPoolWithNetworkStatus{
		networkNameToPciAddress: make(map[string]string),
	}

	if err := pool.loadNetworkNameToPciAddress(sriovIfaces, multusNetworks, vmiAnnotations); err != nil {
		return nil, err
	}

	return pool, nil
}

func (p *PCIAddressPoolWithNetworkStatus) loadNetworkNameToPciAddress(sriovIfaces []v1.Interface, multusNetworks []v1.Network, vmiAnnotations map[string]string) error {
	multusInterfaceNameToNetworkStatusMap, err := loadMultusInterfaceNameToNetworkStatus(vmiAnnotations)
	if err != nil {
		return err
	}
	sriovNetworkToMultusInterfaceNameMap := loadNetworkNameToMultusInterfaceName(sriovIfaces, multusNetworks)

	for networkName, multusInterfaceName := range sriovNetworkToMultusInterfaceNameMap {
		networkStatusEntry := multusInterfaceNameToNetworkStatusMap[multusInterfaceName]
		if networkStatusEntry.DeviceInfo == nil || networkStatusEntry.DeviceInfo.Pci == nil {
			return errors.New(fmt.Sprintf("SRIOV interface %s has no corresponding device-info/pci-address in the appropriate network status annotations", networkName))
		}

		pciAddress := networkStatusEntry.DeviceInfo.Pci.PciAddress
		if pciAddress == "" {
			return errors.New(fmt.Sprintf("failed to associate pci-address to SRIOV interface %s", networkName))
		}
		p.networkNameToPciAddress[networkName] = pciAddress
	}
	return nil
}

func (p *PCIAddressPoolWithNetworkStatus) Pop(networkName string) (string, error) {
	pciAddress, exists := p.networkNameToPciAddress[networkName]
	if !exists {
		return "", fmt.Errorf("resource for SR-IOV network %s does not exist", networkName)
	}

	delete(p.networkNameToPciAddress, networkName)
	return pciAddress, nil
}

func loadNetworkNameToMultusInterfaceName(interfaceList []v1.Interface, networkList []v1.Network) map[string]string {
	networkToMultusInterfaceNameMap := map[string]string{}
	for i, network := range networkList {
		// multusInterfaceName follows a predefined indexation of the non-default multus networks (See pkg/virt-controller/services/multus_annotations.go: generateMultusCNIAnnotation.
		// In order to not break this convention, this for loop should not break/continue
		multusInterfaceName := fmt.Sprintf("net%d", i+1)
		associatedNetwork := getIfaceByNetworkName(interfaceList, network.Name)
		if associatedNetwork != nil {
			networkToMultusInterfaceNameMap[network.Name] = multusInterfaceName
		}
	}
	return networkToMultusInterfaceNameMap
}

func loadMultusInterfaceNameToNetworkStatus(vmiAnnotations map[string]string) (map[string]networkv1.NetworkStatus, error) {
	multusInterfaceNameToNetworkStatusMap := map[string]networkv1.NetworkStatus{}
	networkStatuses, err := getNetworkStatusList(vmiAnnotations)
	if err != nil {
		return nil, err
	}

	for _, networkStatus := range networkStatuses {
		multusInterfaceNameToNetworkStatusMap[networkStatus.Interface] = networkStatus
	}
	return multusInterfaceNameToNetworkStatusMap, nil
}

func getIfaceByNetworkName(ifaces []v1.Interface, networkName string) *v1.Interface {
	for i, iface := range ifaces {
		if iface.Name == networkName {
			return &ifaces[i]
		}
	}
	return nil
}

func getNetworkStatusList(vmiAnnotations map[string]string) ([]networkv1.NetworkStatus, error) {
	netStatusesJson, isSet := vmiAnnotations[networkv1.NetworkStatusAnnot]
	if !isSet {
		return nil, errors.New(fmt.Sprintf("%s annotation is not present on vmi annotations", networkv1.NetworkStatusAnnot))
	}

	var netStatuses []networkv1.NetworkStatus
	err := json.Unmarshal([]byte(netStatusesJson), &netStatuses)

	return netStatuses, err
}
