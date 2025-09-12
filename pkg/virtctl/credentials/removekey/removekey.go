package removekey

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials/common"
)

func NewCommand() *cobra.Command {
	cmdFlags := &removeSSHKeyFlags{}
	cmd := &cobra.Command{
		Use:     "remove-ssh-key",
		Short:   "Remove credentials from a virtual machine.",
		Example: exampleUsage,
		RunE:    cmdFlags.runRemoveKeyCommand,
	}
	cmdFlags.AddToCommand(cmd)

	return cmd
}

const exampleUsage = `  # Remove an SSH key for a running virtual machine.
  {{ProgramName}} credentials remove-ssh-key --user <username> --file <path-to-ssh-public-key> <vm-name>

  # Remove an SSH key for a running virtual machine. Key is provided as literal parameter.
  {{ProgramName}} credentials remove-ssh-key --user <username> --value <literal-ssh-public-key> <vm-name>

  # Remove an SSH key from a secret that is not owned by the virtual machine.
  {{ProgramName}} credentials remove-ssh-key --user <username> --file <path-to-ssh-public-key> --force <vm-name>
`

type removeSSHKeyFlags struct {
	common.SSHCommandFlags

	Force bool
}

func (r *removeSSHKeyFlags) AddToCommand(cmd *cobra.Command) {
	r.SSHCommandFlags.AddToCommand(cmd)

	cmd.Flags().BoolVar(&r.Force, "force", false, "Force update of secret, even if it's not owned by the VM.")
}

func (r *removeSSHKeyFlags) runRemoveKeyCommand(cmd *cobra.Command, args []string) error {
	vmName := args[0]

	virtCli, k8sCli, vmNamespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("error getting kubevirt client or namespace: %w", err)
	}

	// Reading the key before accessing cluster
	sshKey, err := common.GetSSHKey(&r.SSHCommandFlags)
	if err != nil {
		return fmt.Errorf("error getting ssh key: %w", err)
	}

	vm, err := virtCli.VirtualMachine(vmNamespace).Get(cmd.Context(), vmName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting virtual machine: %w", err)
	}

	secrets := common.GetSSHSecretsForUser(vm.Spec.Template.Spec.AccessCredentials, r.User)
	if len(secrets) == 0 {
		cmd.Printf("No secrets associated with user %s", r.User)
		return nil
	}

	var filteredSecrets []string
	if r.Secret == "" {
		filteredSecrets = secrets
	} else {
		if common.ContainsValue(secrets, r.Secret) {
			filteredSecrets = append(filteredSecrets, r.Secret)
		} else {
			return fmt.Errorf("secret %s is not associated with user %s", r.Secret, r.User)
		}
	}

	for _, secretName := range filteredSecrets {
		err := removeKeyFromSecret(cmd.Context(), k8sCli, vm, secretName, sshKey, r.Force)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeKeyFromSecret(
	ctx context.Context,
	cli kubernetes.Interface,
	vm *v1.VirtualMachine,
	secretName string,
	key string,
	force bool,
) error {
	// Looping, because Update API call can fail with conflict
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		secret, err := cli.CoreV1().Secrets(vm.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			// Secret does not exist, nothing to do
			return nil
		}
		if err != nil {
			return err
		}

		if !force {
			// Check if secret is owned by the VM. This is useful to not accidentally update a secret that is used by multiple VMs.
			if !common.IsOwnedByVM(secret, vm) {
				return fmt.Errorf("secret %s does not have an owner reference pointing to VM %s", secretName, vm.Name)
			}
		}

		for fileName, data := range secret.Data {
			updatedData := removeSSHKeyFromBytes(key, data)
			if len(updatedData) == 0 {
				delete(secret.Data, fileName)
			} else {
				secret.Data[fileName] = updatedData
			}
		}

		_, err = cli.CoreV1().Secrets(vm.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	})
}

func removeSSHKeyFromBytes(key string, data []byte) []byte {
	lines := strings.Split(string(data), "\n")

	resultLines := make([]string, 0, len(lines))
	for i := range lines {
		if !common.LineContainsKey(lines[i], key) && strings.TrimSpace(lines[i]) != "" {
			resultLines = append(resultLines, lines[i])
		}
	}

	return []byte(strings.Join(resultLines, "\n"))
}
