package addkey_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Credentials add-ssh-key", func() {
	const (
		vmName     = "test-vm"
		secretName = "test-secret"
		userName   = "test-user"
		testKey    = "test-key"
	)

	var (
		kubeClient *fake.Clientset
		virtClient *kubevirtfake.Clientset

		vmi *v1.VirtualMachineInstance
		vm  *v1.VirtualMachine
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()

		kubeClient.Fake.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created, ok := action.(k8stesting.CreateAction)
			Expect(ok).To(BeTrue())

			// GenerateName is not handled by default
			secret := created.GetObject().(*corev1.Secret)
			if secret.GenerateName != "" {
				secret.Name = secret.GenerateName + rand.String(6)
			}
			return false, secret, nil
		})

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		vmi = libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName(vmName),
			libvmi.WithAccessCredentialSSHPublicKey(secretName, userName),
		)
		var err error
		vmi, err = virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).
			Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).
			Create(context.Background(), libvmi.NewVirtualMachine(vmi), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: metav1.NamespaceDefault,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
					Kind:       v1.VirtualMachineGroupVersionKind.Kind,
					Name:       vm.Name,
					UID:        vm.UID,
					Controller: pointer.P(true),
				}},
			},
		}
		_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fail if no key is specified", func() {
		err := runAddKeyCommand(
			"--user", userName,
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("one of --file, or --value must be specified")))
	})

	It("should fail if key file does not exist", func() {
		err := runAddKeyCommand(
			"--user", userName,
			"--file", "some-nonexisting-file.pub",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
	})

	It("should fail if VM does not exist", func() {
		err := runAddKeyCommand(
			"--user", userName,
			"--value", "test-key",
			"nonexisting-vmi",
		)
		Expect(err).To(MatchError(ContainSubstring("\"nonexisting-vmi\" not found")))
	})

	It("should fail if no user is specified", func() {
		err := runAddKeyCommand(
			"--value", "test-key",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("required flag(s) \"user\" not set")))
	})

	It("should add a new secret, if no secret is specified for a user", func() {
		// VM is not running
		err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).
			Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		const newUser = "new-user"
		err = runAddKeyCommand(
			"--user", newUser,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		var newSecretName string
	outerLoop:
		for _, credential := range vm.Spec.Template.Spec.AccessCredentials {
			if credential.SSHPublicKey != nil &&
				credential.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
				for _, user := range credential.SSHPublicKey.PropagationMethod.QemuGuestAgent.Users {
					if user == newUser {
						newSecretName = credential.SSHPublicKey.Source.Secret.SecretName
						break outerLoop
					}
				}
			}
		}
		Expect(newSecretName).ToNot(BeEmpty())

		expectSecretToContainKey(kubeClient, newSecretName, testKey)
	})

	It("should fail if multiple secrets are specified for a user, and --secret parameter is not set", func() {
		appendToAccessCredentials(virtClient, vm, v1.AccessCredential{
			SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
				Source: v1.SSHPublicKeyAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: "second-secret",
					},
				},
				PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
						Users: []string{userName},
					},
				},
			},
		})

		err := runAddKeyCommand(
			"--user", userName,
			"--value", "test-key",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("multiple secrets specified")))
	})

	It("should patch secret", func() {
		err := runAddKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secretName, testKey)
	})

	It("should add second key to secret", func() {
		patchSecret(kubeClient, secretName, patch.WithReplace("/data", map[string][]byte{"old-key.pub": []byte("old-key")}))

		err := runAddKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		secret, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), secretName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(secret.Data).To(HaveLen(2))
		Expect(secret.Data).To(ContainElement([]byte(testKey)))
	})

	It("should patch secret with key from file", func() {
		const testKeyFile = "key contents in file"
		filename := filepath.Join(GinkgoT().TempDir(), "test-key-file")
		Expect(os.WriteFile(filename, []byte(testKeyFile), 0o600)).To(Succeed())

		err := runAddKeyCommand(
			"--user", userName,
			"--file", filename,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secretName, testKeyFile)
	})

	It("should patch the secret specified by parameter", func() {
		const secondSecretName = "second-secret"
		secondSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secondSecretName,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
					Kind:       v1.VirtualMachineGroupVersionKind.Kind,
					Name:       vm.Name,
					UID:        vm.UID,
					Controller: pointer.P(true),
				}},
			},
		}
		_, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secondSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		appendToAccessCredentials(virtClient, vm, v1.AccessCredential{
			SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
				Source: v1.SSHPublicKeyAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: secondSecretName,
					},
				},
				PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
						Users: []string{userName},
					},
				},
			},
		})

		err = runAddKeyCommand(
			"--user", userName,
			"--secret", secondSecretName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secondSecretName, testKey)
	})

	It("should not add key if secret already contains the key", func() {
		patchSecret(kubeClient, secretName, patch.WithReplace("/data", map[string][]byte{"some-name": []byte(testKey)}))

		err := runAddKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secretName, testKey)
	})

	It("should fail if secret is not owned by the VM", func() {
		patchSecret(kubeClient, secretName, patch.WithRemove("/metadata/ownerReferences"))

		err := runAddKeyCommand(
			"--user", userName,
			"--value", "test-key",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("does not have an owner reference pointing to VM")))
	})

	It("should patch secret not owned by VM, with --force option", func() {
		patchSecret(kubeClient, secretName, patch.WithRemove("/metadata/ownerReferences"))

		err := runAddKeyCommand(
			"--user", userName,
			"--value", testKey,
			"--force",
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secretName, testKey)
	})

	Context("with --create-secret set", func() {
		It("should create a new secret and patch VM", func() {
			// VM should not be running
			err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).
				Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runAddKeyCommand(
				"--user", userName,
				"--value", testKey,
				"--create-secret",
				vmName,
			)
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Template.Spec.AccessCredentials).To(HaveLen(2))

			credential := vm.Spec.Template.Spec.AccessCredentials[1]
			expectAccessCredentialIsSSHWithUser(&credential, userName)

			newSecretName := credential.SSHPublicKey.Source.Secret.SecretName
			expectSecretToContainKey(kubeClient, newSecretName, testKey)
		})

		It("should patch VM with nil AccessCredentials slice", func() {
			// VM should not be running
			err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).
				Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			payload, err := patch.New(patch.WithRemove("/spec/template/spec/accessCredentials")).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).
				Patch(context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runAddKeyCommand(
				"--user", userName,
				"--value", testKey,
				"--create-secret",
				vmName,
			)
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Template.Spec.AccessCredentials).To(HaveLen(1))

			credential := vm.Spec.Template.Spec.AccessCredentials[0]
			expectAccessCredentialIsSSHWithUser(&credential, userName)

			newSecretName := credential.SSHPublicKey.Source.Secret.SecretName
			expectSecretToContainKey(kubeClient, newSecretName, testKey)
		})

		It("should fail to add secret for running VM", func() {
			err := runAddKeyCommand(
				"--user", userName,
				"--value", "test-key",
				"--create-secret",
				vmName,
			)
			Expect(err).To(MatchError(ContainSubstring("virtual machine " + vmName + " is running")))
		})

		It("should add secret to running VM with --force option", func() {
			err := runAddKeyCommand(
				"--user", userName,
				"--value", testKey,
				"--create-secret",
				"--force",
				vmName,
			)
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Template.Spec.AccessCredentials).To(HaveLen(2))

			credential := vm.Spec.Template.Spec.AccessCredentials[1]
			expectAccessCredentialIsSSHWithUser(&credential, userName)

			newSecretName := credential.SSHPublicKey.Source.Secret.SecretName
			expectSecretToContainKey(kubeClient, newSecretName, testKey)
		})
	})

	Context("with --update-secret set", func() {
		It("should fail if no secret is specified for a user", func() {
			err := runAddKeyCommand(
				"--user", "unknown-user",
				"--value", "test-key",
				"--update-secret",
				vmName,
			)
			Expect(err).To(MatchError(ContainSubstring("no secrets specified for user")))
		})
	})
})

func appendToAccessCredentials(virtClient *kubevirtfake.Clientset, vm *v1.VirtualMachine, accessCredential v1.AccessCredential) {
	vm.Spec.Template.Spec.AccessCredentials = append(vm.Spec.Template.Spec.AccessCredentials, accessCredential)
	payload, err := patch.New(
		patch.WithReplace("/spec/template/spec/accessCredentials", vm.Spec.Template.Spec.AccessCredentials),
	).GeneratePayload()
	Expect(err).ToNot(HaveOccurred())
	_, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).
		Patch(context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func patchSecret(kubeClient kubernetes.Interface, name string, option patch.PatchOption) {
	payload, err := patch.New(option).GeneratePayload()
	Expect(err).ToNot(HaveOccurred())
	_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).
		Patch(context.Background(), name, types.JSONPatchType, payload, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func expectSecretToContainKey(kubeClient kubernetes.Interface, name, key string) {
	secret, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(secret.Data).To(HaveLen(1))
	Expect(secret.Data).To(ContainElement([]byte(key)))
}

func expectAccessCredentialIsSSHWithUser(credential *v1.AccessCredential, user string) {
	Expect(credential.SSHPublicKey).ToNot(BeNil())
	Expect(credential.SSHPublicKey.Source.Secret).ToNot(BeNil())
	Expect(credential.SSHPublicKey.Source.Secret.SecretName).ToNot(BeEmpty())
	Expect(credential.SSHPublicKey.PropagationMethod.QemuGuestAgent).ToNot(BeNil())
	Expect(credential.SSHPublicKey.PropagationMethod.QemuGuestAgent.Users).To(ContainElement(user))
}

func runAddKeyCommand(args ...string) error {
	return testing.NewRepeatableVirtctlCommand(append([]string{"credentials", "add-ssh-key"}, args...)...)()
}
