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

package portforward

import (
	"context"
	"fmt"
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

// vsockResource adapts a VirtualMachineInstance's VSOCK subresource to the
// portforwardableResource interface, so the existing stdio and local-listener
// forwarding code can tunnel over VSOCK without any changes.
type vsockResource struct {
	iface  kubecli.VirtualMachineInstanceInterface
	useTLS bool
}

func (v vsockResource) PortForward(name string, port int, protocol string) (kvcorev1.StreamInterface, error) {
	if err := validateVsockPort(port, protocol); err != nil {
		return nil, err
	}

	useTLS := v.useTLS
	return v.iface.VSOCK(name, &v1.VSOCKOptions{
		TargetPort: uint32(port), //nolint:gosec // bounds-checked above
		UseTLS:     &useTLS,
	})
}

// validateVsockPort checks that a port can be forwarded over VSOCK: only TCP is
// supported, and the target port must fit in VSOCKOptions.TargetPort (uint32,
// but constrained here to the uint16 range like any application port).
func validateVsockPort(port int, protocol string) error {
	if protocol != protocolTCP {
		return fmt.Errorf("VSOCK does not support protocol %q, only TCP is supported", protocol)
	}
	if port <= 0 || port > math.MaxUint16 {
		return fmt.Errorf("invalid port %d: port must be between 1 and %d", port, math.MaxUint16)
	}
	return nil
}

// setVsockResource validates the requested ports, resolves the target to a running
// VirtualMachineInstance (a VM always shares its VMI's name, so vm/ and vmi/ targets
// resolve the same way), and configures o.resource to forward over VSOCK.
func (o *PortForward) setVsockResource(
	ctx context.Context, client kubecli.KubevirtClient, namespace, name string, ports []forwardedPort,
) error {
	for _, port := range ports {
		if err := validateVsockPort(port.remote, port.protocol); err != nil {
			return err
		}
	}

	vmiClient := client.VirtualMachineInstance(namespace)
	vmi, err := vmiClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to find VirtualMachineInstance %q: %w", name, err)
	}
	if vmi.Status.Phase != v1.Running {
		return fmt.Errorf("VirtualMachineInstance %q is not running (phase: %s)", name, vmi.Status.Phase)
	}

	o.resource = vsockResource{iface: vmiClient, useTLS: vsockUseTLS}
	return nil
}
