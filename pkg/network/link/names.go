/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
