package removekey_test

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

var _ = Describe("Credentials remove-ssh-key", func() {
	const (
		vmName     = "test-vm"
		secretName = "test-secret"
		userName   = "test-user"
		testKey    = "test-key"
	)

	var (
		kubeClient *fake.Clientset
		virtClient *kubevirtfake.Clientset

		vmi    *v1.VirtualMachineInstance
		vm     *v1.VirtualMachine
		secret *corev1.Secret
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
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
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

		secret = &corev1.Secret{
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

		secret, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
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
		Expect(err).To(MatchError(ContainSubstring("required flag(s) \"user\" not set")))
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

		secret.Data[secondDataKey] = []byte(secondDataValue)
		secret.Data[multipleKeys] = []byte(testKey + "\n" + thirdKeyValue)
		patchSecret(kubeClient, secret.Name, patch.WithReplace("/data", secret.Data))

		err := runRemoveKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		secret, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), secretName, metav1.GetOptions{})
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
		payload, err := patch.New(
			patch.WithReplace("/spec/template/spec/accessCredentials", vm.Spec.Template.Spec.AccessCredentials),
		).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).
			Patch(context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

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
		patchSecret(kubeClient, secret.Name, patch.WithRemove("/metadata/ownerReferences"))

		err := runRemoveKeyCommand(
			"--user", userName,
			"--value", testKey,
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("does not have an owner reference pointing to VM")))
	})

	It("should patch secret not owned by VM, with --force option", func() {
		patchSecret(kubeClient, secret.Name, patch.WithRemove("/metadata/ownerReferences"))

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

func patchSecret(kubeClient kubernetes.Interface, name string, option patch.PatchOption) {
	payload, err := patch.New(option).GeneratePayload()
	Expect(err).ToNot(HaveOccurred())
	_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).
		Patch(context.Background(), name, types.JSONPatchType, payload, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func expectSecretToBeEmpty(kubeClient kubernetes.Interface, name string) {
	secret, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(secret.Data).To(BeEmpty())
}

func runRemoveKeyCommand(args ...string) error {
	return testing.NewRepeatableVirtctlCommand(append([]string{"credentials", "remove-ssh-key"}, args...)...)()
}
