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

// Package pipe provides utilities for proxying domain notify connections
// between virt-launcher pods and virt-handler's notify server.
//
// The pipe acts as a bridge, creating a unix socket within the VMI's
// filesystem that proxies to virt-handler's domain-notify.sock.
package pipe

import (
	"fmt"
	"net"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

// InjectNotify injects the domain-notify.sock into the VMI pod and listens for connections
func InjectNotify(pod isolation.IsolationResult, virtShareDir string,
	nonRoot bool) (net.Listener, error) {
	root, err := pod.MountRoot()
	if err != nil {
		return nil, err
	}
	socketDir, err := root.AppendAndResolveWithRelativeRoot(virtShareDir)
	if err != nil {
		return nil, err
	}

	listener, err := safepath.ListenUnixNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		return nil, fmt.Errorf("failed to create unix socket for proxy service: %w", err)
	}

	if nonRoot {
		socketPath, err := safepath.JoinNoFollow(socketDir, "domain-notify-pipe.sock")
		if err != nil {
			return nil, err
		}

		err = diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath)
		if err != nil {
			return nil, fmt.Errorf("unable to change ownership for domain notify: %w", err)
		}
	}

	return listener, nil
}
