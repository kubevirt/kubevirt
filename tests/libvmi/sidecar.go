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

package libvmi

import (
	"fmt"

	"kubevirt.io/kubevirt/tests/flags"
)

const (
	HookSidecarImage = "example-hook-sidecar"
)

func RenderSidecar(version string) map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars": fmt.Sprintf(
			`[{"args": ["--version", %q],"image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`,
			version,
			flags.KubeVirtUtilityRepoPrefix,
			HookSidecarImage,
			flags.KubeVirtUtilityVersionTag,
		),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
}

func RenderInvalidSMBiosSidecar() map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars": fmt.Sprintf(
			`[{"image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`,
			flags.KubeVirtUtilityRepoPrefix,
			HookSidecarImage,
			flags.KubeVirtUtilityVersionTag,
		),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
}
