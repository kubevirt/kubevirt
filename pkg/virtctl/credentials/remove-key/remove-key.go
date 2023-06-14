package remove_key

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/credentials/common"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmdFlags := &removeSshKeyFlags{}
	cmd := &cobra.Command{
		Use:     "remove-ssh-key",
		Short:   "Remove credentials from a virtual machine.",
		Example: exampleUsage,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveKeyCommand(clientConfig, cmdFlags, cmd, args)
		},
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

type removeSshKeyFlags struct {
	common.SshCommandFlags

	Force bool
}

func (r *removeSshKeyFlags) AddToCommand(cmd *cobra.Command) {
	r.SshCommandFlags.AddToCommand(cmd)

	cmd.Flags().BoolVar(&r.Force, "force", false, "Force update of secret, even if it's not owned by the VM.")
}

func runRemoveKeyCommand(clientConfig clientcmd.ClientConfig, cmdFlags *removeSshKeyFlags, cmd *cobra.Command, args []string) error {
	vmName := args[0]

	vmNamespace, _, err := clientConfig.Namespace()
	if err != nil {
		return fmt.Errorf("error getting namespace: %w", err)
	}

	// Reading the key before accessing cluster
	sshKey, err := common.GetSshKey(&cmdFlags.SshCommandFlags)
	if err != nil {
		return fmt.Errorf("error getting ssh key: %w", err)
	}

	cli, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("error getting kubevirt client: %w", err)
	}

	vm, err := cli.VirtualMachine(vmNamespace).Get(cmd.Context(), vmName, &metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting virtual machine: %w", err)
	}

	secrets := common.GetSshSecretsForUser(vm.Spec.Template.Spec.AccessCredentials, cmdFlags.User)
	if len(secrets) == 0 {
		cmd.Printf("No secrets associated with user %s", cmdFlags.User)
		return nil
	}

	var filteredSecrets []string
	if cmdFlags.Secret == "" {
		filteredSecrets = secrets
	} else {
		if common.ContainsValue(secrets, cmdFlags.Secret) {
			filteredSecrets = append(filteredSecrets, cmdFlags.Secret)
		} else {
			return fmt.Errorf("secret %s is not associated with user %s", cmdFlags.Secret, cmdFlags.User)
		}
	}

	for _, secretName := range filteredSecrets {
		err := removeKeyFromSecret(cmd.Context(), cli, vm, secretName, sshKey, cmdFlags.Force)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeKeyFromSecret(ctx context.Context, cli kubecli.KubevirtClient, vm *v1.VirtualMachine, secretName string, key string, force bool) error {
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
			if !common.IsOwnedByVm(secret, vm) {
				return fmt.Errorf("secret %s does not have an owner reference pointing to VM %s", secretName, vm.Name)
			}
		}

		for fileName, data := range secret.Data {
			updatedData := removeSshKeyFromBytes(key, data)
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

func removeSshKeyFromBytes(key string, data []byte) []byte {
	lines := strings.Split(string(data), "\n")

	resultLines := make([]string, 0, len(lines))
	for i := range lines {
		if !common.LineContainsKey(lines[i], key) && len(strings.TrimSpace(lines[i])) > 0 {
			resultLines = append(resultLines, lines[i])
		}
	}

	return []byte(strings.Join(resultLines, "\n"))
}
