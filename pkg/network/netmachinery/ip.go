/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package netmachinery

import (
	"net"
)

// NextIP increments the IP address by one (in-place).
func NextIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
