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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package watch

import (
	k8sv1 "k8s.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func calculateDynamicInterfaces(vmi *v1.VirtualMachineInstance) ([]v1.Interface, []v1.Network, bool) {
	vmiSpecIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	ifacesToHotUnplugExist := len(vmi.Spec.Domain.Devices.Interfaces) > len(vmiSpecIfaces)

	vmiSpecNets := vmi.Spec.Networks
	if ifacesToHotUnplugExist {
		vmiSpecNets = vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, vmiSpecIfaces)
	}
	ifacesToHotplugExist := len(vmispec.NetworksToHotplug(vmiSpecNets, vmi.Status.Interfaces)) > 0

	isIfaceChangeRequired := ifacesToHotplugExist || ifacesToHotUnplugExist
	if !isIfaceChangeRequired {
		return nil, nil, false
	}
	return vmiSpecIfaces, vmiSpecNets, isIfaceChangeRequired
}

func trimDoneInterfaceRequests(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	if len(vm.Status.InterfaceRequests) == 0 {
		return
	}

	vmiExist := vmi != nil
	var vmiIndexedInterfaces map[string]v1.Interface
	if vmiExist {
		vmiIndexedInterfaces = vmispec.IndexInterfaceSpecByName(vmi.Spec.Domain.Devices.Interfaces)
	}

	vmIndexedInterfaces := vmispec.IndexInterfaceSpecByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces)
	updateIfaceRequests := make([]v1.VirtualMachineInterfaceRequest, 0)
	for _, request := range vm.Status.InterfaceRequests {
		removeRequest := false
		switch {
		case request.AddInterfaceOptions != nil:
			ifaceName := request.AddInterfaceOptions.Name
			_, existsInVMTemplate := vmIndexedInterfaces[ifaceName]

			if vmiExist {
				_, existsInVMISpec := vmiIndexedInterfaces[ifaceName]
				removeRequest = existsInVMTemplate && existsInVMISpec
			} else {
				removeRequest = existsInVMTemplate
			}
		case request.RemoveInterfaceOptions != nil:
			ifaceName := request.RemoveInterfaceOptions.Name
			vmIface, existsInVMTemplate := vmIndexedInterfaces[ifaceName]
			absentIfaceInVMTemplate := existsInVMTemplate && vmIface.State == v1.InterfaceStateAbsent

			if vmiExist {
				vmiIface, existsInVMISpec := vmiIndexedInterfaces[ifaceName]
				absentIfaceInVMISpec := existsInVMISpec && vmiIface.State == v1.InterfaceStateAbsent
				removeRequest = absentIfaceInVMTemplate && absentIfaceInVMISpec
			} else {
				removeRequest = absentIfaceInVMTemplate
			}
		}

		if !removeRequest {
			updateIfaceRequests = append(updateIfaceRequests, request)
		}
	}
	vm.Status.InterfaceRequests = updateIfaceRequests
}

func handleDynamicInterfaceRequests(vm *v1.VirtualMachine) {
	if len(vm.Status.InterfaceRequests) == 0 {
		return
	}

	vmTemplateCopy := vm.Spec.Template.Spec.DeepCopy()
	for i := range vm.Status.InterfaceRequests {
		vmTemplateCopy = controller.ApplyNetworkInterfaceRequestOnVMISpec(
			vmTemplateCopy, &vm.Status.InterfaceRequests[i])
	}
	vm.Spec.Template.Spec = *vmTemplateCopy

	return
}

func syncPodAnnotations(pod *k8sv1.Pod, newAnnotations map[string]string) ([]byte, error) {
	var patchOps []string
	for key, newValue := range newAnnotations {
		if podAnnotationValue, keyExist := pod.Annotations[key]; !keyExist || (keyExist && podAnnotationValue != newValue) {
			patchOp, err := prepareAnnotationsPatchAddOp(key, newValue)
			if err != nil {
				return nil, err
			}
			patchOps = append(patchOps, patchOp)
		}
	}
	return controller.GeneratePatchBytes(patchOps), nil
}

func calculateMultusAnnotation(namespace string, interfaces []v1.Interface, networks []v1.Network, pod *k8sv1.Pod) (map[string]string, error) {
	podAnnotations := pod.GetAnnotations()

	indexedMultusStatusIfaces := services.NonDefaultMultusNetworksIndexedByIfaceName(pod)
	networkToPodIfaceMap := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(networks, indexedMultusStatusIfaces)
	multusAnnotations, err := services.GenerateMultusCNIAnnotationFromNameScheme(namespace, interfaces, networks, networkToPodIfaceMap)
	if err != nil {
		return nil, err
	}
	log.Log.Object(pod).V(4).Infof(
		"current multus annotation for pod: %s; updated multus annotation for pod with: %s",
		podAnnotations[networkv1.NetworkAttachmentAnnot],
		multusAnnotations,
	)

	if multusAnnotations != "" {
		newAnnotations := map[string]string{networkv1.NetworkAttachmentAnnot: multusAnnotations}
		return newAnnotations, nil
	}
	return nil, nil
}
