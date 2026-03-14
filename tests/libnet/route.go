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
	"fmt"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
)

type Route struct {
	Dst     string `json:"dst,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	Dev     string `json:"dev,omitempty"`
}

type Neighbor struct {
	Dst   string   `json:"dst,omitempty"`
	Dev   string   `json:"dev,omitempty"`
	State []string `json:"state,omitempty"`
}

const defaultRoute = "default"

func HasDefaultRoute(vmi *v1.VirtualMachineInstance, ipFamily k8sv1.IPFamily, timeout time.Duration) bool {
	if ipFamily == k8sv1.IPv6Protocol {
		return hasDefaultRouteIPv6AndNeigh(vmi, timeout)
	}
	return hasDefaultRouteIPv4(vmi, timeout)
}

func hasDefaultRouteIPv4(vmi *v1.VirtualMachineInstance, timeout time.Duration) bool {
	const ipv4FamilyArg = "-4"
	routes, err := queryDefaultRoutes(vmi, ipv4FamilyArg, timeout)
	if err != nil {
		return false
	}

	for _, route := range routes {
		if (route.Dst == defaultRoute) && route.Gateway != "" {
			return true
		}
	}
	return false
}

func hasDefaultRouteIPv6AndNeigh(vmi *v1.VirtualMachineInstance, timeout time.Duration) bool {
	const ipv6FamilyArg = "-6"
	routes, err := queryDefaultRoutes(vmi, ipv6FamilyArg, timeout)
	if err != nil {
		return false
	}

	for _, route := range routes {
		if (route.Dst == defaultRoute) && route.Gateway != "" {
			return hasNeighbor(vmi, route.Gateway, timeout)
		}
	}
	return false
}

func hasNeighbor(vmi *v1.VirtualMachineInstance, dest string, timeout time.Duration) bool {
	neighbors, err := queryNeighbors(vmi, timeout)
	if err != nil {
		return false
	}
	for _, neigh := range neighbors {
		if neigh.Dst == dest && isValidNeighborState(neigh.State) {
			return true
		}
	}
	return false
}

func queryDefaultRoutes(vmi *v1.VirtualMachineInstance, ipFamilyArg string, timeout time.Duration) ([]Route, error) {
	const routeCmd = "ip -json %s route show default"
	output, err := console.RunCommandAndStoreOutput(vmi, fmt.Sprintf(routeCmd, ipFamilyArg), timeout)
	if err != nil {
		return nil, err
	}
	var routes []Route
	if err := json.Unmarshal([]byte(output), &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

func queryNeighbors(vmi *v1.VirtualMachineInstance, timeout time.Duration) ([]Neighbor, error) {
	const neighCmd = "ip -6 -json neigh show"
	output, err := console.RunCommandAndStoreOutput(vmi, neighCmd, timeout)
	if err != nil {
		return nil, err
	}
	var neighbors []Neighbor
	if err := json.Unmarshal([]byte(output), &neighbors); err != nil {
		return nil, err
	}
	return neighbors, nil
}

func isValidNeighborState(states []string) bool {
	for _, state := range states {
		state = strings.ToUpper(state)
		switch state {
		case "REACHABLE", "STALE", "DELAY", "PROBE", "PERMANENT":
			return true
		}
	}
	return false
}
