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
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMIsMutator struct {
	ClusterConfig           *virtconfig.ClusterConfig
	VMIPresetInformer       cache.SharedIndexInformer
	NamespaceLimitsInformer cache.SharedIndexInformer
}

func (mutator *VMIsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
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

	var patch []utiltypes.PatchOperation

	// Patch the spec, metadata and status with defaults if we deal with a create operation
	if ar.Request.Operation == admissionv1.Create {
		// Apply presets
		err = applyPresets(newVMI, mutator.VMIPresetInformer)
		if err != nil {
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
					Code:    http.StatusUnprocessableEntity,
				},
			}
		}

		// Apply namespace limits
		applyNamespaceLimitRangeValues(newVMI, mutator.NamespaceLimitsInformer)

		// Set VMI defaults
		log.Log.Object(newVMI).V(4).Info("Apply defaults")
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

		// Do some specific setting for Arm64 Arch. It should put before SetObjectDefaults_VirtualMachineInstance
		if webhooks.IsARM64() {
			log.Log.V(4).Info("Apply Arm64 specific setting")
			err = webhooks.SetVirtualMachineInstanceArm64Defaults(newVMI)
			if err != nil {
				// if SetVirtualMachineInstanceArm64Defaults fails, it's due to a validation error, which will get caught in the validation webhook after mutation finishes.
				log.Log.V(2).Infof("Failed to setting for Arm64: %s", err)
			}
		} else {
			mutator.setDefaultCPUModel(newVMI)
		}
		if newVMI.IsRealtimeEnabled() {
			log.Log.V(4).Info("Add realtime node label selector")
			addRealtimeNodeSelector(newVMI)
		}

		// Add foreground finalizer
		newVMI.Finalizers = append(newVMI.Finalizers, v1.VirtualMachineInstanceFinalizer)

		// Set the phase to pending to avoid blank status
		newVMI.Status.Phase = v1.Pending

		now := metav1.NewTime(time.Now())
		newVMI.Status.PhaseTransitionTimestamps = append(newVMI.Status.PhaseTransitionTimestamps, v1.VirtualMachineInstancePhaseTransitionTimestamp{
			Phase:                    newVMI.Status.Phase,
			PhaseTransitionTimestamp: now,
		})

		if mutator.ClusterConfig.NonRootEnabled() {
			if err := canBeNonRoot(newVMI); err != nil {
				return &admissionv1.AdmissionResponse{
					Result: &metav1.Status{
						Message: err.Error(),
						Code:    http.StatusUnprocessableEntity,
					},
				}
			} else {
				if newVMI.ObjectMeta.Annotations == nil {
					newVMI.ObjectMeta.Annotations = make(map[string]string)
				}
				newVMI.ObjectMeta.Annotations[v1.NonRootVMIAnnotation] = ""
			}
		}

		var value interface{}
		value = newVMI.Spec
		patch = append(patch, utiltypes.PatchOperation{
			Op:    "replace",
			Path:  "/spec",
			Value: value,
		})

		value = newVMI.ObjectMeta
		patch = append(patch, utiltypes.PatchOperation{
			Op:    "replace",
			Path:  "/metadata",
			Value: value,
		})

		value = newVMI.Status
		patch = append(patch, utiltypes.PatchOperation{
			Op:    "replace",
			Path:  "/status",
			Value: value,
		})

	} else if ar.Request.Operation == admissionv1.Update {
		// Ignore status updates if they are not coming from our service accounts
		// TODO: As soon as CRDs support field selectors we can remove this and just enable
		// the status subresource. Until then we need to update Status and Metadata labels in parallel for e.g. Migrations.
		if !reflect.DeepEqual(newVMI.Status, oldVMI.Status) {
			if !webhooks.IsKubeVirtServiceAccount(ar.Request.UserInfo.Username) {
				patch = append(patch, utiltypes.PatchOperation{
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

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
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
	// create cpu topology struct
	if vmi.Spec.Domain.CPU == nil {
		vmi.Spec.Domain.CPU = &v1.CPU{}
	}

	// if vmi doesn't have cpu model set
	if vmi.Spec.Domain.CPU.Model == "" {
		if clusterConfigCPUModel := mutator.ClusterConfig.GetCPUModel(); clusterConfigCPUModel != "" {
			//set is as vmi cpu model
			vmi.Spec.Domain.CPU.Model = clusterConfigCPUModel
		} else {
			vmi.Spec.Domain.CPU.Model = v1.DefaultCPUModel
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
	machineType := mutator.ClusterConfig.GetMachineType()

	if machine := vmi.Spec.Domain.Machine; machine != nil {
		if machine.Type == "" {
			machine.Type = machineType
		}
	} else {
		vmi.Spec.Domain.Machine = &v1.Machine{Type: machineType}
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
	if cpuRequest := mutator.ClusterConfig.GetCPURequest(); !cpuRequest.Equal(resource.MustParse(virtconfig.DefaultCPURequest)) {
		if _, exists := resources.Requests[k8sv1.ResourceCPU]; !exists {
			if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.DedicatedCPUPlacement {
				return
			}
			if resources.Requests == nil {
				resources.Requests = k8sv1.ResourceList{}
			}
			resources.Requests[k8sv1.ResourceCPU] = *cpuRequest
		}
	}

}

func canBeNonRoot(vmi *v1.VirtualMachineInstance) error {
	// VirtioFS doesn't work with session mode
	if util.IsVMIVirtiofsEnabled(vmi) {
		return fmt.Errorf("VirtioFS doesn't work with session mode(used by nonroot)")
	}

	if util.IsSRIOVVmi(vmi) {
		return fmt.Errorf("SRIOV doesn't work with nonroot")
	}
	return nil
}

// AddRealtimeNodeSelector adds the realtime node selector
func addRealtimeNodeSelector(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.NodeSelector == nil {
		vmi.Spec.NodeSelector = map[string]string{}
	}
	vmi.Spec.NodeSelector[v1.RealtimeLabel] = ""
}
