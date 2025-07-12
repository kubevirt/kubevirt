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

package annotations

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/velero"
)

type Generator struct{}

func (g Generator) Generate(vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	const computeContainerName = "compute"

	return map[string]string{
		velero.PreBackupHookContainerAnnotation: computeContainerName,
		velero.PreBackupHookCommandAnnotation: fmt.Sprintf(
			"[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", %q, \"--namespace\", %q]",
			vmi.Name,
			vmi.Namespace,
		),
		velero.PostBackupHookContainerAnnotation: computeContainerName,
		velero.PostBackupHookCommandAnnotation: fmt.Sprintf(
			"[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", %q, \"--namespace\", %q]",
			vmi.Name,
			vmi.Namespace,
		),
	}, nil
}
