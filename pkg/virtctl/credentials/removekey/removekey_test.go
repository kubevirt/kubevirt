package removekey_test

import (
	"context"
	"os"
	"path/filepath"

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
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Credentials", func() {
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

	const testKey = "test-key"

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
					Controller: pointer.P(true),
				}},
			},
			Data: map[string][]byte{
				"some-name": []byte(testKey),
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

		vmiInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, name string, _ any) (*v1.VirtualMachineInstance, error) {
				if name == vmName && vmi != nil {
					return vmi, nil
				}
				return nil, errors.NewNotFound(schema.GroupResource{
					Group:    core.GroupName,
					Resource: "VirtualMachineInstance",
				}, name)
			}).AnyTimes()

		vmInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, name string, _ any) (*v1.VirtualMachine, error) {
				if name == vmName && vm != nil {
					return vm, nil
				}
				return nil, errors.NewNotFound(schema.GroupResource{
					Group:    core.GroupName,
					Resource: "VirtualMachine",
				}, name)
			}).AnyTimes()

		vmInterface.EXPECT().Patch(gomock.Any(), gomock.Any(), types.JSONPatchType, gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, name string, _ any, patchData []byte, _ any, _ ...any) (*v1.VirtualMachine, error) {
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

				vmJSON, err := json.Marshal(vm)
				if err != nil {
					return nil, err
				}

				modifiedVMJSON, err := patch.Apply(vmJSON)
				if err != nil {
					return nil, err
				}

				err = json.Unmarshal(modifiedVMJSON, vm)
				if err != nil {
					return nil, err
				}

				return vm, nil
			}).AnyTimes()
	})

	It("should fail if no key is specified", func() {
		err := runRemoveKeyCommand(
			"--user", userName,
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("one of --file, or --value must be specified")))
	})

	It("should fail if key file does not exist", func() {
		err := runRemoveKeyCommand(
			"--user", userName,
			"--file", "some-nonexisting-file.pub",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
	})

	It("should fail if VM does not exist", func() {
		err := runRemoveKeyCommand(
			"--user", userName,
			"--value", testKey,
			"nonexisting-vmi",
		)
		Expect(err).To(MatchError(ContainSubstring("\"nonexisting-vmi\" not found")))
	})

	It("should fail if no user is specified", func() {
		err := runRemoveKeyCommand(
			"--value", testKey,
			vmName,
		)
		Expect(err).To(HaveOccurred())
	})

	It("should remove key from secret", func() {
		err := runRemoveKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToBeEmpty(kubeClient, secretName)
	})

	It("should remove key from secret with multiple keys", func() {
		const secondDataKey = "second-key"
		const secondDataValue = "second-key-value"
		const thirdKeyValue = "third-key"
		const multipleKeys = "multiple-keys"

		updateSecret(kubeClient, secretName, func(secret *corev1.Secret) {
			secret.Data[secondDataKey] = []byte(secondDataValue)
			secret.Data[multipleKeys] = []byte(testKey + "\n" + thirdKeyValue)
		})

		err := runRemoveKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		secret, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), secretName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(secret.Data).To(HaveLen(2))
		Expect(secret.Data).To(HaveKeyWithValue(secondDataKey, []byte(secondDataValue)))
		Expect(secret.Data).To(HaveKeyWithValue(multipleKeys, []byte(thirdKeyValue)))
	})

	It("should remove key from secret with key from file", func() {
		filename := filepath.Join(GinkgoT().TempDir(), "test-key-file")
		Expect(os.WriteFile(filename, []byte(testKey), 0o600)).To(Succeed())

		err := runRemoveKeyCommand(
			"--user", userName,
			"--file", filename,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToBeEmpty(kubeClient, secretName)
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
			Data: map[string][]byte{
				"test-file": []byte(testKey),
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
				},
			})

		err = runRemoveKeyCommand(
			"--user", userName,
			"--secret", secondSecretName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToBeEmpty(kubeClient, secondSecretName)
	})

	It("should fail if secret is not owned by the VM", func() {
		updateSecret(kubeClient, secretName, func(secret *corev1.Secret) {
			secret.OwnerReferences = nil
		})

		err := runRemoveKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("does not have an owner reference pointing to VM")))
	})

	It("should patch secret not owned by VM, with --force option", func() {
		updateSecret(kubeClient, secretName, func(secret *corev1.Secret) {
			secret.OwnerReferences = nil
		})

		err := runRemoveKeyCommand(
			"--user", userName,
			"--value", testKey,
			"--force",
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToBeEmpty(kubeClient, secretName)
	})
})

func updateSecret(cli kubernetes.Interface, name string, updateFunc func(secret *corev1.Secret)) {
	secret, err := cli.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	updateFunc(secret)

	_, err = cli.CoreV1().Secrets(metav1.NamespaceDefault).Update(context.Background(), secret, metav1.UpdateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func expectSecretToBeEmpty(cli kubernetes.Interface, name string) {
	secret, err := cli.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	ExpectWithOffset(1, secret.Data).To(BeEmpty())
}

func runRemoveKeyCommand(args ...string) error {
	return clientcmd.NewRepeatableVirtctlCommand(append([]string{"credentials", "remove-ssh-key"}, args...)...)()
}
