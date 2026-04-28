/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virthandler

import v1 "kubevirt.io/api/core/v1"

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

func guestAgentCommandSubsetSupported(requiredCommands []string, availableCmdsMap map[string]bool) bool {
	for _, cmd := range requiredCommands {
		if enabled, exists := availableCmdsMap[cmd]; !exists || !enabled {
			return false
		}
	}
	return true
}

func isGuestAgentSupported(vmi *v1.VirtualMachineInstance, commands []v1.GuestAgentCommandInfo) (bool, string) {
	availableCmdsMap := make(map[string]bool, len(commands))
	for _, command := range commands {
		availableCmdsMap[command.Name] = command.Enabled
	}

	if !guestAgentCommandSubsetSupported(requiredGuestAgentCommands, availableCmdsMap) {
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

	if checkSSH && !sshRelatedCommandsSupported(availableCmdsMap) {
		return false, "This guest agent doesn't support required public key commands"
	}

	if checkPasswd && !guestAgentCommandSubsetSupported(passwordRelatedGuestAgentCommands, availableCmdsMap) {
		return false, "This guest agent doesn't support required password commands"
	}

	return true, "This guest agent is supported"
}

func sshRelatedCommandsSupported(availableCmdsMap map[string]bool) bool {
	return guestAgentCommandSubsetSupported(sshRelatedGuestAgentCommands, availableCmdsMap) ||
		guestAgentCommandSubsetSupported(oldSSHRelatedGuestAgentCommands, availableCmdsMap)
}
