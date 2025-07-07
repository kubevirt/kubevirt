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

package device

import (
	"os"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
)

func USBDevicesFound(vmiHostDevices []v1.HostDevice) bool {
	for _, device := range vmiHostDevices {
		env := util.ResourceNameToEnvVar(v1.USBResourcePrefix, device.DeviceName)
		if _, ok := os.LookupEnv(env); ok {
			return true
		}
	}
	return false
}
