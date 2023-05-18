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

package admitters

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/resource"
)

const (
	ifaceRequestMinValue = 0
	ifaceRequestMaxValue = 32
)

func validateInterfaceRequestIsInRange(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	requests := resource.ExtendedResourceList{ResourceList: spec.Domain.Resources.Requests}
	value := requests.Interface().Value()

	if !isIntegerValue(value, requests) || value < ifaceRequestMinValue || value > ifaceRequestMaxValue {
		causes = []metav1.StatusCause{{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(
				"provided resources interface requests must be an integer between %d to %d",
				ifaceRequestMinValue,
				ifaceRequestMaxValue,
			),
			Field: field.Child("domain", "resources", "requests", string(resource.ResourceInterface)).String(),
		}}
	}
	return causes
}

func isIntegerValue(value int64, requests resource.ExtendedResourceList) bool {
	return value*1000 == requests.Interface().MilliValue()
}
