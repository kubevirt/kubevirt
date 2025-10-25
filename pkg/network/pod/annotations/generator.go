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

package annotations

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	k8scorev1 "k8s.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type clusterConfigurer interface {
	GetNetworkBindings() map[string]v1.InterfaceBindingPlugin
}

type networkSelectionElement map[string]interface{}

func (n networkSelectionElement) clone() networkSelectionElement {
	for k, v := range n {
		n[k] = v
	}
	return n
}

type Generator struct {
	clusterConfigurer clusterConfigurer
}

func NewGenerator(clusterConfigurer clusterConfigurer) Generator {
	return Generator{
		clusterConfigurer: clusterConfigurer,
	}
}

// Generate generates network related annotations for a newly created virt-launcher pod
func (g Generator) Generate(vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	nonAbsentIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	nonAbsentNets := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, nonAbsentIfaces)
	multusAnnotation, err := multus.GenerateCNIAnnotation(
		vmi.Namespace,
		nonAbsentIfaces,
		nonAbsentNets,
		g.clusterConfigurer.GetNetworkBindings(),
	)
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
		// both annotations do the same thing, but we need to support both for backward compatibility
		annotations[istio.KubeVirtTrafficAnnotation] = defaultBridgeName
		annotations[istio.RerouteVirtualInterfacesAnnotation] = defaultBridgeName
	}

	return annotations, nil
}

// GenerateFromSource generates ordinal pod interfaces naming scheme for a migration target in case the migration source pod uses it
func (g Generator) GenerateFromSource(vmi *v1.VirtualMachineInstance, sourcePod *k8scorev1.Pod) (map[string]string, error) {
	if !namescheme.PodHasOrdinalInterfaceName(multus.NetworkStatusesFromPod(sourcePod)) {
		return nil, nil
	}

	ordinalNameScheme := namescheme.CreateOrdinalNetworkNameScheme(vmi.Spec.Networks)
	multusNetworksAnnotation, err := multus.GenerateCNIAnnotationFromNameScheme(
		vmi.Namespace,
		vmi.Spec.Domain.Devices.Interfaces,
		vmi.Spec.Networks,
		ordinalNameScheme,
		g.clusterConfigurer.GetNetworkBindings(),
	)
	if err != nil {
		return nil, err
	}

	return map[string]string{networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation}, nil
}

// GenerateFromActivePod generates additional pod annotations, bases on information that exists on a live virt-launcher pod
func (g Generator) GenerateFromActivePod(vmi *v1.VirtualMachineInstance, pod *k8scorev1.Pod) map[string]string {
	annotations := map[string]string{}

	if deviceInfoAnnotation := g.generateDeviceInfoAnnotation(vmi, pod); deviceInfoAnnotation != "" {
		annotations[downwardapi.NetworkInfoAnnot] = deviceInfoAnnotation
	}

	if updatedMultusAnnotation, shouldUpdate := g.generateMultusAnnotation(vmi, pod); shouldUpdate {
		annotations[networkv1.NetworkAttachmentAnnot] = updatedMultusAnnotation
	}

	return annotations
}

func (g Generator) generateMultusAnnotation(vmi *v1.VirtualMachineInstance, pod *k8scorev1.Pod) (string, bool) {
	vmiSpecIfaces, vmiSpecNets, ifaceChangeRequired := ifacesAndNetsForMultusAnnotationUpdate(vmi)
	if !ifaceChangeRequired {
		return "", false
	}

	podIfaceNamesByNetworkName := namescheme.CreateFromNetworkStatuses(vmiSpecNets, multus.NetworkStatusesFromPod(pod))
	updatedMultusAnnotation, err := multus.GenerateCNIAnnotationFromNameScheme(
		vmi.Namespace,
		vmiSpecIfaces,
		vmiSpecNets,
		podIfaceNamesByNetworkName,
		g.clusterConfigurer.GetNetworkBindings(),
	)
	if err != nil {
		return "", false
	}

	currentMultusNetworkSelectors, err := parseAnnotation(pod.Annotations[networkv1.NetworkAttachmentAnnot])
	if err != nil {
		// some vendors have non-JSON annotations
		// we do not support them, hence just ignore and log them
		log.Log.Errorf("failed to parse pod multus annotations: %v", err)
	}
	origMultusNetworkSelectors := deepCopy(currentMultusNetworkSelectors)

	updatedNetworkSelectors, err := parseAnnotation(updatedMultusAnnotation)
	if err != nil {
		log.Log.Errorf("failed to parse updated multus annotations: %v", err)
		return "", false
	}

	currentMultusNetworkSelectors = mergeMultusAnnotations(currentMultusNetworkSelectors, updatedNetworkSelectors)
	for _, net := range netSelectorsToRemove(podIfaceNamesByNetworkName, vmi, g.clusterConfigurer.GetNetworkBindings()) {
		delete(currentMultusNetworkSelectors, net)
	}

	if reflect.DeepEqual(currentMultusNetworkSelectors, origMultusNetworkSelectors) {
		return "", false
	}

	if len(currentMultusNetworkSelectors) == 0 {
		return "", true
	}
	jsonAnnotation, err := json.Marshal(createAnnotationFromElements(currentMultusNetworkSelectors))
	if err != nil {
		log.Log.Errorf("could not generate multus annotation string, skipping: %v", err)
		return "", false
	}
	updatedMultusAnnotation = string(jsonAnnotation)

	const logLevel = 4
	log.Log.Object(pod).V(logLevel).Infof(
		"current multus annotation for pod: %s; updated multus annotation for pod with: %s",
		pod.Annotations[networkv1.NetworkAttachmentAnnot],
		updatedMultusAnnotation,
	)

	return updatedMultusAnnotation, true
}

func mergeMultusAnnotations(
	currentMultusNetworkSelectors map[string]networkSelectionElement,
	desiredMultusAnnotation map[string]networkSelectionElement,
) map[string]networkSelectionElement {
	for key, val := range desiredMultusAnnotation {
		currentMultusNetworkSelectors[key] = val
	}
	return currentMultusNetworkSelectors
}

func parseAnnotation(s string) (map[string]networkSelectionElement, error) {
	var networkSelectorElements []networkSelectionElement
	networkSelectorElementMap := make(map[string]networkSelectionElement)

	if s == "" {
		return networkSelectorElementMap, nil
	}
	if err := json.Unmarshal([]byte(s), &networkSelectorElements); err != nil {
		return networkSelectorElementMap, err
	}

	for _, element := range networkSelectorElements {
		name, exists := element["name"]
		if !exists {
			return networkSelectorElementMap, fmt.Errorf("networkSelectorElements does not contain name: %s", s)
		}
		namespace, exists := element["namespace"]
		if !exists {
			namespace = "na"
		}
		key := fmt.Sprintf("%s/%s", name, namespace)
		networkSelectorElementMap[key] = element
	}
	return networkSelectorElementMap, nil
}

func createAnnotationFromElements(elementsMap map[string]networkSelectionElement) []networkSelectionElement {
	var elements []networkSelectionElement

	keys := make([]string, 0, len(elementsMap))
	for k := range elementsMap {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, k := range keys {
		elements = append(elements, elementsMap[k])
	}
	return elements
}

func deepCopy(src map[string]networkSelectionElement) map[string]networkSelectionElement {
	if src == nil {
		return nil
	}
	dest := make(map[string]networkSelectionElement)
	for k, v := range src {
		dest[k] = v.clone()
	}
	return dest
}

func (g Generator) generateDeviceInfoAnnotation(vmi *v1.VirtualMachineInstance, pod *k8scorev1.Pod) string {
	ifaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.SRIOV != nil || vmispec.HasBindingPluginDeviceInfo(iface, g.clusterConfigurer.GetNetworkBindings())
	})

	networkDeviceInfoMap := deviceinfo.MapNetworkNameToDeviceInfo(vmi.Spec.Networks, ifaces, multus.NetworkStatusesFromPod(pod))
	if len(networkDeviceInfoMap) == 0 {
		return ""
	}

	return downwardapi.CreateNetworkInfoAnnotationValue(networkDeviceInfoMap)
}

func shouldAddIstioKubeVirtAnnotation(vmi *v1.VirtualMachineInstance) bool {
	interfacesWithMasqueradeBinding := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.Masquerade != nil
	})

	return len(interfacesWithMasqueradeBinding) > 0
}

func ifacesAndNetsForMultusAnnotationUpdate(vmi *v1.VirtualMachineInstance) ([]v1.Interface, []v1.Network, bool) {
	vmiNonAbsentSpecIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	ifacesToHotUnplugExist := len(vmi.Spec.Domain.Devices.Interfaces) > len(vmiNonAbsentSpecIfaces)

	ifacesStatusByName := vmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces, nil)
	ifacesToAnnotate := vmispec.FilterInterfacesSpec(vmiNonAbsentSpecIfaces, func(iface v1.Interface) bool {
		_, ifaceInStatus := ifacesStatusByName[iface.Name]
		sriovIfaceNotPlugged := iface.SRIOV != nil && !ifaceInStatus
		return !sriovIfaceNotPlugged
	})

	networksToAnnotate := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, ifacesToAnnotate)

	ifacesToHotplug := vmispec.FilterInterfacesSpec(ifacesToAnnotate, func(iface v1.Interface) bool {
		_, inStatus := ifacesStatusByName[iface.Name]
		return !inStatus
	})
	ifacesToHotplugExist := len(ifacesToHotplug) > 0

	ifaceChangeRequired := ifacesToHotplugExist || ifacesToHotUnplugExist
	if !ifaceChangeRequired {
		return nil, nil, false
	}
	return ifacesToAnnotate, networksToAnnotate, ifaceChangeRequired
}

func netSelectorsToRemove(
	networkNameScheme map[string]string,
	vmi *v1.VirtualMachineInstance,
	registeredBindingPlugins map[string]v1.InterfaceBindingPlugin,
) []string {
	ifacesFromSpec := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})

	ifacesToHotUnplug := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State == v1.InterfaceStateAbsent
	})
	networksToUnplug := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, ifacesToHotUnplug)

	if len(vmi.Spec.Domain.Devices.Interfaces) > len(ifacesFromSpec) {
		netSelectorsToUnplug, err := multus.GenerateNetSelectorsForCNIAnnotation(
			vmi.Namespace,
			ifacesToHotUnplug,
			networksToUnplug,
			networkNameScheme,
			registeredBindingPlugins,
		)
		if err != nil {
			// TODO: Decide what to do in case of err
			log.Log.Errorf("failed to generate net selectors to be hot unplugged: %v", err)
			return nil
		}
		var keysToRemove []string
		for _, net := range netSelectorsToUnplug {
			namespace := net.Namespace
			if namespace == "" {
				namespace = "NA"
			}
			key := fmt.Sprintf("%s/%s", net.Name, namespace)
			keysToRemove = append(keysToRemove, key)
		}
		return keysToRemove
	}

	return []string{}
}
