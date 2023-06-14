package add_key

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials/common"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmdFlags := &addSshKeyFlags{}
	cmd := &cobra.Command{
		Use:     "add-ssh-key",
		Short:   "Add credentials to a virtual machine.",
		Args:    templates.ExactArgs("add-ssh-key", 1),
		Example: exampleUsage,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddKeyCommand(clientConfig, cmdFlags, cmd, args)
		},
	}
	cmdFlags.AddToCommand(cmd)

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

const exampleUsage = `  # Add an SSH key for a running virtual machine.
  {{ProgramName}} credentials add-ssh-key --user <username> --file <path-to-ssh-public-key> <vm-name>

  # Add an SSH key for a running virtual machine. Key is provided as literal parameter.
  {{ProgramName}} credentials add-ssh-key --user <username> --value <literal-ssh-public-key> <vm-name>

  # Add an SSH key to a secret that is not owned by the virtual machine.
  {{ProgramName}} credentials add-ssh-key --user <username> --file <path-to-ssh-public-key> --force <vm-name>

  # Create a new secret with the SSH key, and assign it to the specified virtual machine. 
  {{ProgramName}} credentials add-ssh-key --create-secret --user <username> --file <path-to-ssh-public-key> <vm-name>

  # Create a new secret with the SSH key, and assign it to a running VM. It will take effect after restart.
  {{ProgramName}} credentials add-ssh-key --create-secret --user <username> --file <path-to-ssh-public-key> --force <vm-name>
`

type addSshKeyFlags struct {
	common.SshCommandFlags

	CreateSecret bool
	UpdateSecret bool

	Force bool
}

func (a *addSshKeyFlags) AddToCommand(cmd *cobra.Command) {
	a.SshCommandFlags.AddToCommand(cmd)

	const (
		createSecretFlag = "create-secret"
		updateSecretFlag = "update-secret"
	)

	cmd.Flags().BoolVar(&a.CreateSecret, createSecretFlag, false, "Create a new secret for the SSH key. The new key will not be added to a running VM. Use --force to add a new secret even if the VM is running.")
	cmd.Flags().BoolVar(&a.UpdateSecret, updateSecretFlag, false, "Add the SSH key to an existing secret. Use --force option, if the secret does not have owner reference pointing to the VM.")
	cmd.MarkFlagsMutuallyExclusive(createSecretFlag, updateSecretFlag)

	cmd.Flags().BoolVar(&a.Force, "force", false, "Force update of secret, even if it's not owned by the VM.")
}

func runAddKeyCommand(clientConfig clientcmd.ClientConfig, cmdFlags *addSshKeyFlags, cmd *cobra.Command, args []string) error {
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

	if shouldCreateNewSecret(cmdFlags, vm) {
		return addSecretWithSshKey(cmd, cli, cmdFlags, vm, sshKey)
	}
	return updateSecretWithSshKey(cmd, cli, cmdFlags, vm, sshKey)
}

func addSecretWithSshKey(cmd *cobra.Command, cli kubecli.KubevirtClient, cmdFlags *addSshKeyFlags, vm *v1.VirtualMachine, sshKey string) (err error) {
	if !cmdFlags.Force {
		// Only create a secret if VM is not running.
		_, err := cli.VirtualMachineInstance(vm.Namespace).Get(cmd.Context(), vm.Name, &metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("virtual machine %s is running. Use --force flag to update a running VM, it will take effect after restart", vm.Name)
		}
		if !errors.IsNotFound(err) {
			return fmt.Errorf("error when getting virtual machine instance: %w", err)
		}
	}

	secret := newSecretWithKey(vm, sshKey)
	secret, err = cli.CoreV1().Secrets(vm.Namespace).Create(cmd.Context(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating secret: %w", err)
	}

	accessCredential := newAccessCredential(secret.Name, cmdFlags.User)
	accessCredentialPatch := patchToAddAccessCredential(accessCredential)

	// First, Try to add the new access credential to the existing array.
	_, err = cli.VirtualMachine(vm.Namespace).Patch(cmd.Context(), vm.Name, types.JSONPatchType, common.MustMarshalPatch(accessCredentialPatch), &metav1.PatchOptions{})
	if err != nil {
		// If it fails, it probably means that the array is nil. Try to add the array.
		fullPatch := common.MustMarshalPatch(append(patchToAddAccessCredentialsArray(), accessCredentialPatch)...)
		_, err = cli.VirtualMachine(vm.Namespace).Patch(cmd.Context(), vm.Name, types.JSONPatchType, fullPatch, &metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("error patching virtual machine: %w", err)
		}
	}

	return nil
}

func updateSecretWithSshKey(cmd *cobra.Command, cli kubecli.KubevirtClient, cmdFlags *addSshKeyFlags, vm *v1.VirtualMachine, sshKey string) error {
	secrets := common.GetSshSecretsForUser(vm.Spec.Template.Spec.AccessCredentials, cmdFlags.User)
	if len(secrets) == 0 {
		return fmt.Errorf("no secrets specified for user: %s", cmdFlags.User)
	}

	secretName, err := common.FindSecretOrGetFirst(cmdFlags.Secret, secrets)
	if err != nil {
		return err
	}

	secret, err := cli.CoreV1().Secrets(vm.Namespace).Get(cmd.Context(), secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting secret \"%s\": %w", secretName, err)
	}

	if secretContainsKey(secret.Data, sshKey) {
		cmd.Printf("Secret \"%s\" already contains this SSH key.", secretName)
		return nil
	}

	if !cmdFlags.Force {
		// Check if secret is owned by the VM. This is useful to not accidentally update a secret that is used by multiple VMs.
		if !common.IsOwnedByVm(secret, vm) {
			return fmt.Errorf("secret %s does not have an owner reference pointing to VM %s", secretName, vm.Name)
		}
	}

	addKeyPatch := common.AddKeyToSecretPatchOp(common.RandomWithPrefix("ssh-key-"), []byte(sshKey))

	// First, try patch to add a new key
	_, err = cli.CoreV1().Secrets(vm.Namespace).Patch(cmd.Context(), secretName, types.JSONPatchType, common.MustMarshalPatch(addKeyPatch), metav1.PatchOptions{})
	if err != nil {
		// If it fails, the /data may be nil. Try a patch that adds the /data field
		fullPatch := common.MustMarshalPatch(append(common.AddDataFieldToSecretPatchOp(), addKeyPatch)...)
		_, err = cli.CoreV1().Secrets(vm.Namespace).Patch(cmd.Context(), secretName, types.JSONPatchType, fullPatch, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("error patching secret \"%s\": %w", secretName, err)
		}
	}

	cmd.Printf("Successfully added the key to secret \"%s\"", secretName)
	return nil
}

func shouldCreateNewSecret(flags *addSshKeyFlags, vm *v1.VirtualMachine) bool {
	if flags.CreateSecret {
		return true
	}
	if flags.UpdateSecret {
		return false
	}

	// Default behavior: Create a new secret, if no secret is defined for a user
	secrets := common.GetSshSecretsForUser(vm.Spec.Template.Spec.AccessCredentials, flags.User)
	return len(secrets) == 0
}

func newSecretWithKey(vm *v1.VirtualMachine, sshKey string) *core.Secret {
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: vm.Name + "-ssh-key-",
			Namespace:    vm.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
				Kind:       v1.VirtualMachineGroupVersionKind.Kind,
				Name:       vm.Name,
				UID:        vm.UID,
				Controller: pointer.Bool(true),
			}},
		},
		Data: map[string][]byte{
			common.RandomWithPrefix("ssh-key-"): []byte(sshKey),
		},
	}
}

func secretContainsKey(secretData map[string][]byte, key string) bool {
	for _, data := range secretData {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if common.LineContainsKey(line, key) {
				return true
			}
		}
	}
	return false
}

func newAccessCredential(secretName string, user string) *v1.AccessCredential {
	return &v1.AccessCredential{
		SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
			Source: v1.SSHPublicKeyAccessCredentialSource{
				Secret: &v1.AccessCredentialSecretSource{
					SecretName: secretName,
				},
			},
			PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
				QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
					Users: []string{user},
				},
			},
		},
	}
}

func patchToAddAccessCredentialsArray() []patch.PatchOperation {
	return []patch.PatchOperation{{
		Op:    patch.PatchTestOp,
		Path:  "/spec/template/spec/accessCredentials",
		Value: nil,
	}, {
		Op:    patch.PatchAddOp,
		Path:  "/spec/template/spec/accessCredentials",
		Value: []v1.AccessCredential{},
	}}
}

func patchToAddAccessCredential(credential *v1.AccessCredential) patch.PatchOperation {
	return patch.PatchOperation{
		Op:    patch.PatchAddOp,
		Path:  "/spec/template/spec/accessCredentials/-",
		Value: credential,
	}
}
