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
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMIsMutator struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (mutator *VMIsMutator) Mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineInstanceGroupVersionResource.Group, webhooks.VirtualMachineInstanceGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineInstanceGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Get new VMI from admission response
	newVMI, oldVMI, err := webhookutils.GetVMIFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var patch []patchOperation

	// Patch the spec with defaults if we deal with a create operation
	if ar.Request.Operation == v1beta1.Create {
		informers := webhooks.GetInformers()

		if mutator.ClusterConfig.NonRootEnabled() {
			if newVMI.ObjectMeta.Annotations == nil {
				newVMI.ObjectMeta.Annotations = make(map[string]string)
			}
			newVMI.ObjectMeta.Annotations["nonroot"] = ""
		}

		// Apply presets
		err = applyPresets(newVMI, informers.VMIPresetInformer)
		if err != nil {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
					Code:    http.StatusUnprocessableEntity,
				},
			}
		}

		// Apply namespace limits
		applyNamespaceLimitRangeValues(newVMI, informers.NamespaceLimitsInformer)

		// Set VMI defaults
		log.Log.Object(newVMI).V(4).Info("Apply defaults")
		mutator.setDefaultCPUModel(newVMI)
		mutator.setDefaultMachineType(newVMI)
		mutator.setDefaultResourceRequests(newVMI)
		mutator.setDefaultGuestCPUTopology(newVMI)
		mutator.setDefaultPullPoliciesOnContainerDisks(newVMI)
		err = mutator.setDefaultNetworkInterface(newVMI)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
		v1.SetObjectDefaults_VirtualMachineInstance(newVMI)

		// In a future, yet undecided, release either libvirt or QEMU are going to check the hyperv dependencies, so we can get rid of this code.
		// Until that time, we need to handle the hyperv deps to avoid obscure rejections from QEMU later on
		log.Log.V(4).Info("Set HyperV dependencies")
		err = webhooks.SetVirtualMachineInstanceHypervFeatureDependencies(newVMI)
		if err != nil {
			// HyperV is a special case. If our best-effort attempt fails, we should leave
			// rejection to be performed later on in the validating webhook, and continue here.
			// Please note this means that partial changes may have been performed.
			// This is OK since each dependency must be atomic and independent (in ACID sense),
			// so the VMI configuration is still legal.
			log.Log.V(2).Infof("Failed to set HyperV dependencies: %s", err)
		}

		// Add foreground finalizer
		newVMI.Finalizers = append(newVMI.Finalizers, v1.VirtualMachineInstanceFinalizer)

		var value interface{}
		value = newVMI.Spec
		patch = append(patch, patchOperation{
			Op:    "replace",
			Path:  "/spec",
			Value: value,
		})

		value = newVMI.ObjectMeta
		patch = append(patch, patchOperation{
			Op:    "replace",
			Path:  "/metadata",
			Value: value,
		})
	} else if ar.Request.Operation == v1beta1.Update {
		// Ignore status updates if they are not coming from our service accounts
		// TODO: As soon as CRDs support field selectors we can remove this and just enable
		// the status subresource. Until then we need to update Status and Metadata labels in parallel for e.g. Migrations.
		if !reflect.DeepEqual(newVMI.Status, oldVMI.Status) {
			if !webhooks.IsKubeVirtServiceAccount(ar.Request.UserInfo.Username) {
				patch = append(patch, patchOperation{
					Op:    "replace",
					Path:  "/status",
					Value: oldVMI.Status,
				})
			}
		}

	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func (mutator *VMIsMutator) setDefaultNetworkInterface(obj *v1.VirtualMachineInstance) error {
	autoAttach := obj.Spec.Domain.Devices.AutoattachPodInterface
	if autoAttach != nil && *autoAttach == false {
		return nil
	}

	// Override only when nothing is specified
	if len(obj.Spec.Networks) == 0 && len(obj.Spec.Domain.Devices.Interfaces) == 0 {
		iface := v1.NetworkInterfaceType(mutator.ClusterConfig.GetDefaultNetworkInterface())
		switch iface {
		case v1.BridgeInterface:
			if !mutator.ClusterConfig.IsBridgeInterfaceOnPodNetworkEnabled() {
				return fmt.Errorf("Bridge interface is not enabled in kubevirt-config")
			}
			obj.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		case v1.MasqueradeInterface:
			obj.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		case v1.SlirpInterface:
			if !mutator.ClusterConfig.IsSlirpInterfaceEnabled() {
				return fmt.Errorf("Slirp interface is not enabled in kubevirt-config")
			}
			defaultIface := v1.DefaultSlirpNetworkInterface()
			obj.Spec.Domain.Devices.Interfaces = []v1.Interface{*defaultIface}
		}

		obj.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	}
	return nil
}

func (mutator *VMIsMutator) setDefaultCPUModel(vmi *v1.VirtualMachineInstance) {
	//if vmi doesn't have cpu topology or cpu model set
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		if defaultCPUModel := mutator.ClusterConfig.GetCPUModel(); defaultCPUModel != "" {
			// create cpu topology struct
			if vmi.Spec.Domain.CPU == nil {
				vmi.Spec.Domain.CPU = &v1.CPU{}
			}
			//set is as vmi cpu model
			vmi.Spec.Domain.CPU.Model = defaultCPUModel
		}
	}
}

func (mutator *VMIsMutator) setDefaultGuestCPUTopology(vmi *v1.VirtualMachineInstance) {
	cores := uint32(1)
	threads := uint32(1)
	sockets := uint32(1)
	vmiCPU := vmi.Spec.Domain.CPU
	if vmiCPU == nil || (vmiCPU.Cores == 0 && vmiCPU.Sockets == 0 && vmiCPU.Threads == 0) {
		// create cpu topology struct
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		//if cores, sockets, threads are not set, take value from domain resources request or limits and
		//set value into sockets, which have best performance (https://bugzilla.redhat.com/show_bug.cgi?id=1653453)
		resources := vmi.Spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuLimit.Value())
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuRequests.Value())
		}

		vmi.Spec.Domain.CPU.Sockets = sockets
		vmi.Spec.Domain.CPU.Cores = cores
		vmi.Spec.Domain.CPU.Threads = threads
	}
}

func (mutator *VMIsMutator) setDefaultMachineType(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Machine.Type == "" {
		vmi.Spec.Domain.Machine.Type = mutator.ClusterConfig.GetMachineType()
	}
}

func (mutator *VMIsMutator) setDefaultPullPoliciesOnContainerDisks(vmi *v1.VirtualMachineInstance) {
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil && volume.ContainerDisk.ImagePullPolicy == "" {
			if strings.HasSuffix(volume.ContainerDisk.Image, ":latest") || !strings.ContainsAny(volume.ContainerDisk.Image, ":@") {
				volume.ContainerDisk.ImagePullPolicy = k8sv1.PullAlways
			} else {
				volume.ContainerDisk.ImagePullPolicy = k8sv1.PullIfNotPresent
			}
		}
	}
}

func (mutator *VMIsMutator) setDefaultResourceRequests(vmi *v1.VirtualMachineInstance) {

	resources := &vmi.Spec.Domain.Resources

	if !resources.Limits.Cpu().IsZero() && resources.Requests.Cpu().IsZero() {
		if resources.Requests == nil {
			resources.Requests = k8sv1.ResourceList{}
		}
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
	}

	if !resources.Limits.Memory().IsZero() && resources.Requests.Memory().IsZero() {
		if resources.Requests == nil {
			resources.Requests = k8sv1.ResourceList{}
		}
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	}

	if _, exists := resources.Requests[k8sv1.ResourceMemory]; !exists {
		var memory *resource.Quantity
		if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
			memory = vmi.Spec.Domain.Memory.Guest
		}
		if memory == nil && vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
			if hugepagesSize, err := resource.ParseQuantity(vmi.Spec.Domain.Memory.Hugepages.PageSize); err == nil {
				memory = &hugepagesSize
			}
		}
		if memory != nil && memory.Value() > 0 {
			if resources.Requests == nil {
				resources.Requests = k8sv1.ResourceList{}
			}
			overcommit := mutator.ClusterConfig.GetMemoryOvercommit()
			if overcommit == 100 {
				resources.Requests[k8sv1.ResourceMemory] = *memory
			} else {
				value := (memory.Value() * int64(100)) / int64(overcommit)
				resources.Requests[k8sv1.ResourceMemory] = *resource.NewQuantity(value, memory.Format)
			}
			memoryRequest := resources.Requests[k8sv1.ResourceMemory]
			log.Log.Object(vmi).V(4).Infof("Set memory-request to %s as a result of memory-overcommit = %v%%", memoryRequest.String(), overcommit)
		}
	}

	if _, exists := resources.Requests[k8sv1.ResourceCPU]; !exists {
		if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.DedicatedCPUPlacement {
			return
		}
		if resources.Requests == nil {
			resources.Requests = k8sv1.ResourceList{}
		}
		resources.Requests[k8sv1.ResourceCPU] = *mutator.ClusterConfig.GetCPURequest()
	}
}
