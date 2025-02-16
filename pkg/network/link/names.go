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
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const tapNameForPrimaryIface = "tap0"

func GenerateTapDeviceName(podInterfaceName string, network v1.Network) string {
	if vmispec.IsSecondaryMultusNetwork(network) {
		return "tap" + podInterfaceName[3:]
	}

	return tapNameForPrimaryIface
}

func GenerateBridgeName(podInterfaceName string) string {
	trimmedName := strings.TrimPrefix(podInterfaceName, namescheme.HashedIfacePrefix)
	return "k6t-" + trimmedName
}

func GenerateNewBridgedVmiInterfaceName(originalPodInterfaceName string) string {
	trimmedName := strings.TrimPrefix(originalPodInterfaceName, namescheme.HashedIfacePrefix)
	return fmt.Sprintf("%s-nic", trimmedName)
}
