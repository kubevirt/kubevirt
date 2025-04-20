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
 */

package common

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
)

type CommandFlags struct {
	User   string
	Secret string
}

func (c *CommandFlags) AddToCommand(cmd *cobra.Command) {
	const userFlag = "user"
	cmd.Flags().StringVarP(&c.User, userFlag, "u", "", "Name of the user.")
	err := cmd.MarkFlagRequired(userFlag)
	if err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&c.Secret, "secret", "", "Name of the secret with SSH keys.")
}

type SSHCommandFlags struct {
	CommandFlags

	SSHPubKeyFile    string
	SSHPubKeyLiteral string
}

const (
	keyFileFlag  = "file"
	keyValueFlag = "value"
)

func (s *SSHCommandFlags) AddToCommand(cmd *cobra.Command) {
	s.CommandFlags.AddToCommand(cmd)

	cmd.Flags().StringVarP(&s.SSHPubKeyFile, keyFileFlag, "f", "", "Path to the SSH public key file.")
	err := cmd.MarkFlagFilename(keyFileFlag)
	if err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&s.SSHPubKeyLiteral, keyValueFlag, "", "Literal value of the SSH public key.")
	cmd.MarkFlagsMutuallyExclusive(keyFileFlag, keyValueFlag)
}

func GetSSHKey(flags *SSHCommandFlags) (string, error) {
	if flags.SSHPubKeyLiteral != "" {
		return flags.SSHPubKeyLiteral, nil
	}

	if flags.SSHPubKeyFile == "" {
		return "", fmt.Errorf("one of --%s, or --%s must be specified", keyFileFlag, keyValueFlag)
	}
	data, err := os.ReadFile(flags.SSHPubKeyFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetSSHSecretsForUser(accessCredentials []v1.AccessCredential, user string) []string {
	var result []string
	for i := range accessCredentials {
		credential := &accessCredentials[i]
		if credential.SSHPublicKey == nil {
			continue
		}

		if credential.SSHPublicKey.Source.Secret == nil || credential.SSHPublicKey.Source.Secret.SecretName == "" {
			continue
		}
		secretName := credential.SSHPublicKey.Source.Secret.SecretName

		if credential.SSHPublicKey.PropagationMethod.QemuGuestAgent == nil {
			continue
		}

		if ContainsValue(credential.SSHPublicKey.PropagationMethod.QemuGuestAgent.Users, user) {
			result = append(result, secretName)
		}
	}
	return result
}

func FindSecretOrGetFirst(secretName string, secrets []string) (string, error) {
	if secretName == "" {
		if len(secrets) > 1 {
			return "", fmt.Errorf("multiple secrets specified, use option --secret to specify which secret to modify")
		}
		return secrets[0], nil
	}

	if ContainsValue(secrets, secretName) {
		return secretName, nil
	}

	return "", fmt.Errorf("secret %s was not assigned", secretName)
}

func IsOwnedByVM(obj metav1.Object, vm *v1.VirtualMachine) bool {
	for _, ownerReference := range obj.GetOwnerReferences() {
		if ownerReference.Kind == v1.VirtualMachineGroupVersionKind.Kind &&
			ownerReference.Name == vm.Name &&
			ownerReference.UID == vm.UID {
			return true
		}
	}
	return false
}

func ContainsValue(slice []string, value string) bool {
	for _, val := range slice {
		if val == value {
			return true
		}
	}
	return false
}

func LineContainsKey(line, key string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), key)
}

func RandomWithPrefix(prefix string) string {
	const suffixLength = 6
	return prefix + rand.String(suffixLength)
}
