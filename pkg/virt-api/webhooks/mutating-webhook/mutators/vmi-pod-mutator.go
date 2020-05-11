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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package mutators

import (
	"encoding/json"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/client-go/api/v1"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

type VMIPodsMutator struct {
	ClusterConfig   *virtconfig.ClusterConfig
	VirtCli         kubecli.KubevirtClient
	LauncherVersion string
}

func (mutator *VMIPodsMutator) Mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	raw := ar.Request.Object.Raw
	pod := k8sv1.Pod{}

	err := json.Unmarshal(raw, &pod)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal vmi pod")
		return webhookutils.ToAdmissionResponseError(err)
	}

	if pod.Annotations == nil {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	vmiSpecJSON, ok := pod.Annotations["kubevirt.io/vmi-spec-json"]
	if !ok {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	vmi := &v1.VirtualMachineInstance{}
	// Make sure someone can't escape their namespace
	vmi.Name = pod.Name
	vmi.GenerateName = pod.GenerateName
	vmi.Namespace = pod.Namespace
	if vmi.Namespace == "" {
		// TODO use const for this
		vmi.Namespace = "default"
	}
	err = json.Unmarshal([]byte(vmiSpecJSON), vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal vmi json in pod %s/%s", pod.Namespace, pod.Name)
		return webhookutils.ToAdmissionResponseError(err)
	}

	// TODO VMI needs to delete itself when pod is deleted
	// TODO VMI needs to timeout and delete itself if pod doesn't show up after ~60 seconds
	if vmi.Annotations == nil {
		vmi.Annotations = make(map[string]string)
	}
	vmi.Annotations["kubevirt.io/ignore-pod-create"] = ""

	vmi, err = mutator.VirtCli.VirtualMachineInstance(vmi.Namespace).Create(vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to create vmi for pod %s/%s with uid %s", pod.Namespace, pod.Name, string(pod.UID))
		return webhookutils.ToAdmissionResponseError(err)
	}

	// TODO these values need to be set like they are in virt-controller via cli args otherwise
	// options like pull secret won't be set correctly.
	templateService := services.NewTemplateService(mutator.LauncherVersion,
		"/var/run/kubevirt",
		"/var/lib/kubevirt",
		"/var/run/kubevirt-ephemeral-disks",
		"/var/run/kubevirt/container-disks",
		"",
		nil, // TODO send in a real pvc cache informer here, otherwise this will crash if PVC or DataVolume is used
		mutator.VirtCli,
		mutator.ClusterConfig,
		107,
	)

	newPod, err := templateService.RenderLaunchManifest(vmi)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	newPod.Name = pod.Name
	newPod.Namespace = pod.Namespace
	newPod.GenerateName = pod.GenerateName

	var patch []patchOperation
	var value interface{}
	value = newPod.Spec
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	value = newPod.ObjectMeta
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/metadata",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to render vmi pod's patch %s/%s", pod.Namespace, pod.Name)
		return webhookutils.ToAdmissionResponseError(err)
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}
