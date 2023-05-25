package add_key_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/pointer"
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/testutils"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Credentials add-ssh-key", func() {
	const (
		vmName     = "test-vm"
		secretName = "test-secret"
		userName   = "test-user"
	)

	var (
		kubeClient *fake.Clientset

		vmi *v1.VirtualMachineInstance
		vm  *v1.VirtualMachine
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()

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

		vmi = api.NewMinimalVMI(vmName)
		vmi.Spec.AccessCredentials = []v1.AccessCredential{{
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
		}}

		vm = kubecli.NewMinimalVM(vmName)
		vm.Namespace = metav1.NamespaceDefault
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: vmi.Spec,
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: metav1.NamespaceDefault,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
					Kind:       v1.VirtualMachineGroupVersionKind.Kind,
					Name:       vm.Name,
					UID:        vm.UID,
					Controller: pointer.Bool(true),
				}},
			},
		}

		_, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)

		vmiInterface := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface := kubecli.NewMockVirtualMachineInterface(ctrl)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		vmiInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, name string, _ any) (*v1.VirtualMachineInstance, error) {
			if name == vmName && vmi != nil {
				return vmi, nil
			}
			return nil, errors.NewNotFound(schema.GroupResource{
				Group:    core.GroupName,
				Resource: "VirtualMachineInstance",
			}, name)
		}).AnyTimes()

		vmInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, name string, _ any) (*v1.VirtualMachine, error) {
			if name == vmName && vm != nil {
				return vm, nil
			}
			return nil, errors.NewNotFound(schema.GroupResource{
				Group:    core.GroupName,
				Resource: "VirtualMachine",
			}, name)
		}).AnyTimes()

		vmInterface.EXPECT().Patch(gomock.Any(), gomock.Any(), types.JSONPatchType, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, name string, _ any, patchData []byte, _ any, _ ...any) (*v1.VirtualMachine, error) {
			if name != vmName || vm == nil {
				return nil, errors.NewNotFound(schema.GroupResource{
					Group:    core.GroupName,
					Resource: "VirtualMachine",
				}, name)
			}

			patch, err := jsonpatch.DecodePatch(patchData)
			if err != nil {
				return nil, err
			}

			vmJson, err := json.Marshal(vm)
			if err != nil {
				return nil, err
			}

			modifiedVmJson, err := patch.Apply(vmJson)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(modifiedVmJson, vm)
			if err != nil {
				return nil, err
			}

			return vm, nil
		}).AnyTimes()

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
		Expect(err).To(HaveOccurred())
	})

	It("should add a new secret, if no secret is specified for a user", func() {
		const testKey = "test-key"
		const newUser = "new-user"

		// VM is not running
		vmi = nil

		err := runAddKeyCommand(
			"--user", newUser,
			"--value", testKey,
			vmName,
		)
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
		vm.Spec.Template.Spec.AccessCredentials = append(vm.Spec.Template.Spec.AccessCredentials,
			v1.AccessCredential{
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
				}})

		err := runAddKeyCommand(
			"--user", userName,
			"--value", "test-key",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("multiple secrets specified")))
	})

	It("should patch secret", func() {
		const testKey = "test-key"
		err := runAddKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secretName, testKey)
	})

	It("should add second key to secret", func() {
		updateSecret(kubeClient, secretName, func(secret *corev1.Secret) {
			secret.Data = map[string][]byte{
				"old-key.pub": []byte("old-key"),
			}
		})

		const testKey = "test-key"
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
		const testKey = "key contents in file"

		filename := filepath.Join(GinkgoT().TempDir(), "test-key-file")
		Expect(os.WriteFile(filename, []byte(testKey), 0666)).To(Succeed())

		err := runAddKeyCommand(
			"--user", userName,
			"--file", filename,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secretName, testKey)
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
					Controller: pointer.Bool(true),
				}},
			},
		}

		_, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secondSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Template.Spec.AccessCredentials = append(vm.Spec.Template.Spec.AccessCredentials,
			v1.AccessCredential{
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
				}})

		const testKey = "test-key"
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
		const testKey = "test-key"

		updateSecret(kubeClient, secretName, func(secret *corev1.Secret) {
			secret.Data = map[string][]byte{
				"some-name": []byte(testKey),
			}
		})

		err := runAddKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainKey(kubeClient, secretName, testKey)
	})

	It("should fail if secret is not owned by the VM", func() {
		updateSecret(kubeClient, secretName, func(secret *corev1.Secret) {
			secret.OwnerReferences = nil
		})

		err := runAddKeyCommand(
			"--user", userName,
			"--value", "test-key",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("does not have an owner reference pointing to VM")))
	})

	It("should patch secret not owned by VM, with --force option", func() {
		updateSecret(kubeClient, secretName, func(secret *corev1.Secret) {
			secret.OwnerReferences = nil
		})

		const testKey = "test-key"
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
			vmi = nil

			const testKey = "test-key"
			err := runAddKeyCommand(
				"--user", userName,
				"--value", testKey,
				"--create-secret",
				vmName,
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.AccessCredentials).To(HaveLen(2))

			credential := vm.Spec.Template.Spec.AccessCredentials[1]
			expectAccessCredentialIsSshWithUser(&credential, userName)

			newSecretName := credential.SSHPublicKey.Source.Secret.SecretName
			expectSecretToContainKey(kubeClient, newSecretName, testKey)
		})

		It("should patch VM with nil AccessCredentials slice", func() {
			// VM should not be running
			vmi = nil

			vm.Spec.Template.Spec.AccessCredentials = nil

			const testKey = "test-key"
			err := runAddKeyCommand(
				"--user", userName,
				"--value", testKey,
				"--create-secret",
				vmName,
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.AccessCredentials).To(HaveLen(1))

			credential := vm.Spec.Template.Spec.AccessCredentials[0]
			expectAccessCredentialIsSshWithUser(&credential, userName)

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
			const testKey = "test-key"
			err := runAddKeyCommand(
				"--user", userName,
				"--value", testKey,
				"--create-secret",
				"--force",
				vmName,
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.AccessCredentials).To(HaveLen(2))

			credential := vm.Spec.Template.Spec.AccessCredentials[1]
			expectAccessCredentialIsSshWithUser(&credential, userName)

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

func expectSecretToContainKey(cli kubernetes.Interface, name string, key string) {
	secret, err := cli.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	ExpectWithOffset(1, secret.Data).To(HaveLen(1))
	ExpectWithOffset(1, secret.Data).To(ContainElement([]byte(key)))
}

func expectAccessCredentialIsSshWithUser(credential *v1.AccessCredential, user string) {
	Expect(credential.SSHPublicKey).ToNot(BeNil())
	Expect(credential.SSHPublicKey.Source.Secret).ToNot(BeNil())
	Expect(credential.SSHPublicKey.Source.Secret.SecretName).ToNot(BeEmpty())
	Expect(credential.SSHPublicKey.PropagationMethod.QemuGuestAgent).ToNot(BeNil())
	Expect(credential.SSHPublicKey.PropagationMethod.QemuGuestAgent.Users).To(ContainElement(user))
}

func updateSecret(cli kubernetes.Interface, name string, updateFunc func(secret *corev1.Secret)) {
	secret, err := cli.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	updateFunc(secret)

	_, err = cli.CoreV1().Secrets(metav1.NamespaceDefault).Update(context.Background(), secret, metav1.UpdateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func runAddKeyCommand(args ...string) error {
	return clientcmd.NewRepeatableVirtctlCommand(append([]string{"credentials", "add-ssh-key"}, args...)...)()
}

func TestAddSshKey(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
