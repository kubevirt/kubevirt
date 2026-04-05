/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package istio

import (
	"strings"

	v1 "kubevirt.io/api/core/v1"
)

func ProxyInjectionEnabled(vmi *v1.VirtualMachineInstance) bool {
	if val, ok := vmi.GetAnnotations()[InjectSidecarAnnotation]; ok {
		return strings.EqualFold(val, "true")
	}
	return false
}

func GetLoopbackAddress() string {
	return "127.0.0.6"
}
