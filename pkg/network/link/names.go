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

package link

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

func GenerateTapDeviceName(podInterfaceName string) string {
	if namescheme.OrdinalInterfaceName(podInterfaceName) {
		return "tap" + podInterfaceName[3:]
	}

	return "tap" + podInterfaceName[:11]
}

func GenerateBridgeName(podInterfaceName string) string {
	if namescheme.OrdinalInterfaceName(podInterfaceName) {
		return "k6t-" + podInterfaceName
	}

	return "k6t-" + podInterfaceName[3:]

}

func GenerateNewBridgedVmiInterfaceName(originalPodInterfaceName string) string {
	if namescheme.OrdinalInterfaceName(originalPodInterfaceName) {
		return fmt.Sprintf("%s-nic", originalPodInterfaceName)
	}

	return fmt.Sprintf("%s-nic", originalPodInterfaceName[3:])
}
