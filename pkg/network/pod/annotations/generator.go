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

package annotations

import (
	k8Scorev1 "k8s.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/network"
)

type Generator struct {
	clusterConfig *virtconfig.ClusterConfig
}

func NewGenerator(clusterConfig *virtconfig.ClusterConfig) Generator {
	return Generator{
		clusterConfig: clusterConfig,
	}
}

func (g Generator) Generate(vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	nonAbsentIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	nonAbsentNets := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, nonAbsentIfaces)
	multusAnnotation, err := multus.GenerateCNIAnnotation(vmi.Namespace, nonAbsentIfaces, nonAbsentNets, g.clusterConfig)
	if err != nil {
		return nil, err
	}

	annotations := map[string]string{}

	if multusAnnotation != "" {
		annotations[networkv1.NetworkAttachmentAnnot] = multusAnnotation
	}

	defaultMultusNetworks := vmispec.FilterNetworksSpec(vmi.Spec.Networks, func(network v1.Network) bool {
		return network.NetworkSource.Multus != nil && network.NetworkSource.Multus.Default
	})

	if len(defaultMultusNetworks) > 0 {
		annotations[multus.DefaultNetworkCNIAnnotation] = defaultMultusNetworks[0].Multus.NetworkName
	}

	if shouldAddIstioKubeVirtAnnotation(vmi) {
		const defaultBridgeName = "k6t-eth0"
		annotations[istio.KubeVirtTrafficAnnotation] = defaultBridgeName
	}

	return annotations, nil
}

func (g Generator) GenerateFromSource(vmi *v1.VirtualMachineInstance, sourcePod *k8Scorev1.Pod) (map[string]string, error) {
	annotations := map[string]string{}

	if namescheme.PodHasOrdinalInterfaceName(network.NonDefaultMultusNetworksIndexedByIfaceName(sourcePod)) {
		ordinalNameScheme := namescheme.CreateOrdinalNetworkNameScheme(vmi.Spec.Networks)
		multusNetworksAnnotation, err := multus.GenerateCNIAnnotationFromNameScheme(
			vmi.Namespace,
			vmi.Spec.Domain.Devices.Interfaces,
			vmi.Spec.Networks,
			ordinalNameScheme,
			g.clusterConfig,
		)
		if err != nil {
			return nil, err
		}

		annotations[networkv1.NetworkAttachmentAnnot] = multusNetworksAnnotation
	}

	return annotations, nil
}

func shouldAddIstioKubeVirtAnnotation(vmi *v1.VirtualMachineInstance) bool {
	interfacesWithMasqueradeBinding := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.Masquerade != nil
	})

	return len(interfacesWithMasqueradeBinding) > 0
}
