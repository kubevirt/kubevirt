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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/cache"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

func applyNamespaceLimitRangeValues(vmi *kubev1.VirtualMachineInstance, limitrangeInformer cache.SharedIndexInformer) {
	isMemoryFieldExist := func(resource k8sv1.ResourceList) bool {
		_, ok := resource[k8sv1.ResourceMemory]
		return ok
	}

	vmiResources := vmi.Spec.Domain.Resources

	// Copy namespace memory limits (if exists) to the VM spec
	if vmiResources.Limits == nil || !isMemoryFieldExist(vmiResources.Limits) {

		namespaceMemLimit, err := getNamespaceLimits(vmi.Namespace, limitrangeInformer)
		if err == nil && !namespaceMemLimit.IsZero() {
			if vmiResources.Limits == nil {
				vmi.Spec.Domain.Resources.Limits = make(k8sv1.ResourceList)
			}
			log.Log.Object(vmi).V(4).Info("Apply namespace limits")
			vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] = *namespaceMemLimit
		}
	}

}

func getNamespaceLimits(namespace string, limitrangeInformer cache.SharedIndexInformer) (*resource.Quantity, error) {
	finalLimit := &resource.Quantity{Format: resource.BinarySI}

	// there can be multiple LimitRange values set for the same resource in
	// a namespace, we need to find the minimal
	limits, err := limitrangeInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	for _, limit := range limits {
		for _, val := range limit.(*k8sv1.LimitRange).Spec.Limits {
			mem := val.Default.Memory()
			if val.Type == k8sv1.LimitTypeContainer {
				if !mem.IsZero() {
					if finalLimit.IsZero() != (mem.Cmp(*finalLimit) < 0) {
						finalLimit = mem
					}
				}
			}
		}
	}
	return finalLimit, nil
}
