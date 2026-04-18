/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virthandler

import (
	"fmt"
	"net"

	v1 "kubevirt.io/api/core/v1"
)

// FindMigrationIP looks for dedicated migration network migration0. If found, sets migration IP to it
func FindMigrationIP(migrationIp string) (string, error) {
	ief, err := net.InterfaceByName(v1.MigrationInterfaceName)
	if err != nil {
		return migrationIp, nil
	}
	addrs, err := ief.Addrs()
	if err != nil { // get addresses
		return migrationIp, fmt.Errorf("%s present but doesn't have an IP", v1.MigrationInterfaceName)
	}
	for _, addr := range addrs {
		if !addr.(*net.IPNet).IP.IsGlobalUnicast() {
			// skip local/multicast IPs
			continue
		}
		ip := addr.(*net.IPNet).IP.To16()
		if ip != nil {
			return ip.String(), nil
		}
	}

	return migrationIp, fmt.Errorf("no IP found on %s", v1.MigrationInterfaceName)
}
