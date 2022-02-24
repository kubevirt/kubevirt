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

package dns

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	v1 "kubevirt.io/api/core/v1"
)

// Sanitize hostname according to DNS label rules
// If the hostname is taken from vmi.Spec.Hostname
// then it already passed DNS label validations.
func SanitizeHostname(vmi *v1.VirtualMachineInstance) string {

	hostName := strings.Split(vmi.Name, ".")[0]
	if len(hostName) > validation.DNS1123LabelMaxLength {
		hostName = hostName[:validation.DNS1123LabelMaxLength]
	}
	if vmi.Spec.Hostname != "" {
		hostName = vmi.Spec.Hostname
	}

	return hostName
}
