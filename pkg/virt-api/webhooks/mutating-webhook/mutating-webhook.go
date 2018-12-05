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

package mutating_webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type mutateFunc func(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

func serve(resp http.ResponseWriter, req *http.Request, mutate mutateFunc) {
	response := v1beta1.AdmissionReview{}
	review, err := webhooks.GetAdmissionReview(req)

	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	reviewResponse := mutate(review)
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}
	// reset the Object and OldObject, they are not needed in a response.
	review.Request.Object = runtime.RawExtension{}
	review.Request.OldObject = runtime.RawExtension{}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Log.Reason(err).Errorf("failed json encode webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := resp.Write(responseBytes); err != nil {
		log.Log.Reason(err).Errorf("failed to write webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	resp.WriteHeader(http.StatusOK)
}

func mutateVMIs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	if ar.Request.Resource != webhooks.VirtualMachineInstanceGroupVersionResource {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineInstanceGroupVersionResource.Resource)
		return webhooks.ToAdmissionResponseError(err)
	}

	if resp := webhooks.ValidateSchema(kubev1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vmi := v1.VirtualMachineInstance{}

	err := json.Unmarshal(raw, &vmi)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	informers := webhooks.GetInformers()

	// Apply presets
	err = applyPresets(&vmi, informers.VMIPresetInformer)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusUnprocessableEntity,
			},
		}
	}

	// Apply namespace limits
	applyNamespaceLimitRangeValues(&vmi, informers.NamespaceLimitsInformer)

	// Set VMI defaults
	log.Log.Object(&vmi).V(4).Info("Apply defaults")
	kubev1.SetObjectDefaults_VirtualMachineInstance(&vmi)

	// Add foreground finalizer
	vmi.Finalizers = append(vmi.Finalizers, v1.VirtualMachineInstanceFinalizer)

	var patch []patchOperation
	var value interface{}
	value = vmi.Spec
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	value = vmi.ObjectMeta
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/metadata",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}

}

func mutatePods(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	if ar.Request.Resource.Resource != "pods" {
		err := fmt.Errorf("expect pods resource type, but received %s", ar.Request.Resource.Resource)
		return webhooks.ToAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	pod := k8sv1.Pod{}

	err := json.Unmarshal(raw, &pod)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	vmName, ok := pod.Annotations[v1.VirtualMachineWorkloadRef]
	if !ok {
		// no action required because annotation is not present
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	vmInformer := webhooks.GetInformers().VMInformer
	clientset := webhooks.GetInformers().KubeClient

	namespace := pod.Namespace
	if namespace == "" {
		namespace = "default"
	}

	key := namespace + "/" + vmName
	obj, exists, err := vmInformer.GetStore().GetByKey(key)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	} else if !exists {
		err := fmt.Errorf("VirtualMachine reference %s does not exist", key)
		return webhooks.ToAdmissionResponseError(err)
	}
	vm := obj.(*v1.VirtualMachine)

	// 1. create VMI
	vmi := v1.NewVMIReferenceFromNameWithNS(vm.ObjectMeta.Namespace, "")
	vmi.ObjectMeta = vm.Spec.Template.ObjectMeta
	if pod.Name != "" {
		vmi.ObjectMeta.Name = "vmi-" + pod.Name
	}
	if pod.GenerateName != "" {
		vmi.ObjectMeta.GenerateName = "vmi-" + pod.GenerateName
	}

	vmi.ObjectMeta.Labels = vm.Spec.Template.ObjectMeta.Labels
	vmi.ObjectMeta.Annotations = map[string]string{}
	vmi.ObjectMeta.Annotations[v1.K8sWorkloadControlled] = ""
	vmi.Spec = vm.Spec.Template.Spec

	vmi, err = clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Create(vmi)
	if err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Failed to create VirtualMachineInstance: %s/%s", vmi.Namespace, vmi.Name)
		return webhooks.ToAdmissionResponseError(err)
	}

	// 2. re-create the pod in place with right details
	// TODO make these directory and secret variables configurable.
	// TODO ensure any references datavolumes are created before allowing
	// the pod to be scheduled... basically reject the pod if datavolumes
	// don't exist yet for the vmi
	virtShareDir := "/var/run/kubevirt"
	// TODO obviously this image won't be hardcoded
	launcherImage := "registry:5000/kubevirt/virt-launcher:devel"
	templateService := services.NewTemplateService(launcherImage,
		virtShareDir,
		virtShareDir+"-ephemeral-disks",
		"",
		webhooks.GetInformers().ConfigMapInformer.GetStore(),
		webhooks.GetInformers().PVCInformer.GetStore(),
		clientset)

	templatePod, err := templateService.RenderLaunchManifest(vmi)

	// remove the Connroller boolean from the owner reference since
	// the k8s controller is technically the owner
	isController := false
	templatePod.OwnerReferences[0].Controller = &isController

	templatePod.Name = pod.Name
	templatePod.GenerateName = pod.GenerateName
	templatePod.Namespace = pod.Namespace
	templatePod.OwnerReferences = append(templatePod.OwnerReferences, pod.OwnerReferences...)

	mergedLabels := map[string]string{}
	mergedAnnotations := map[string]string{}

	for k, v := range templatePod.Labels {
		mergedLabels[k] = v
	}
	for k, v := range pod.Labels {
		mergedLabels[k] = v
	}
	for k, v := range templatePod.Annotations {
		mergedAnnotations[k] = v
	}
	for k, v := range pod.Annotations {
		mergedAnnotations[k] = v
	}

	templatePod.Labels = mergedLabels
	templatePod.Annotations = mergedAnnotations

	var patch []patchOperation
	var value interface{}
	value = templatePod.Spec
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	value = templatePod.ObjectMeta
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/metadata",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func ServePods(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, mutatePods)
}

func ServeVMIs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, mutateVMIs)
}
