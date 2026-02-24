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

package virthandler

import (
	"fmt"
	"net"

	v1 "kubevirt.io/api/core/v1"
)

// FindMigrationIP looks for dedicated migration network migration0. If found, returns its IP.
// When the interface does not exist, returns the pod IP (migrationIp) with no error.
// When the interface exists but has no usable IP, returns an error unless allowFallbackOnError
// is true (e.g. allowMigrationNetworkFallback in MigrationConfiguration), in which case
// returns the pod IP so migrations can use the pod network instead.
func FindMigrationIP(migrationIp string, allowFallbackOnError bool) (string, error) {
	ief, err := net.InterfaceByName(v1.MigrationInterfaceName)
	if err != nil {
		return migrationIp, nil
	}
	addrs, err := ief.Addrs()
	if err != nil {
		if allowFallbackOnError {
			return migrationIp, nil
		}
		return migrationIp, fmt.Errorf("%s present but doesn't have an IP", v1.MigrationInterfaceName)
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || !ipnet.IP.IsGlobalUnicast() {
			// skip local/multicast IPs
			continue
		}
		ip := ipnet.IP.To16()
		if ip != nil {
			return ip.String(), nil
		}
	}
	if allowFallbackOnError {
		return migrationIp, nil
	}
	return migrationIp, fmt.Errorf("no IP found on %s", v1.MigrationInterfaceName)
}
