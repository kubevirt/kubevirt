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

package dns

import (
	"strings"

	"github.com/openshift/library-go/pkg/build/naming"
	"k8s.io/apimachinery/pkg/util/validation"

	v1 "kubevirt.io/api/core/v1"
)

const (
	// AccessCredentialsSuffix is the suffix used for access credentials volume names
	// to ensure they remain DNS-1123 compliant even when the secret name is long
	AccessCredentialsSuffix = "access-cred"
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

// SanitizeAccessCredentialVolumeName ensures the volume name conforms to DNS-1123 label standard.
// It uses [naming.GetName] to properly handle length constraints while preserving
// the AccessCredentialsSuffix, ensuring consistency with getSecretDir.
func SanitizeAccessCredentialVolumeName(secretName string) string {
	return naming.GetName(secretName, AccessCredentialsSuffix, validation.DNS1123LabelMaxLength)
}
