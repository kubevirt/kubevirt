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

package agent

import (
	"testing"

	v1 "kubevirt.io/api/core/v1"
)

func TestEvaluateGuestAgentStatus(t *testing.T) {
	basicCommands := func() []v1.GuestAgentCommandInfo {
		commands := make([]v1.GuestAgentCommandInfo, 0, len(requiredGuestAgentCommands))
		for _, name := range requiredGuestAgentCommands {
			commands = append(commands, v1.GuestAgentCommandInfo{Name: name, Enabled: true})
		}
		return commands
	}

	t.Run("disconnected", func(t *testing.T) {
		status := EvaluateGuestAgentStatus(&v1.VirtualMachineInstance{}, false, "", nil, nil)
		if status.Connected || status.SupportState != v1.GuestAgentSupportUnknown || status.UnsupportedReason != "" {
			t.Fatalf("unexpected disconnected status: %#v", status)
		}
	})

	t.Run("pending capabilities", func(t *testing.T) {
		status := EvaluateGuestAgentStatus(&v1.VirtualMachineInstance{}, true, "", nil, nil)
		if !status.Connected || status.SupportState != v1.GuestAgentSupportUnknown {
			t.Fatalf("unexpected pending status: %#v", status)
		}
	})

	t.Run("supported commands", func(t *testing.T) {
		status := EvaluateGuestAgentStatus(&v1.VirtualMachineInstance{}, true, "5.2", basicCommands(), nil)
		if status.SupportState != v1.GuestAgentSupportSupported || status.UnsupportedReason != "" {
			t.Fatalf("unexpected supported status: %#v", status)
		}
	})

	t.Run("unsupported commands", func(t *testing.T) {
		status := EvaluateGuestAgentStatus(&v1.VirtualMachineInstance{}, true, "5.2", basicCommands()[1:], nil)
		if status.SupportState != v1.GuestAgentSupportUnsupported ||
			status.UnsupportedReason != "This guest agent doesn't support required basic commands" {
			t.Fatalf("unexpected unsupported status: %#v", status)
		}
	})

	t.Run("supported legacy version", func(t *testing.T) {
		status := EvaluateGuestAgentStatus(&v1.VirtualMachineInstance{}, true, "4.2.1", nil, []string{"4.*"})
		if status.SupportState != v1.GuestAgentSupportSupported {
			t.Fatalf("unexpected legacy supported status: %#v", status)
		}
	})

	t.Run("unsupported legacy version", func(t *testing.T) {
		status := EvaluateGuestAgentStatus(&v1.VirtualMachineInstance{}, true, "99.0", nil, []string{"4.*"})
		if status.SupportState != v1.GuestAgentSupportUnsupported ||
			status.UnsupportedReason == "" {
			t.Fatalf("unexpected legacy unsupported status: %#v", status)
		}
	})

	t.Run("access credential commands", func(t *testing.T) {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				AccessCredentials: []v1.AccessCredential{{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				}},
			},
		}

		status := EvaluateGuestAgentStatus(vmi, true, "5.2", basicCommands(), nil)
		if status.SupportState != v1.GuestAgentSupportUnsupported ||
			status.UnsupportedReason != "This guest agent doesn't support required public key commands" {
			t.Fatalf("unexpected credential status: %#v", status)
		}
	})
}
