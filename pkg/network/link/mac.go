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
