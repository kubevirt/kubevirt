/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package link

import (
	"fmt"
	"net"
)

const StaticMasqueradeBridgeMAC = "02:00:00:00:00:00"

func IsReserved(mac string) bool {
	return mac == StaticMasqueradeBridgeMAC
}

// ValidateMacAddress performs a validation of the address validity in terms of format and size.
// An empty mac address input is ignored (i.e. is considered valid).
func ValidateMacAddress(macAddress string) error {
	if macAddress == "" {
		return nil
	}
	mac, err := net.ParseMAC(macAddress)
	if err != nil {
		return fmt.Errorf("malformed MAC address (%s)", macAddress)
	}
	const macLen = 6
	if len(mac) > macLen {
		return fmt.Errorf("too long MAC address (%s)", macAddress)
	}
	return nil
}
