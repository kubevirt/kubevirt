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
 *
 */

package admitter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	v1 "kubevirt.io/api/core/v1"
)

func ValidateSlirpBinding(
	field *k8sfield.Path,
	spec *v1.VirtualMachineInstanceSpec,
	config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	for idx, ifaceSpec := range spec.Domain.Devices.Interfaces {
		if ifaceSpec.Slirp == nil {
			continue
		}
		net := vmispec.LookupNetworkByName(spec.Networks, ifaceSpec.Name)
		if net == nil {
			continue
		}

		if net.Pod == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Slirp interface only implemented with pod network",
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
			})
		} else if !config.IsSlirpInterfaceEnabled() {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Slirp interface is not enabled in kubevirt-config",
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
			})
		}
	}
	return causes
}
