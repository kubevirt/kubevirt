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
	var currentMultusAnnotationArr, multusAnnotationItemsToRemoveArr, multusAnnotationItemsToModifyArr, updatedMultusAnnotationArr []map[string]interface{}
	var err error

	currentMultusAnnotation := pod.Annotations[networkv1.NetworkAttachmentAnnot]
	if currentMultusAnnotationArr, err = parseAnnotation(currentMultusAnnotation); err != nil {
		log.Log.Errorf("failed to unmarshall current pod annotation: %v", err)
		panic(err)
	}

	vmiSpecIfaces, vmiSpecNets, ifaceChangeRequired := ifacesAndNetsForMultusAnnotationUpdate(vmi)
	if !ifaceChangeRequired {
		return "", false
	}
	podIfaceNamesByNetworkName := namescheme.CreateFromNetworkStatuses(vmiSpecNets, multus.NetworkStatusesFromPod(pod))

	multusAnnotationItemsToModify, err := multus.GenerateCNIAnnotationFromNameScheme(
		vmi.Namespace,
		vmiSpecIfaces,
		vmiSpecNets,
		podIfaceNamesByNetworkName,
		g.clusterConfigurer.GetNetworkBindings(),
	)
	if err != nil {
		return "", false
	}
	if multusAnnotationItemsToModifyArr, err = parseAnnotation(multusAnnotationItemsToModify); err != nil {
		log.Log.Errorf("failed to unmarshall annotation to modify: %v", err)
		panic(err)
	}

	ifacesToUnplug, nwOfIfacesToUnplug, unplugRequired := ifacesAndNetsToRemoveFromMultusAnnotationUpdate(vmi)
	if unplugRequired {
		multusAnnotationItemsToRemove, err := multus.GenerateCNIAnnotationFromNameScheme(
			vmi.Namespace,
			ifacesToUnplug,
			nwOfIfacesToUnplug,
			podIfaceNamesByNetworkName,
			g.clusterConfigurer.GetNetworkBindings(),
		)
		if err != nil {
			return "", false
		}
		if multusAnnotationItemsToRemoveArr, err = parseAnnotation(multusAnnotationItemsToRemove); err != nil {
			log.Log.Errorf("failed to unmarshall annotation to remove: %v", err)
			panic(err)
		}

		itemsToRemoveSet := makeLookupMap(multusAnnotationItemsToRemoveArr)
		for _, item := range currentMultusAnnotationArr {
			key := fmt.Sprintf("%s/%s", item["name"], item["namespace"])
			if _, exists := itemsToRemoveSet[key]; !exists {
				updatedMultusAnnotationArr = append(updatedMultusAnnotationArr, item)
			}
		}
	}

	itemsToModifyMap := makeLookupMap(multusAnnotationItemsToModifyArr)

	for index, currentItem := range updatedMultusAnnotationArr {
		key := fmt.Sprintf("%s/%s", currentItem["name"], currentItem["namespace"])
		if modifyItem, found := itemsToModifyMap[key]; found {
			for k, v := range modifyItem {
				currentItem[k] = v
			}
			delete(itemsToModifyMap, key)
		}
		updatedMultusAnnotationArr[index] = currentItem
	}
	// Add any remaining items
	for _, newItem := range itemsToModifyMap {
		updatedMultusAnnotationArr = append(updatedMultusAnnotationArr, newItem)
	}
	if len(currentMultusAnnotation) == 0 {
		updatedMultusAnnotationArr = multusAnnotationItemsToModifyArr
	}

	var updatedMultusAnnotation []byte

	//Marshall the updatedMultusAnnotationArr
	if updatedMultusAnnotationArr != nil {
		updatedMultusAnnotation, err = json.Marshal(updatedMultusAnnotationArr)
		if err != nil {
			log.Log.Errorf("failed to marshall updated multus annotation: %v", err)
			panic(err)
		}
	}
	//Compare updatedMultusAnnotation and currentMultusAnnotation
	if reflect.DeepEqual(currentMultusAnnotationArr, updatedMultusAnnotationArr) {
		return "", false
	} else {
		return string(updatedMultusAnnotation), true
	}

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

func ifacesAndNetsToRemoveFromMultusAnnotationUpdate(vmi *v1.VirtualMachineInstance) ([]v1.Interface, []v1.Network, bool) {
	vmiNonAbsentSpecIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})

	// find what to unplug
	ifacesToHotUnplug := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State == v1.InterfaceStateAbsent
	})
	networksToUnplug := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, ifacesToHotUnplug)

	ifacesToHotUnplugExist := len(vmi.Spec.Domain.Devices.Interfaces) > len(vmiNonAbsentSpecIfaces)

	return ifacesToHotUnplug, networksToUnplug, ifacesToHotUnplugExist
}

func parseAnnotation(s string) ([]map[string]interface{}, error) {
	var annotation []map[string]interface{}
	if s == "" {
		return annotation, nil
	}
	if err := json.Unmarshal([]byte(s), &annotation); err != nil {
		return nil, err
	}
	return annotation, nil
}

func makeLookupMap(annotItemsArr []map[string]interface{}) map[string]map[string]interface{} {
	// We assume that name+namespace is a unique identifier
	lookupMap := make(map[string]map[string]interface{})
	for _, item := range annotItemsArr {
		key := fmt.Sprintf("%s/%s", item["name"], item["namespace"])
		lookupMap[key] = item
	}
	return lookupMap
}
