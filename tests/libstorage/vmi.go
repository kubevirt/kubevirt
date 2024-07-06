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

package libstorage

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
)

func LookupVolumeTargetPath(vmi *v1.VirtualMachineInstance, volumeName string) string {
	for _, volStatus := range vmi.Status.VolumeStatus {
		if volStatus.Name == volumeName {
			return fmt.Sprintf("/dev/%s", volStatus.Target)
		}
	}

	return ""
}
