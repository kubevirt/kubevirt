package virtctl

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute][virtctl]credentials", func() {
	const (
		testKey1 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR2Ah+NcKPU9wDXP7DibuXrkvXCL/YH/w++3M3zZK27WSfjngsawM/Kai8oGXwmjFCprP77COkdBqg2Dpr/ulQ/7h4GwVb/Cjcwov/LOWg5aRAUa1NYRZ75CErMuGW9kSAd42mxeSslLK91hdlCFJP3qMPbkTvlrGAw+6WzwQEmQA1S1D7KC1yJTW6gtgkkKVYNnOhvuGDrCzoOyxb1SfjAhKSk3OkkotdBlWK8TWynGkYhptLAP9pQvCgtRMJPBQ6OWjVV5qkT6yY2hjG6frYnwDotI5OXdOBjbx0Oaa3sFRC983YDIh9lbEKeQxckykg9Iys2fT/NZUbze46hSA/8bG4hDqU0X7+dHN+Ite2/vRjEeaRaWzm9t7+/nxzxibr2x38fkxtNwGYv6VHTyoBTVj/mVqku+NM7pzGGD5X2nB28gbJTCnRPtd4kLIHfg7IYjfHpIBXwfq5jnRlYrIraqkEljZ6iAF4xZGQkQYZQhhwNErJ4+cOFadwG11pdhs= test-key-1"
		testKey2 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCx0qmeOuJUi9Sh05Aq/OqxKA4gY+ZuJMEFG9eKQA0nCyb+yBXmVg3T0Leg9JJl5wzygSHyeDstuB6kdGzspufTRQ2YXf0RqlaRfSdc06LlDoc1af4Z6W3Hy4NMwK/nQR9b4Dx8mDLgnxqjueIOu3yZN3ZGr7xsZr+dsygPJQfLSGzbMQ71U/Rh9ETvIU8/aY0hVWb0rMpnQ0X1NBDfqqwSAx9I3kdn1TWkaIDM++lB+g02QsKkTj/MOFBa9gweI0jmjFbbGfwKrTUFLTNYr5M80/Qoj2/KPMEhlIQMBMTNPS9EtgqzlPZyj7Bnmh1UYdMcqYklOhqOJ/rXNlcAIlkA/MMpb/LMCLQUvJuJ51fPaZIqBqxtvY9wVs+CtpjWmouBmjtKe57EadCTyTjuZkxQihTzINyXETjw9U0wnaMJQVhexTjZmR6p7Utz+MoU0R12gfQUirVYX3zQdSQbe/aqX6vbuct+/zoWjkQCdoGABkBP7Y4/FFnBd4hnVJaRRes= test-key-2"
	)

	const (
		sshKeySecretNamePrefix   = "test-secret-ssh-"
		passwordSecretNamePrefix = "test-secret-password-"
		userName                 = "root"
	)

	var (
		cli kubecli.KubevirtClient

		sshKeySecretName   string
		passwordSecretName string
		vm                 *kubevirtv1.VirtualMachine
	)

	BeforeEach(func() {
		cli = kubevirt.Client()

		sshKeySecretName = sshKeySecretNamePrefix + rand.String(6)
		passwordSecretName = passwordSecretNamePrefix + rand.String(6)

		vmi := libvmi.NewFedora()
		vmi.Namespace = util.NamespaceTestDefault

		vm = tests.NewRandomVirtualMachine(vmi, false)

		vm.Spec.Template.Spec.AccessCredentials = []kubevirtv1.AccessCredential{{
			SSHPublicKey: &kubevirtv1.SSHPublicKeyAccessCredential{
				Source: kubevirtv1.SSHPublicKeyAccessCredentialSource{
					Secret: &kubevirtv1.AccessCredentialSecretSource{
						SecretName: sshKeySecretName,
					},
				},
				PropagationMethod: kubevirtv1.SSHPublicKeyAccessCredentialPropagationMethod{
					QemuGuestAgent: &kubevirtv1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
						Users: []string{userName},
					},
				},
			},
		}, {
			UserPassword: &kubevirtv1.UserPasswordAccessCredential{
				Source: kubevirtv1.UserPasswordAccessCredentialSource{
					Secret: &kubevirtv1.AccessCredentialSecretSource{
						SecretName: passwordSecretName,
					},
				},
				PropagationMethod: kubevirtv1.UserPasswordAccessCredentialPropagationMethod{
					QemuGuestAgent: &kubevirtv1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
				},
			},
		}}

		vm, err := cli.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), vm)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			err := cli.VirtualMachine(util.NamespaceTestDefault).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
			Expect(err).To(Or(
				Not(HaveOccurred()),
				Satisfy(errors.IsNotFound),
			))
		})

		keySecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: sshKeySecretName,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: kubevirtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
					Kind:       kubevirtv1.VirtualMachineGroupVersionKind.Kind,
					Name:       vm.Name,
					UID:        vm.UID,
					Controller: pointer.Bool(true),
				}},
			},
			Data: map[string][]byte{
				"key-file.pub": []byte(testKey1),
			},
		}

		keySecret, err = cli.CoreV1().Secrets(util.NamespaceTestDefault).Create(context.Background(), keySecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Delete(context.Background(), keySecret.Name, metav1.DeleteOptions{})
			Expect(err).To(Or(
				Not(HaveOccurred()),
				Satisfy(errors.IsNotFound),
			))
		})

		passwordSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: passwordSecretName,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: kubevirtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
					Kind:       kubevirtv1.VirtualMachineGroupVersionKind.Kind,
					Name:       vm.Name,
					UID:        vm.UID,
					Controller: pointer.Bool(true),
				}},
			},
			Data: map[string][]byte{
				userName: []byte("test-password"),
			},
		}

		passwordSecret, err = cli.CoreV1().Secrets(util.NamespaceTestDefault).Create(context.Background(), passwordSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Delete(context.Background(), passwordSecret.Name, metav1.DeleteOptions{})
			Expect(err).To(Or(
				Not(HaveOccurred()),
				Satisfy(errors.IsNotFound),
			))
		})
	})

	Context("add-ssh-key", func() {
		It("[test_id:TODO] should add ssh key to a secret", func() {
			err := clientcmd.NewRepeatableVirtctlCommand(
				"credentials", "add-ssh-key",
				"--user", userName,
				"--value", testKey2,
				"--namespace", util.NamespaceTestDefault,
				vm.Name,
			)()
			Expect(err).ToNot(HaveOccurred())

			secret, err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Get(context.Background(), sshKeySecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(secret.Data).To(HaveLen(2))
			Expect(secret.Data).To(ContainElement([]byte(testKey2)))
		})

		It("[test_id:TODO] should add ssh key from a file", func() {
			filename := filepath.Join(GinkgoT().TempDir(), "test-key-file.pub")
			Expect(os.WriteFile(filename, []byte(testKey2), 0666)).To(Succeed())

			err := clientcmd.NewRepeatableVirtctlCommand(
				"credentials", "add-ssh-key",
				"--user", userName,
				"--file", filename,
				"--namespace", util.NamespaceTestDefault,
				vm.Name,
			)()
			Expect(err).ToNot(HaveOccurred())

			secret, err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Get(context.Background(), sshKeySecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(secret.Data).To(HaveLen(2))
			Expect(secret.Data).To(ContainElement([]byte(testKey2)))
		})

		It("[test_id:TODO] should create a new secret and patch the VM", func() {
			const newUser = "fedora"

			err := clientcmd.NewRepeatableVirtctlCommand(
				"credentials", "add-ssh-key",
				"--user", newUser,
				"--value", testKey2,
				"--create-secret",
				"--namespace", util.NamespaceTestDefault,
				vm.Name,
			)()
			Expect(err).ToNot(HaveOccurred())

			vm, err := cli.VirtualMachine(util.NamespaceTestDefault).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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

			secret, err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Get(context.Background(), newSecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(secret.Data).To(HaveLen(1))
			Expect(secret.Data).To(ContainElement([]byte(testKey2)))
		})
	})

	Context("remove-ssh-key", func() {
		It("[test_id:TODO] should remove ssh key from a secret", func() {
			err := clientcmd.NewRepeatableVirtctlCommand(
				"credentials", "remove-ssh-key",
				"--user", userName,
				"--value", testKey1,
				"--namespace", util.NamespaceTestDefault,
				vm.Name,
			)()
			Expect(err).ToNot(HaveOccurred())

			secret, err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Get(context.Background(), sshKeySecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(secret.Data).To(BeEmpty())
		})

		It("[test_id:TODO] should remove ssh key read from file", func() {
			filename := filepath.Join(GinkgoT().TempDir(), "test-key-file.pub")
			Expect(os.WriteFile(filename, []byte(testKey1), 0666)).To(Succeed())

			err := clientcmd.NewRepeatableVirtctlCommand(
				"credentials", "remove-ssh-key",
				"--user", userName,
				"--file", filename,
				"--namespace", util.NamespaceTestDefault,
				vm.Name,
			)()
			Expect(err).ToNot(HaveOccurred())

			secret, err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Get(context.Background(), sshKeySecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(secret.Data).To(BeEmpty())
		})
	})

	Context("set-password", func() {
		It("[test_id:TODO] should set password for a user", func() {
			const newPassword = "new-password"
			err := clientcmd.NewRepeatableVirtctlCommand(
				"credentials", "set-password",
				"--user", userName,
				"--password", newPassword,
				"--namespace", util.NamespaceTestDefault,
				vm.Name,
			)()
			Expect(err).ToNot(HaveOccurred())

			secret, err := cli.CoreV1().Secrets(util.NamespaceTestDefault).Get(context.Background(), passwordSecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(secret.Data).To(HaveLen(1))
			Expect(secret.Data).To(HaveKeyWithValue(userName, []byte(newPassword)))
		})
	})
})
