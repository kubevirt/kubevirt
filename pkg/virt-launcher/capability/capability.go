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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package capability

import (
	"fmt"

	"golang.org/x/sys/unix"

	"kubevirt.io/client-go/log"
)

var (
	capget = unix.Capget
	capset = unix.Capset
)

// GetPermittedCaps reads the current process's Permitted capability set
// and returns all permitted capability numbers.
func GetPermittedCaps() ([]uintptr, error) {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData

	if err := capget(&hdr, &data[0]); err != nil {
		return nil, fmt.Errorf("capget: %w", err)
	}

	var caps []uintptr
	for i := uint(0); i < 64; i++ {
		var permitted bool
		if i < 32 {
			permitted = data[0].Permitted&(uint32(1)<<i) != 0
		} else {
			permitted = data[1].Permitted&(uint32(1)<<(i-32)) != 0
		}
		if permitted {
			caps = append(caps, uintptr(i))
		}
	}
	return caps, nil
}

// EnsureInheritable raises the given capabilities into the Inheritable set.
// This is necessary because when a binary has file capabilities, the kernel
// clears the Inheritable set — so PR_CAP_AMBIENT_RAISE will fail unless we
// explicitly restore the Inheritable bits via Capset.
func EnsureInheritable(caps []uintptr) error {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData

	if err := capget(&hdr, &data[0]); err != nil {
		return fmt.Errorf("capget: %w", err)
	}

	for _, c := range caps {
		if c < 32 {
			data[0].Inheritable |= uint32(1) << uint(c)
		} else {
			data[1].Inheritable |= uint32(1) << uint(c-32)
		}
	}

	return capset(&hdr, &data[0])
}

// BuildAmbientCaps reads the process's Permitted capabilities, ensures they
// are all raised into the Inheritable set, and returns them for use as
// SysProcAttr.AmbientCaps. This propagates all pod-granted capabilities
// through the exec chain (monitor → launcher → virtqemud → QEMU).
//
// On failure, falls back to the minimum required capability
// (CAP_NET_BIND_SERVICE) to avoid breaking the existing boot path.
func BuildAmbientCaps() []uintptr {
	caps, err := GetPermittedCaps()
	if err != nil {
		log.Log.Warningf("failed to read permitted capabilities, falling back to CAP_NET_BIND_SERVICE: %v", err)
		return []uintptr{unix.CAP_NET_BIND_SERVICE}
	}

	if len(caps) == 0 {
		log.Log.Warningf("no permitted capabilities found, falling back to CAP_NET_BIND_SERVICE")
		return []uintptr{unix.CAP_NET_BIND_SERVICE}
	}

	if err := EnsureInheritable(caps); err != nil {
		log.Log.Warningf("failed to ensure capabilities are inheritable: %v", err)
	}

	return caps
}
