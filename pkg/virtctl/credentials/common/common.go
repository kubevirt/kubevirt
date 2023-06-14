package common

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
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

type SshCommandFlags struct {
	CommandFlags

	SshPubKeyFile    string
	SshPubKeyLiteral string
}

const (
	keyFileFlag  = "file"
	keyValueFlag = "value"
)

func (s *SshCommandFlags) AddToCommand(cmd *cobra.Command) {
	s.CommandFlags.AddToCommand(cmd)

	cmd.Flags().StringVarP(&s.SshPubKeyFile, keyFileFlag, "f", "", "Path to the SSH public key file.")
	err := cmd.MarkFlagFilename(keyFileFlag)
	if err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(&s.SshPubKeyLiteral, keyValueFlag, "", "Literal value of the SSH public key.")
	cmd.MarkFlagsMutuallyExclusive(keyFileFlag, keyValueFlag)

}

func GetSshKey(flags *SshCommandFlags) (string, error) {
	if flags.SshPubKeyLiteral != "" {
		return flags.SshPubKeyLiteral, nil
	}

	if flags.SshPubKeyFile == "" {
		return "", fmt.Errorf("one of --%s, or --%s must be specified", keyFileFlag, keyValueFlag)
	}
	data, err := os.ReadFile(flags.SshPubKeyFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetSshSecretsForUser(accessCredentials []v1.AccessCredential, user string) []string {
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

func IsOwnedByVm(obj metav1.Object, vm *v1.VirtualMachine) bool {
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

func LineContainsKey(line string, key string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), key)
}

func AddDataFieldToSecretPatchOp() []patch.PatchOperation {
	return []patch.PatchOperation{{
		Op:    patch.PatchTestOp,
		Path:  "/data",
		Value: nil,
	}, {
		Op:    patch.PatchAddOp,
		Path:  "/data",
		Value: map[string][]byte{},
	}}
}

func AddKeyToSecretPatchOp(keyName string, key []byte) patch.PatchOperation {
	return patch.PatchOperation{
		Op:    patch.PatchAddOp,
		Path:  "/data/" + keyName,
		Value: key,
	}
}

func MustMarshalPatch(patches ...patch.PatchOperation) []byte {
	data, err := patch.GeneratePatchPayload(patches...)
	if err != nil {
		panic(err)
	}
	return data
}

func RandomWithPrefix(prefix string) string {
	return prefix + rand.String(6)
}
