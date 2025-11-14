package addkey

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials/common"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand() *cobra.Command {
	cmdFlags := &addSSHKeyFlags{}
	cmd := &cobra.Command{
		Use:     "add-ssh-key",
		Short:   "Add credentials to a virtual machine.",
		Args:    cobra.ExactArgs(1),
		Example: exampleUsage,
		RunE:    cmdFlags.runAddKeyCommand,
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

type addSSHKeyFlags struct {
	common.SSHCommandFlags

	CreateSecret bool
	UpdateSecret bool

	Force bool
}

func (a *addSSHKeyFlags) AddToCommand(cmd *cobra.Command) {
	a.SSHCommandFlags.AddToCommand(cmd)

	const (
		createSecretFlag = "create-secret"
		updateSecretFlag = "update-secret"
	)

	cmd.Flags().BoolVar(&a.CreateSecret, createSecretFlag, false,
		"Create a new secret for the SSH key. The new key will not be added to a running VM. "+
			"Use --force to add a new secret even if the VM is running.")
	cmd.Flags().BoolVar(&a.UpdateSecret, updateSecretFlag, false,
		"Add the SSH key to an existing secret. Use --force option, if the secret does not have owner reference pointing to the VM.")
	cmd.MarkFlagsMutuallyExclusive(createSecretFlag, updateSecretFlag)

	cmd.Flags().BoolVar(&a.Force, "force", false, "Force update of secret, even if it's not owned by the VM.")
}

func (a *addSSHKeyFlags) runAddKeyCommand(cmd *cobra.Command, args []string) error {
	vmName := args[0]

	virtCli, k8sCli, vmNamespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("error getting kubevirt client or namespace: %w", err)
	}

	// Reading the key before accessing cluster
	sshKey, err := common.GetSSHKey(&a.SSHCommandFlags)
	if err != nil {
		return fmt.Errorf("error getting ssh key: %w", err)
	}

	vm, err := virtCli.VirtualMachine(vmNamespace).Get(cmd.Context(), vmName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting virtual machine: %w", err)
	}

	if a.shouldCreateNewSecret(vm) {
		return a.addSecretWithSSHKey(cmd, virtCli, k8sCli, vm, sshKey)
	}
	return a.updateSecretWithSSHKey(cmd, k8sCli, vm, sshKey)
}

func (a *addSSHKeyFlags) addSecretWithSSHKey(
	cmd *cobra.Command,
	cli kubecli.KubevirtClient,
	k8sCli kubernetes.Interface,
	vm *v1.VirtualMachine,
	sshKey string,
) error {
	if !a.Force {
		// Only create a secret if VM is not running.
		_, err := cli.VirtualMachineInstance(vm.Namespace).Get(cmd.Context(), vm.Name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf(
				"virtual machine %s is running. Use --force flag to update a running VM, it will take effect after restart",
				vm.Name)
		}
		if !errors.IsNotFound(err) {
			return fmt.Errorf("error when getting virtual machine instance: %w", err)
		}
	}

	secret := newSecretWithKey(vm, sshKey)
	secret, err := k8sCli.CoreV1().Secrets(vm.Namespace).Create(cmd.Context(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating secret: %w", err)
	}
	const (
		accessCredentialPath      = "/spec/template/spec/accessCredentials"   // #nosec
		accessCredentialArrayPath = "/spec/template/spec/accessCredentials/-" // #nosec
	)
	accessCredential := newAccessCredential(secret.Name, a.User)
	accessCredentialPatch, err := patch.New(patch.WithAdd(accessCredentialArrayPath, accessCredential)).GeneratePayload()
	if err != nil {
		return err
	}
	// First, Try to add the new access credential to the existing array.
	if _, err = cli.VirtualMachine(vm.Namespace).Patch(cmd.Context(),
		vm.Name,
		types.JSONPatchType,
		accessCredentialPatch,
		metav1.PatchOptions{}); err == nil {
		return nil
	}

	// If it fails, it probably means that the array is nil. Try to add the array.
	fullPatch, err := patch.New(
		patch.WithTest(accessCredentialPath, nil),
		patch.WithAdd(accessCredentialPath, []v1.AccessCredential{}),
		patch.WithAdd(accessCredentialArrayPath, accessCredential),
	).GeneratePayload()
	if err != nil {
		return err
	}
	if _, err = cli.VirtualMachine(vm.Namespace).Patch(cmd.Context(), vm.Name, types.JSONPatchType,
		fullPatch, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("error patching virtual machine: %w", err)
	}

	return nil
}

func (a *addSSHKeyFlags) updateSecretWithSSHKey(
	cmd *cobra.Command,
	k8sCli kubernetes.Interface,
	vm *v1.VirtualMachine,
	sshKey string,
) error {
	secrets := common.GetSSHSecretsForUser(vm.Spec.Template.Spec.AccessCredentials, a.User)
	if len(secrets) == 0 {
		return fmt.Errorf("no secrets specified for user: %s", a.User)
	}

	secretName, err := common.FindSecretOrGetFirst(a.Secret, secrets)
	if err != nil {
		return err
	}

	secret, err := k8sCli.CoreV1().Secrets(vm.Namespace).Get(cmd.Context(), secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting secret \"%s\": %w", secretName, err)
	}

	if secretContainsKey(secret.Data, sshKey) {
		cmd.Printf("Secret \"%s\" already contains this SSH key.", secretName)
		return nil
	}

	if !a.Force {
		// Check if secret is owned by the VM. This is useful to not accidentally update a secret that is used by multiple VMs.
		if !common.IsOwnedByVM(secret, vm) {
			return fmt.Errorf("secret %s does not have an owner reference pointing to VM %s", secretName, vm.Name)
		}
	}
	keyPath := fmt.Sprintf("/data/%s", common.RandomWithPrefix("ssh-key-"))
	addKeyPatch, err := patch.New(patch.WithAdd(keyPath, []byte(sshKey))).GeneratePayload()
	if err != nil {
		return err
	}
	// First, try patch to add a new key
	_, err = k8sCli.CoreV1().Secrets(vm.Namespace).Patch(
		cmd.Context(),
		secretName,
		types.JSONPatchType,
		addKeyPatch,
		metav1.PatchOptions{})
	if err != nil {
		// If it fails, the /data may be nil. Try a patch that adds the /data field
		fullPatch, err := patch.New(
			patch.WithTest("/data", nil),
			patch.WithAdd("/data", map[string][]byte{}),
			patch.WithAdd(keyPath, []byte(sshKey)),
		).GeneratePayload()
		if err != nil {
			return err
		}
		_, err = k8sCli.CoreV1().Secrets(vm.Namespace).Patch(cmd.Context(), secretName, types.JSONPatchType, fullPatch, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("error patching secret \"%s\": %w", secretName, err)
		}
	}

	cmd.Printf("Successfully added the key to secret \"%s\"", secretName)
	return nil
}

func (a *addSSHKeyFlags) shouldCreateNewSecret(vm *v1.VirtualMachine) bool {
	if a.CreateSecret {
		return true
	}
	if a.UpdateSecret {
		return false
	}

	// Default behavior: Create a new secret, if no secret is defined for a user
	secrets := common.GetSSHSecretsForUser(vm.Spec.Template.Spec.AccessCredentials, a.User)
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
				Controller: pointer.P(true),
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

func newAccessCredential(secretName, user string) *v1.AccessCredential {
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
