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

package libnet

import (
	"context"
	"fmt"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
)

// PingFromVMConsole performs a ping through the provided VMI console.
// Optional arguments for the ping command may be provided, overwriting the default ones.
// (default ping options: "-c 5, -w 10")
// Note: The maximum overall command timeout is 20 seconds.
func PingFromVMConsole(vmi *v1.VirtualMachineInstance, ipAddr string, args ...string) error {
	const maxCommandTimeout = 20 * time.Second

	pingString := "ping"
	if net.IsIPv6String(ipAddr) {
		pingString = "ping -6"
	}

	if len(args) == 0 {
		args = []string{"-c 5", "-w 10"}
	}
	args = append([]string{pingString}, args...)
	args = append(args, ipAddr)
	cmdCheck := strings.Join(args, " ")

	err := console.RunCommand(vmi, cmdCheck, maxCommandTimeout)
	if err != nil {
		return fmt.Errorf("failed to ping VMI %s, error: %v", vmi.Name, err)
	}
	return nil
}

func WaitForGuestNetworkReady(vmi *v1.VirtualMachineInstance, ipFamily k8sv1.IPFamily, timeout time.Duration) error {
	checkRouteCmd := "ip route show default | grep -q 'default via'"
	if ipFamily == k8sv1.IPv6Protocol {
		checkRouteCmd = "ip -6 route show default | grep -q 'default via' && ip -6 neigh show | grep -qE 'REACHABLE|STALE|DELAY|PROBE'"
	}

	return wait.PollUntilContextTimeout(
		context.Background(),
		time.Second,
		timeout,
		true,
		func(ctx context.Context) (done bool, err error) {
			err = console.RunCommand(vmi, checkRouteCmd, 5*time.Second)
			if err != nil {
				return false, nil
			}
			return true, nil
		},
	)
}
