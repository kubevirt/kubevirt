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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vmi

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func (c *VMIController) handleDynamicInterfaceRequests(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	podAnnotations := pod.GetAnnotations()

	indexedMultusStatusIfaces := services.NonDefaultMultusNetworksIndexedByIfaceName(pod)
	networkToPodIfaceMap := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(vmi.Spec.Networks, indexedMultusStatusIfaces)
	multusAnnotations, err := services.GenerateMultusCNIAnnotationFromNameScheme(vmi, networkToPodIfaceMap)
	if err != nil {
		return err
	}
	log.Log.Object(pod).V(4).Infof(
		"current multus annotation for pod: %s; updated multus annotation for pod with: %s",
		podAnnotations[networkv1.NetworkAttachmentAnnot],
		multusAnnotations,
	)

	if multusAnnotations != "" {
		newAnnotations := map[string]string{networkv1.NetworkAttachmentAnnot: multusAnnotations}
		patchedPod, err := c.syncPodAnnotations(pod, newAnnotations)
		if err != nil {
			return err
		}
		*pod = *patchedPod
	}

	return nil
}

func (c *VMIController) updateInterfaceStatus(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	indexedMultusStatusIfaces := services.NonDefaultMultusNetworksIndexedByIfaceName(pod)
	ifaceNamingScheme := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(vmi.Spec.Networks, indexedMultusStatusIfaces)
	for _, network := range vmi.Spec.Networks {
		vmiIfaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, network.Name)
		podIfaceName, wasFound := ifaceNamingScheme[network.Name]
		if !wasFound {
			return fmt.Errorf("could not find the pod interface name for network [%s]", network.Name)
		}

		if _, exists := indexedMultusStatusIfaces[podIfaceName]; exists {
			if vmiIfaceStatus == nil {
				vmi.Status.Interfaces = append(vmi.Status.Interfaces, virtv1.VirtualMachineInstanceNetworkInterface{
					Name:       network.Name,
					InfoSource: vmispec.InfoSourceMultusStatus,
				})
			} else {
				vmiIfaceStatus.InfoSource = vmispec.AddInfoSource(vmiIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
			}
		}
	}

	return nil
}
