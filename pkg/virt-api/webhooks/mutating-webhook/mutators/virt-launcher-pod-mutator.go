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
 */

package mutators

import (
	"encoding/json"
	"fmt"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"net"
	"slices"
)

const MultusCustomizationAnnotation = "k8s.v1.cni.cncf.io/networks-customization"

type VirtLauncherPodsMutator struct {
	ClusterConfig           *virtconfig.ClusterConfig
	VirtLauncherPodInformer cache.SharedIndexInformer
}

func (mutator *VirtLauncherPodsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, "", "pods") {
		err := fmt.Errorf("expect resource to be '%s'", "pods")
		return webhookutils.ToAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}

	err := json.Unmarshal(raw, &pod)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if value, exists := pod.Labels[kubevirtv1.AppLabel]; !exists || value != "virt-launcher" {
		err := fmt.Errorf("admission failed, pod '%s' is not a virt-launcher instance", pod.Name)
		return webhookutils.ToAdmissionResponseError(err)
	}

	rawKubevirtMultusAnnotation := pod.Annotations[networkv1.NetworkAttachmentAnnot]
	if rawKubevirtMultusAnnotation == "" {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}
	kubevirtMultusAnnotation, err := deserializeMultusAnnotation(rawKubevirtMultusAnnotation)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	rawUserMultusAnnotation := pod.Annotations[MultusCustomizationAnnotation]
	if rawUserMultusAnnotation == "" {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}
	userMultusAnnotation, err := deserializeMultusAnnotation(rawUserMultusAnnotation)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	newMultusAnnotation, err := enrichMultusAnnotation(kubevirtMultusAnnotation, userMultusAnnotation)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	serializedMultusAnnotation, err := json.Marshal(newMultusAnnotation)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	patchType := admissionv1.PatchTypeJSONPatch
	patchBytes, err := patch.GeneratePatchPayload(
		patch.PatchOperation{
			Op:    patch.PatchReplaceOp,
			Path:  "/metadata/annotations/k8s.v1.cni.cncf.io~1networks",
			Value: string(serializedMultusAnnotation),
		},
	)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
}

func enrichMultusAnnotation(kubevirtNetworks, userNetworks []networkv1.NetworkSelectionElement) ([]networkv1.NetworkSelectionElement, error) {
	networks := make([]networkv1.NetworkSelectionElement, 0)

	for _, kubevirtNetwork := range kubevirtNetworks {
		kubevirtNetworkKey := networkKey(kubevirtNetwork)
		userIndex := slices.IndexFunc(userNetworks, func(userNetwork networkv1.NetworkSelectionElement) bool {
			return kubevirtNetworkKey == networkKey(userNetwork)
		})

		if userIndex == -1 {
			networks = append(networks, kubevirtNetwork)
			continue
		}

		userNetwork := userNetworks[userIndex]
		mergedNetwork, err := mergeNetworks(kubevirtNetwork, userNetwork)
		if err != nil {
			return nil, err
		}
		networks = append(networks, *mergedNetwork)
	}

	return networks, nil
}

func mergeNetworks(kubevirtNetwork, userNetwork networkv1.NetworkSelectionElement) (*networkv1.NetworkSelectionElement, error) {
	if userNetwork.InterfaceRequest != "" {
		return nil, fmt.Errorf("interface name can't be modified for '%s' - value is managed by virt-controller", kubevirtNetwork.Name)
	}
	if userNetwork.MacRequest != "" {
		return nil, fmt.Errorf("mac address should be set for '%s' through the interface definition in the VMI spec", kubevirtNetwork.Name)
	}
	if len(userNetwork.IPRequest) > 0 {
		if kubevirtNetwork.IPRequest == nil {
			kubevirtNetwork.IPRequest = []string{}
		}
		kubevirtNetwork.IPRequest = append(kubevirtNetwork.IPRequest, userNetwork.IPRequest...)
	}
	if len(userNetwork.GatewayRequest) > 0 {
		if kubevirtNetwork.GatewayRequest == nil {
			kubevirtNetwork.GatewayRequest = []net.IP{}
		}
		kubevirtNetwork.GatewayRequest = append(kubevirtNetwork.GatewayRequest, userNetwork.GatewayRequest...)
	}
	if len(userNetwork.PortMappingsRequest) > 0 {
		if kubevirtNetwork.PortMappingsRequest == nil {
			kubevirtNetwork.PortMappingsRequest = []*networkv1.PortMapEntry{}
		}
		kubevirtNetwork.PortMappingsRequest = append(kubevirtNetwork.PortMappingsRequest, userNetwork.PortMappingsRequest...)
	}
	if userNetwork.BandwidthRequest != nil {
		kubevirtNetwork.BandwidthRequest = userNetwork.BandwidthRequest
	}
	if userNetwork.InfinibandGUIDRequest != "" {
		kubevirtNetwork.InfinibandGUIDRequest = userNetwork.InfinibandGUIDRequest
	}
	if userNetwork.CNIArgs != nil && len(*userNetwork.CNIArgs) > 0 {
		kubevirtNetwork.CNIArgs = userNetwork.CNIArgs
	}

	return &kubevirtNetwork, nil
}

func networkKey(element networkv1.NetworkSelectionElement) string {
	if element.Namespace != "" {
		return element.Namespace + "/" + element.Name
	}
	return element.Name
}

func deserializeMultusAnnotation(payload string) ([]networkv1.NetworkSelectionElement, error) {
	var multusAnnotation []networkv1.NetworkSelectionElement
	if err := json.Unmarshal([]byte(payload), &multusAnnotation); err != nil {
		return nil, fmt.Errorf("failed to deserialize multus annotation: %v", err)
	}

	return multusAnnotation, nil
}
