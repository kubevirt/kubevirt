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
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

func applyNamespaceLimitRangeValues(vmi *kubev1.VirtualMachineInstance, limitrangeInformer cache.SharedIndexInformer) {
	isResourceRequirementMissing := func(vmiResources kubev1.ResourceRequirements) bool {
		if vmiResources.Limits == nil || vmiResources.Requests == nil {
			return true
		}
		isMemoryAndCpuExist := func(resource k8sv1.ResourceList) bool {
			for _, v := range []k8sv1.ResourceName{k8sv1.ResourceMemory, k8sv1.ResourceCPU} {
				if _, ok := resource[v]; !ok {
					return false
				}
			}
			return true
		}
		if !isMemoryAndCpuExist(vmiResources.Limits) || !isMemoryAndCpuExist(vmiResources.Requests) {
			return true
		}
		return false
	}

	// Copy namespace limits (if exist) to the VM spec
	if isResourceRequirementMissing(vmi.Spec.Domain.Resources) {
		limits, err := limitrangeInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vmi.Namespace)
		if err != nil {
			return
		}

		log.Log.Object(vmi).V(4).Info("Apply namespace limits")
		for _, limit := range limits {
			defaultRequirements := defaultVMIResourceRequirements(limit.(*k8sv1.LimitRange))
			mergeVMIResources(vmi, &defaultRequirements)
		}
	}
}

// See mergeContainerResources in https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/admission/limitranger/admission.go
func mergeVMIResources(vmi *kubev1.VirtualMachineInstance, defaultRequirements *k8sv1.ResourceRequirements) {
	if vmi.Spec.Domain.Resources.Limits == nil {
		vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
	}
	if vmi.Spec.Domain.Resources.Requests == nil {
		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
	}
	// TODO: generate annotations like limitranger admission-plugin in kubernetes
	for k, v := range defaultRequirements.Limits {
		_, found := vmi.Spec.Domain.Resources.Limits[k]
		if !found {
			vmi.Spec.Domain.Resources.Limits[k] = *v.Copy()
		}
	}
	for k, v := range defaultRequirements.Requests {
		_, found := vmi.Spec.Domain.Resources.Requests[k]
		if !found {
			vmi.Spec.Domain.Resources.Requests[k] = *v.Copy()
		}
	}
}

// See defaultContainerResourceRequirements in https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/admission/limitranger/admission.go
func defaultVMIResourceRequirements(limitRange *k8sv1.LimitRange) k8sv1.ResourceRequirements {
	requirements := k8sv1.ResourceRequirements{}
	requirements.Requests = k8sv1.ResourceList{}
	requirements.Limits = k8sv1.ResourceList{}

	for i := range limitRange.Spec.Limits {
		limit := limitRange.Spec.Limits[i]
		if limit.Type == k8sv1.LimitTypeContainer {
			for k, v := range limit.DefaultRequest {
				value := v.Copy()
				requirements.Requests[k8sv1.ResourceName(k)] = *value
			}
			for k, v := range limit.Default {
				value := v.Copy()
				requirements.Limits[k8sv1.ResourceName(k)] = *value
			}
		}
	}
	return requirements
}
