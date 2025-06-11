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

var passwordRelatedGuestAgentCommands = []string{
	"guest-set-user-password",
}

var execProbeGuestAgentCommands = []string{
	"guest-exec-status",
	"guest-exec",
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

	if checkSSH && !guestAgentCommandSubsetSupported(sshRelatedGuestAgentCommands, availableCmdsMap) {
		return false, "This guest agent doesn't support required public key commands"
	}

	if checkPasswd && !guestAgentCommandSubsetSupported(passwordRelatedGuestAgentCommands, availableCmdsMap) {
		return false, "This guest agent doesn't support required password commands"
	}

	var checkExecProbe bool
	if vmi != nil && vmi.Spec.ReadinessProbe != nil && vmi.Spec.ReadinessProbe.Exec != nil {
		checkExecProbe = true
	}
	if vmi != nil && vmi.Spec.LivenessProbe != nil && vmi.Spec.LivenessProbe.Exec != nil {
		checkExecProbe = true
	}

	if checkExecProbe && !guestAgentCommandSubsetSupported(execProbeGuestAgentCommands, availableCmdsMap) {
		return false, "This guest agent doesn't support required exec probe commands"
	}

	return true, "This guest agent is supported"
}
