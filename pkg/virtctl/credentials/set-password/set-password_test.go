package set_password_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/testutils"

	"github.com/golang/mock/gomock"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Credentials set-password", func() {
	const (
		vmName     = "test-vm"
		secretName = "test-secret"
		userName   = "test-user"
	)

	var (
		kubeClient *fake.Clientset

		vm *v1.VirtualMachine
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()

		vmi := api.NewMinimalVMI(vmName)
		vmi.Spec.AccessCredentials = []v1.AccessCredential{{
			UserPassword: &v1.UserPasswordAccessCredential{
				Source: v1.UserPasswordAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{
						SecretName: secretName,
					},
				},
				PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
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

		vmInterface := kubecli.NewMockVirtualMachineInterface(ctrl)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		vmInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, name string, _ any) (*v1.VirtualMachine, error) {
			if name == vmName && vm != nil {
				return vm, nil
			}
			return nil, errors.NewNotFound(schema.GroupResource{
				Group:    core.GroupName,
				Resource: "VirtualMachine",
			}, name)
		}).AnyTimes()
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
		Expect(err).To(HaveOccurred())
	})

	It("should fail if no secret is specified", func() {
		vm.Spec.Template.Spec.AccessCredentials = nil
		err := runSetPasswordCommand(
			"--user", userName,
			"--password", "test-password",
			vmName,
		)
		Expect(err).To(MatchError(ContainSubstring("no secrets assigned to UserPassword AccessCredentials")))
	})

	It("should fail if multiple secrets are specified and --secret parameter is not set ", func() {
		vm.Spec.Template.Spec.AccessCredentials = append(vm.Spec.Template.Spec.AccessCredentials,
			v1.AccessCredential{
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
		const testPass = "test-pass"
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
					Controller: pointer.Bool(true),
				}},
			},
		}

		_, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secondSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Template.Spec.AccessCredentials = append(vm.Spec.Template.Spec.AccessCredentials,
			v1.AccessCredential{
				UserPassword: &v1.UserPasswordAccessCredential{
					Source: v1.UserPasswordAccessCredentialSource{
						Secret: &v1.AccessCredentialSecretSource{
							SecretName: secondSecretName,
						},
					},
					PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
						QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
					},
				}})

		const testPass = "test-pass"
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

func expectSecretToContainUserWithPassword(cli kubernetes.Interface, secretName string, user string, password string) {
	secret, err := cli.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), secretName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	ExpectWithOffset(1, secret.Data).To(HaveLen(1))
	ExpectWithOffset(1, secret.Data).To(HaveKeyWithValue(user, []byte(password)))
}

func runSetPasswordCommand(args ...string) error {
	return clientcmd.NewRepeatableVirtctlCommand(append([]string{"credentials", "set-password"}, args...)...)()
}

func TestSetPassword(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
