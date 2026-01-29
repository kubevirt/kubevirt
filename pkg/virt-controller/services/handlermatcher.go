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

package services

import (
	v1 "kubevirt.io/api/core/v1"
)

// MatchVMIToAdditionalHandler finds an additional virt-handler configuration
// that matches the VMI's node selector. Returns the matching handler config
// or nil if no match is found.
//
// A match occurs when the VMI's node selector contains all the key-value pairs
// from an additional handler's node placement nodeSelector. This means the VMI
// is explicitly targeting nodes that would be served by that additional handler.
func MatchVMIToAdditionalHandler(vmi *v1.VirtualMachineInstance, additionalHandlers []v1.AdditionalVirtHandlerConfig) *v1.AdditionalVirtHandlerConfig {
	if vmi == nil || len(additionalHandlers) == 0 {
		return nil
	}

	vmiNodeSelector := vmi.Spec.NodeSelector
	if len(vmiNodeSelector) == 0 {
		return nil
	}

	for i := range additionalHandlers {
		handler := &additionalHandlers[i]
		if len(handler.NodeSelector) == 0 {
			continue
		}

		if nodeSelectorMatches(vmiNodeSelector, handler.NodeSelector) {
			return handler
		}
	}

	return nil
}

// nodeSelectorMatches checks if the VMI's node selector contains all key-value
// pairs from the handler's node selector. This indicates the VMI is targeting
// nodes that would be served by that handler.
func nodeSelectorMatches(vmiSelector, handlerSelector map[string]string) bool {
	for key, handlerValue := range handlerSelector {
		if vmiValue, exists := vmiSelector[key]; !exists || vmiValue != handlerValue {
			return false
		}
	}
	return true
}

// GetLauncherImageForVMI determines the appropriate virt-launcher image for a VMI.
// If the VMI's node selector matches an additional handler with a custom virt-launcher
// image, that image is returned. Otherwise, the default image is returned.
func GetLauncherImageForVMI(vmi *v1.VirtualMachineInstance, additionalHandlers []v1.AdditionalVirtHandlerConfig, defaultImage string) string {
	handler := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
	if handler != nil && handler.VirtLauncherImage != "" {
		return handler.VirtLauncherImage
	}
	return defaultImage
}
