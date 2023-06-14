package set_password

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/credentials/common"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func SetPasswordCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmdFlags := &passwordCommandFlags{}
	cmd := &cobra.Command{
		Use:     "set-password",
		Short:   "Set password for a user",
		Args:    templates.ExactArgs("set-password", 1),
		Example: exampleUsage,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetPasswordCommand(clientConfig, cmdFlags, cmd, args)
		},
	}
	cmdFlags.AddToCommand(cmd)

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

const exampleUsage = `  # Set a user password for a virtual machine.
  {{ProgramName}} credentials set-password --user <username> --password <password> <vm-name>

  # Set a user password in a secret that is not owned by the virtual machine.
  {{ProgramName}} credentials set-password --user <username> --password <password> --force <vm-name>
`

type passwordCommandFlags struct {
	common.CommandFlags

	Password string

	Force bool
}

func (p *passwordCommandFlags) AddToCommand(cmd *cobra.Command) {
	p.CommandFlags.AddToCommand(cmd)

	const passwordFlag = "password"
	cmd.Flags().StringVarP(&p.Password, passwordFlag, "p", "", "Password for the user")
	err := cmd.MarkFlagRequired(passwordFlag)
	if err != nil {
		panic(err)
	}

	cmd.Flags().BoolVar(&p.Force, "force", false, "Force update of secret, even if it's not owned by the VM.")
}

func runSetPasswordCommand(clientConfig clientcmd.ClientConfig, cmdFlags *passwordCommandFlags, cmd *cobra.Command, args []string) error {
	vmName := args[0]

	vmNamespace, _, err := clientConfig.Namespace()
	if err != nil {
		return fmt.Errorf("error getting namespace: %w", err)
	}

	cli, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("error getting kubevirt client: %w", err)
	}

	vm, err := cli.VirtualMachine(vmNamespace).Get(cmd.Context(), vmName, &metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting virtual machine: %w", err)
	}

	secrets := getPasswordSecrets(vm.Spec.Template.Spec.AccessCredentials)
	if len(secrets) == 0 {
		return fmt.Errorf("no secrets assigned to UserPassword AccessCredentials")
	}

	secretName, err := common.FindSecretOrGetFirst(cmdFlags.Secret, secrets)
	if err != nil {
		return err
	}

	if !cmdFlags.Force {
		secret, err := cli.CoreV1().Secrets(vm.Namespace).Get(cmd.Context(), secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting secret \"%s\": %w", secretName, err)
		}

		// Check if secret is owned by the VM. This is useful to not accidentally update a secret that is used by multiple VMs.
		if !common.IsOwnedByVm(secret, vm) {
			return fmt.Errorf("secret %s does not have an owner reference pointing to VM %s", secretName, vm.Name)
		}
	}

	addKeyPatch := common.AddKeyToSecretPatchOp(cmdFlags.User, []byte(cmdFlags.Password))

	// Try patch to only add the new key.
	_, err = cli.CoreV1().Secrets(vm.Namespace).Patch(cmd.Context(), secretName, types.JSONPatchType, common.MustMarshalPatch(addKeyPatch), metav1.PatchOptions{})
	if err != nil {
		// If it fails, it probably means that /data field is nil. Try second patch to add /data field.
		fullPatch := common.MustMarshalPatch(append(common.AddDataFieldToSecretPatchOp(), addKeyPatch)...)
		_, err = cli.CoreV1().Secrets(vmNamespace).Patch(cmd.Context(), secretName, types.JSONPatchType, fullPatch, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("error patching secret \"%s\": %w", secretName, err)
		}
	}

	cmd.Printf("Successfully set password in secret \"%s\"", secretName)
	return nil
}

func getPasswordSecrets(accessCredentials []v1.AccessCredential) []string {
	var result []string
	for i := range accessCredentials {
		credential := &accessCredentials[i]
		if credential.UserPassword != nil &&
			credential.UserPassword.Source.Secret != nil &&
			credential.UserPassword.Source.Secret.SecretName != "" &&
			credential.UserPassword.PropagationMethod.QemuGuestAgent != nil {
			result = append(result, credential.UserPassword.Source.Secret.SecretName)
		}
	}
	return result
}
