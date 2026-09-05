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
	"fmt"
	"math/rand/v2"
	"net"

	expect "github.com/google/goexpect"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
)

func GenerateRandomMac() net.HardwareAddr {
	// A test MAC needs no cryptographic randomness; math/rand avoids wasting entropy.
	suffix := rand.Uint32() //nolint:gosec
	// 0x02 marks the address as locally administered unicast. The byte conversions
	// intentionally truncate suffix to its low bytes (gosec G115 does not apply), and
	// the 8/16 shifts select those bytes (not mnd magic numbers).
	return net.HardwareAddr{0x02, 0x00, 0x00, byte(suffix), byte(suffix >> 8), byte(suffix >> 16)} //nolint:gosec,mnd
}

func CheckMacAddress(vmi *v1.VirtualMachineInstance, interfaceName, macAddress string) error {
	const timeoute = 15
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	err := console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: ""},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: macAddress},
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: console.ShellSuccess},
	}, timeoute)
	if err != nil {
		return fmt.Errorf(
			"could not check mac address of interface %s: MAC %s was not found in the guest %s: %w",
			interfaceName, macAddress, vmi.Name, err,
		)
	}
	return nil
}
