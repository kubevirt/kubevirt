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
	"fmt"
	"regexp"

	v1 "kubevirt.io/api/core/v1"
)

var requiredGuestAgentCommands = []string{
	"guest-ping",
	"guest-get-time",
	"guest-info",
	"guest-shutdown",
	"guest-network-get-interfaces",
	"guest-get-fsinfo",
	"guest-get-host-name",
	"guest-get-users",
	"guest-get-timezone",
	"guest-get-osinfo",
}

var sshRelatedGuestAgentCommands = []string{
	"guest-ssh-get-authorized-keys",
	"guest-ssh-add-authorized-keys",
	"guest-ssh-remove-authorized-keys",
}

var oldSSHRelatedGuestAgentCommands = []string{
	"guest-exec-status",
	"guest-exec",
	"guest-file-open",
	"guest-file-close",
	"guest-file-read",
	"guest-file-write",
}

var passwordRelatedGuestAgentCommands = []string{
	"guest-set-user-password",
}

func EvaluateGuestAgentStatus(
	vmi *v1.VirtualMachineInstance,
	connected bool,
	version string,
	commands []v1.GuestAgentCommandInfo,
	supportedVersions []string,
) *v1.GuestAgentStatus {
	status := &v1.GuestAgentStatus{
		Connected:    connected,
		SupportState: v1.GuestAgentSupportUnknown,
	}
	if !connected {
		return status
	}

	supported, reason := isGuestAgentSupported(vmi, version, commands, supportedVersions)
	if reason == "" {
		return status
	}

	if supported {
		status.SupportState = v1.GuestAgentSupportSupported
		return status
	}

	status.SupportState = v1.GuestAgentSupportUnsupported
	status.UnsupportedReason = reason
	return status
}

func isGuestAgentSupported(
	vmi *v1.VirtualMachineInstance,
	version string,
	commands []v1.GuestAgentCommandInfo,
	supportedVersions []string,
) (bool, string) {
	if len(commands) == 0 {
		if version == "" {
			return false, ""
		}

		for _, supportedVersion := range supportedVersions {
			matched, err := regexp.MatchString(supportedVersion, version)
			if err == nil && matched {
				return true, "This guest agent is supported"
			}
		}
		return false, fmt.Sprintf("Guest agent version '%s' is not supported", version)
	}

	availableCommands := make(map[string]bool, len(commands))
	for _, command := range commands {
		availableCommands[command.Name] = command.Enabled
	}

	if !commandSubsetSupported(requiredGuestAgentCommands, availableCommands) {
		return false, "This guest agent doesn't support required basic commands"
	}

	checkSSH := false
	checkPassword := false
	if vmi != nil {
		for _, accessCredential := range vmi.Spec.AccessCredentials {
			if accessCredential.SSHPublicKey != nil &&
				accessCredential.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
				checkSSH = true
			}
			if accessCredential.UserPassword != nil &&
				accessCredential.UserPassword.PropagationMethod.QemuGuestAgent != nil {
				checkPassword = true
			}
		}
	}

	if checkSSH &&
		!commandSubsetSupported(sshRelatedGuestAgentCommands, availableCommands) &&
		!commandSubsetSupported(oldSSHRelatedGuestAgentCommands, availableCommands) {
		return false, "This guest agent doesn't support required public key commands"
	}

	if checkPassword && !commandSubsetSupported(passwordRelatedGuestAgentCommands, availableCommands) {
		return false, "This guest agent doesn't support required password commands"
	}

	return true, "This guest agent is supported"
}

func commandSubsetSupported(requiredCommands []string, availableCommands map[string]bool) bool {
	for _, command := range requiredCommands {
		if enabled, exists := availableCommands[command]; !exists || !enabled {
			return false
		}
	}
	return true
}
