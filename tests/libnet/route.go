/*
 * This file is part of the kubevirt project
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

package libnet

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
)

var neighValidStateRegex = regexp.MustCompile(`REACHABLE|STALE|DELAY|PROBE`)

const (
	defaultRoute = "default"
	destination  = "dst"
	gateway      = "gateway"
)

func HasDefaultRoute(vmi *v1.VirtualMachineInstance, ipFamily k8sv1.IPFamily, timeout time.Duration) bool {
	if ipFamily == k8sv1.IPv6Protocol {
		return hasDefaultRouteIPv6(vmi, timeout)
	}
	return hasDefaultRouteIPv4(vmi, timeout)
}

func hasDefaultRouteIPv6(vmi *v1.VirtualMachineInstance, timeout time.Duration) bool {
	const routeCmd = "ip -json -6 route show default"

	routes, err := runJSONConsoleCommand(vmi, routeCmd, timeout)
	if err != nil {
		return false
	}

	for _, route := range routes {
		dst, dstOk := route[destination]
		_, gatewayOk := route[gateway]
		if dstOk && dst == defaultRoute && gatewayOk {
			return hasNeighbor(vmi, timeout)
		}
	}

	return false
}

func hasNeighbor(vmi *v1.VirtualMachineInstance, timeout time.Duration) bool {
	const neighCmd = "ip -6 -json neigh show"
	neighbors, err := runJSONConsoleCommand(vmi, neighCmd, timeout)
	if err != nil {
		return false
	}
	for _, neigh := range neighbors {
		stateVal, ok := neigh["state"]
		if !ok {
			continue
		}
		stateSlice, ok := stateVal.([]interface{})
		if !ok {
			continue
		}
		var parts []string
		for _, s := range stateSlice {
			if str, ok := s.(string); ok {
				parts = append(parts, str)
			}
		}
		if neighValidStateRegex.MatchString(strings.Join(parts, " ")) {
			return true
		}
	}
	return false
}

func hasDefaultRouteIPv4(vmi *v1.VirtualMachineInstance, timeout time.Duration) bool {
	const routeCmd = "ip -json route show default"
	routes, err := runJSONConsoleCommand(vmi, routeCmd, timeout)
	if err != nil {
		return false
	}
	for _, route := range routes {
		dst, dstOk := route[destination]
		_, gatewayOk := route[gateway]
		if dstOk && dst == defaultRoute && gatewayOk {
			return true
		}
	}
	return false
}

func runJSONConsoleCommand(vmi *v1.VirtualMachineInstance, command string, timeout time.Duration) ([]map[string]interface{}, error) {
	output, err := console.RunCommandAndStoreOutput(vmi, command, timeout)
	if err != nil {
		return []map[string]interface{}{}, err
	}
	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return []map[string]interface{}{}, err
	}
	return list, nil
}
