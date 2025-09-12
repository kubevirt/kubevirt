package password_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Credentials set-password", func() {
	const (
		vmName     = "test-vm"
		secretName = "test-secret"
		userName   = "test-user"
		testPass   = "test-pass"
	)

	var (
		kubeClient *fake.Clientset
		virtClient *kubevirtfake.Clientset

		vm *v1.VirtualMachine
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.GetK8sClientFromClientConfig = kubecli.GetMockK8sClientFromClientConfig
		kubecli.MockK8sClientInstance = kubeClient
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		vmi := libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName(vmName),
			libvmi.WithAccessCredentialUserPassword(secretName),
		)
		var err error
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

	It("should fail if no password is specified", func() {
		err := runSetPasswordCommand(
			"--user", userName,
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("required flag(s) \"password\" not set")))
	})

	It("should fail if VMI or VM do not exist", func() {
		err := runSetPasswordCommand(
			"--user", userName,
			"--password", "test-pass",
			"nonexisting-vmi",
		)
		Expect(err).To(MatchError(ContainSubstring("\"nonexisting-vmi\" not found")))
	})

	It("should fail if no user is specified", func() {
		err := runSetPasswordCommand(
			"--password", "test-pass",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("required flag(s) \"user\" not set")))
	})

	It("should fail if no secret is specified", func() {
		payload, err := patch.New(patch.WithRemove("/spec/template/spec/accessCredentials")).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).
			Patch(context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = runSetPasswordCommand(
			"--user", userName,
			"--password", "test-password",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("no secrets assigned to UserPassword AccessCredentials")))
	})

	It("should fail if multiple secrets are specified and --secret parameter is not set ", func() {
		appendToAccessCredentials(virtClient, vm, v1.AccessCredential{
			UserPassword: &v1.UserPasswordAccessCredential{
				Source: v1.UserPasswordAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: "secret-2",
					},
				},
				PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
				},
			},
		})

		err := runSetPasswordCommand(
			"--user", userName,
			"--password", "test-password",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("multiple secrets specified")))
	})

	It("should patch secret", func() {
		err := runSetPasswordCommand(
			"--user", userName,
			"--password", testPass,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainUserWithPassword(kubeClient, secretName, userName, testPass)
	})

	It("should patch the secret specified by parameter", func() {
		const secondSecretName = "second-secret"
		secondSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secondSecretName,
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
		_, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secondSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		appendToAccessCredentials(virtClient, vm, v1.AccessCredential{
			UserPassword: &v1.UserPasswordAccessCredential{
				Source: v1.UserPasswordAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: secondSecretName,
					},
				},
				PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
				},
			},
		})

		err = runSetPasswordCommand(
			"--user", userName,
			"--secret", secondSecretName,
			"--password", testPass,
			vmName,
		)
		Expect(err).ToNot(HaveOccurred())

		expectSecretToContainUserWithPassword(kubeClient, secondSecretName, userName, testPass)
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

func expectSecretToContainUserWithPassword(kubeClient kubernetes.Interface, secretName, user, password string) {
	secret, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), secretName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(secret.Data).To(HaveLen(1))
	Expect(secret.Data).To(HaveKeyWithValue(user, []byte(password)))
}

func runSetPasswordCommand(args ...string) error {
	return testing.NewRepeatableVirtctlCommand(append([]string{"credentials", "set-password"}, args...)...)()
}
