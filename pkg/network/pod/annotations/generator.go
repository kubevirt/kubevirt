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
		annotations[istio.KubeVirtTrafficAnnotation] = defaultBridgeName
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

	// TODO: Check what happens if networkv1.NetworkAttachmentAnnot doesnt exist
	currentMultusNetSel, err := parseAnnotation(pod.Annotations[networkv1.NetworkAttachmentAnnot])
	if err != nil {
		// TODO: decide what to do
		log.Log.Errorf("failed to parse pod multus annotations: %v", err)
	}
	origMultusNetSel := deepCopy(currentMultusNetSel)

	updatedNetSel, err := parseAnnotation(updatedMultusAnnotation)
	if err != nil {
		log.Log.Errorf("failed to parse updated multus annotations: %v", err)
	}

	currentMultusNetSel = mergeMultusAnnotations(currentMultusNetSel, updatedNetSel)
	for _, net := range netSelectorsToRemoveFromMultusAnnotation(
		podIfaceNamesByNetworkName,
		vmi,
		g.clusterConfigurer.GetNetworkBindings()) {
		delete(currentMultusNetSel, net)
	}

	if reflect.DeepEqual(currentMultusNetSel, origMultusNetSel) {
		return "", false
	}

	jsonAnnotation, err := json.Marshal(createAnnotationFromElements(currentMultusNetSel))
	if err != nil {
		log.Log.Errorf("could not generate multus annotation string, skipping: %v", err)
		return "", false
	}
	if string(jsonAnnotation) != "null" {
		updatedMultusAnnotation = string(jsonAnnotation)
	} else {
		updatedMultusAnnotation = ""
	}

	const logLevel = 4
	log.Log.Object(pod).V(logLevel).Infof(
		"current multus annotation for pod: %s; updated multus annotation for pod with: %s",
		pod.Annotations[networkv1.NetworkAttachmentAnnot],
		updatedMultusAnnotation,
	)

	return updatedMultusAnnotation, true
}

func mergeMultusAnnotations(
	currentMultusNetSel map[string]map[string]interface{},
	desiredMultusAnnotation map[string]map[string]interface{},
) map[string]map[string]interface{} {
	for key, val := range desiredMultusAnnotation {
		// TODO: Do we want to preserve extra fields added to desiredNeSel by third party
		currentMultusNetSel[key] = val
	}
	return currentMultusNetSel
}

func parseAnnotation(s string) (map[string]map[string]interface{}, error) {
	var netSelElements []map[string]interface{}
	if s == "" {
		return map[string]map[string]interface{}{}, nil
	}
	if err := json.Unmarshal([]byte(s), &netSelElements); err != nil {
		return map[string]map[string]interface{}{}, err
	}

	netSelElementMap := make(map[string]map[string]interface{})
	for _, element := range netSelElements {
		name, exists := element["name"]
		if !exists {
			return map[string]map[string]interface{}{}, fmt.Errorf("netSelElements does not contain name: %s", s)
		}
		namespace, exists := element["namespace"]
		if !exists {
			namespace = "na"
		}
		// TODO: externalize this if replicated
		key := fmt.Sprintf("%s/%s", name, namespace)
		netSelElementMap[key] = element
	}
	return netSelElementMap, nil
}

func createAnnotationFromElements(elementsMap map[string]map[string]interface{}) []map[string]interface{} {
	var elements []map[string]interface{}
	for _, v := range elementsMap {
		elements = append(elements, v)
	}
	return elements
}

func deepCopy(src map[string]map[string]interface{}) map[string]map[string]interface{} {
	if src == nil {
		return nil
	}
	dest := make(map[string]map[string]interface{})
	for k, v := range src {
		if v == nil {
			dest[k] = nil
			continue
		}
		innerMap := make(map[string]interface{})
		for key, val := range v {
			innerMap[key] = val
		}
		dest[k] = innerMap
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

func netSelectorsToRemoveFromMultusAnnotation(
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
