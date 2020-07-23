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

package tests

import (
	"fmt"

	expect "github.com/google/goexpect"
	"k8s.io/utils/net"

	v1 "kubevirt.io/client-go/api/v1"
)

// PingFromVMConsole performs a ping through the provided VMI console.
func PingFromVMConsole(vmi *v1.VirtualMachineInstance, ipAddr, prompt string) error {
	pingString := "ping"
	if net.IsIPv6String(ipAddr) {
		pingString = "ping -6"
	}
	cmdCheck := fmt.Sprintf("%s %s -c 1 -w 5\n", pingString, ipAddr)
	err := CheckForTextExpecter(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 30)
	if err != nil {
		return fmt.Errorf("Failed to ping VMI %s, error: %v", vmi.Name, err)
	}
	return nil
}
