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

package libvmi

import (
	v1 "kubevirt.io/api/core/v1"
)

// WithAccessCredentialSSHPublicKey adds an AccessCredential that propagates the
// public keys found in secretName to the authorized_keys file of the user with
// name userName via the qemu-guest-agent.
func WithAccessCredentialSSHPublicKey(secretName, userName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.AccessCredentials = append(vmi.Spec.AccessCredentials, v1.AccessCredential{
			SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
				Source: v1.SSHPublicKeyAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: secretName,
					},
				},
				PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
						Users: []string{userName},
					},
				},
			},
		})
	}
}

// WithAccessCredentialUserPassword adds an AccessCredential that propagates the
// user passwords found in secretName via the qemu-guest-agent.
func WithAccessCredentialUserPassword(secretName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.AccessCredentials = append(vmi.Spec.AccessCredentials, v1.AccessCredential{
			UserPassword: &v1.UserPasswordAccessCredential{
				Source: v1.UserPasswordAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: secretName,
					},
				},
				PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
				},
			},
		})
	}
}
