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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libnet

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/tests/console"
)

// PingFromVMConsole performs a ping through the provided VMI console.
// Optional arguments for the ping command may be provided, overwirting the default ones.
// (default ping options: "-c 5, -w 10")
// Note: The maximum overall command timeout is 20 seconds.
func PingFromVMConsole(vmi *v1.VirtualMachineInstance, ipAddr string, args ...string) error {
	const maxCommandTimeout = 20 * time.Second

	pingString := "ping"

	if len(args) == 0 {
		args = []string{"-c 5", "-w 10"}
	}
	if net.IsIPv6String(ipAddr) {
		args = append(args, "-6")
	}

	// Destination has to be the last argument, at some ping implementation
	// it fails if we don't do so
	args = append(args, ipAddr)

	cmd := strings.Join(append([]string{pingString}, args...), " ")

	err := console.RunCommand(vmi, cmd, maxCommandTimeout)
	if err != nil {
		return fmt.Errorf("Failed to ping VMI %s, error: %v", vmi.Name, err)
	}
	return nil
}
