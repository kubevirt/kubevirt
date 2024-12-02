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
 * Copyright The KubeVirt Authors
 *
 */

package virthandler

import v1 "kubevirt.io/api/core/v1"

var RequiredGuestAgentCommands = []string{
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

var SSHRelatedGuestAgentCommands = []string{
	"guest-ssh-get-authorized-keys",
	"guest-ssh-add-authorized-keys",
	"guest-ssh-remove-authorized-keys",
}

var OldSSHRelatedGuestAgentCommands = []string{
	"guest-exec-status",
	"guest-exec",
	"guest-file-open",
	"guest-file-close",
	"guest-file-read",
	"guest-file-write",
}

var PasswordRelatedGuestAgentCommands = []string{
	"guest-set-user-password",
}

func _guestAgentCommandSubsetSupported(requiredCommands []string, commands []v1.GuestAgentCommandInfo) bool {
	var found bool
	for _, cmd := range requiredCommands {
		found = false
		for _, foundCmd := range commands {
			if cmd == foundCmd.Name {
				if foundCmd.Enabled {
					found = true
				}
				break
			}
		}
		if found == false {
			return false
		}
	}
	return true

}

func isGuestAgentSupported(vmi *v1.VirtualMachineInstance, commands []v1.GuestAgentCommandInfo) (bool, string) {
	if !_guestAgentCommandSubsetSupported(RequiredGuestAgentCommands, commands) {
		return false, "This guest agent doesn't support required basic commands"
	}

	checkSSH := false
	checkPasswd := false

	if vmi != nil && vmi.Spec.AccessCredentials != nil {
		for _, accessCredential := range vmi.Spec.AccessCredentials {
			if accessCredential.SSHPublicKey != nil && accessCredential.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
				// defer checking the command list so we only do that once
				checkSSH = true
			}
			if accessCredential.UserPassword != nil && accessCredential.UserPassword.PropagationMethod.QemuGuestAgent != nil {
				// defer checking the command list so we only do that once
				checkPasswd = true
			}

		}
	}

	if checkSSH && !sshRelatedCommandsSupported(commands) {
		return false, "This guest agent doesn't support required public key commands"
	}

	if checkPasswd && !_guestAgentCommandSubsetSupported(PasswordRelatedGuestAgentCommands, commands) {
		return false, "This guest agent doesn't support required password commands"
	}

	return true, "This guest agent is supported"
}

func sshRelatedCommandsSupported(commands []v1.GuestAgentCommandInfo) bool {
	return _guestAgentCommandSubsetSupported(SSHRelatedGuestAgentCommands, commands) ||
		_guestAgentCommandSubsetSupported(OldSSHRelatedGuestAgentCommands, commands)
}
