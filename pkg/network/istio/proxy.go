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
 * Copyright 2021 Red Hat, Inc.
 *
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
