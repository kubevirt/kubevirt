/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package compute

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Guest Access Credentials", func() {

	const (
		fedoraRunningTimeout     = libvmops.StartupTimeoutSecondsLarge
		guestAgentConnectTimeout = 2 * time.Minute
		denyListTimeout          = 2 * time.Minute
		fedoraPassword           = "fedora"
		pubKeySecretID           = "my-pub-key"
		userPassSecretID         = "my-user-pass"
		userData                 = "#cloud-config\nchpasswd: { expire: False }\n"
	)

	keysSecretData := libsecret.DataBytes{
		"my-key1": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"),
		"my-key2": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key2"),
		"my-key3": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key3"),
	}

	DescribeTable("should have ssh-key under authorized keys added", func(withQEMUAccessCredential bool, options ...libvmi.Option) {
		By("Creating a secret with three ssh keys")
		Expect(createNewSecret(testsuite.GetTestNamespace(nil), pubKeySecretID, keysSecretData)).To(Succeed())

		vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewFedora(options...), fedoraRunningTimeout)

		By("Waiting for agent to connect")
		Eventually(matcher.ThisVMI(vmi), guestAgentConnectTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		if withQEMUAccessCredential {
			By("Waiting on access credentials to sync")
			// this ensures the keys have propagated before we attempt to read
			Eventually(matcher.ThisVMI(vmi), denyListTimeout, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAccessCredentialsSynchronized))
		}

		By("Verifying all three pub ssh keys in secret are in VMI guest")
		Expect(console.ExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: "login:"},
			&expect.BSnd{S: "fedora\n"},
			&expect.BExp{R: "Password:"},
			&expect.BSnd{S: fedoraPassword + "\n"},
			&expect.BExp{R: "\\$"},
			&expect.BSnd{S: "cat /home/fedora/.ssh/authorized_keys\n"},
			&expect.BExp{R: "test-ssh-key1"},
			&expect.BSnd{S: "cat /home/fedora/.ssh/authorized_keys\n"},
			&expect.BExp{R: "test-ssh-key2"},
			&expect.BSnd{S: "cat /home/fedora/.ssh/authorized_keys\n"},
			&expect.BExp{R: "test-ssh-key3"},
		}, 3*time.Minute)).To(Succeed())
	},
		Entry("[test_id:6220] using qemu guest agent", decorators.Conformance, true,
			withSSHPK(pubKeySecretID, v1.SSHPublicKeyAccessCredentialPropagationMethod{
				QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
					Users: []string{"fedora"},
				},
			}),
		),
		Entry("[test_id:6224] using configdrive", decorators.Conformance, false,
			libvmi.WithCloudInitConfigDrive(libvmici.WithConfigDriveUserData(userData)),
			withSSHPK(pubKeySecretID, v1.SSHPublicKeyAccessCredentialPropagationMethod{
				ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
			}),
		),
		Entry("using nocloud", decorators.Conformance, false,
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(userData)),
			withSSHPK(pubKeySecretID, v1.SSHPublicKeyAccessCredentialPropagationMethod{
				NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
			}),
		),
	)

	Context("with qemu guest agent", func() {
		const customPassword = "imadethisup"

		pubKeyData := libsecret.DataBytes{
			"my-key1": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"),
		}

		userPassData := libsecret.DataBytes{
			"fedora": []byte(customPassword),
		}

		It("[test_id:6221]should propagate user password", func() {
			vmi := libvmifact.NewFedora(libvmi.WithAccessCredentialUserPassword(userPassSecretID))

			By("Creating a secret with custom password")
			Expect(createNewSecret(testsuite.GetTestNamespace(vmi), userPassSecretID, userPassData)).To(Succeed())

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, fedoraRunningTimeout)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), guestAgentConnectTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Waiting on access credentials to sync")
			Eventually(matcher.ThisVMI(vmi), denyListTimeout, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAccessCredentialsSynchronized))

			By("Verifying signin with custom password works")
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "login:"},
				&expect.BSnd{S: "fedora\n"},
				&expect.BExp{R: "Password:"},
				&expect.BSnd{S: customPassword + "\n"},
				&expect.BExp{R: "\\$"},
			}, 3*time.Minute)).To(Succeed())

		})

		DescribeTable("should update to unsupported agent", func(secretID string, secretData libsecret.DataBytes, options ...libvmi.Option) {
			vmi := libvmifact.NewFedora(options...)

			By("Creating a secret with an ssh key")
			Expect(createNewSecret(testsuite.GetTestNamespace(vmi), secretID, secretData)).To(Succeed())

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, fedoraRunningTimeout)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), guestAgentConnectTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that denylisted commands triggered unsupported guest agent condition")
			Eventually(matcher.ThisVMI(vmi), denyListTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceUnsupportedAgent))

		},
			Entry("[test_id:6222]for public ssh keys", pubKeySecretID, pubKeyData,
				withSSHPK(pubKeySecretID, v1.SSHPublicKeyAccessCredentialPropagationMethod{
					QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
						Users: []string{"fedora"},
					},
				}),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(
					cloudinit.GetFedoraToolsGuestAgentBlacklistUserData("guest-exec,guest-ssh-add-authorized-keys"),
				))),
			Entry("[test_id:6223] for user password", userPassSecretID, userPassData,
				libvmi.WithAccessCredentialUserPassword(userPassSecretID),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(
					cloudinit.GetFedoraToolsGuestAgentBlacklistUserData("guest-set-user-password"),
				))),
		)
	})
}))

func createNewSecret(namespace string, name string, data libsecret.DataBytes) error {
	secret := libsecret.New(name, data)
	_, err := kubevirt.Client().CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}

func withSSHPK(secretName string, propagationMethod v1.SSHPublicKeyAccessCredentialPropagationMethod) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.AccessCredentials = []v1.AccessCredential{
			{
				SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
					Source: v1.SSHPublicKeyAccessCredentialSource{
						Secret: &v1.AccessCredentialSecretSource{
							SecretName: secretName,
						},
					},
					PropagationMethod: propagationMethod,
				},
			},
		}
	}
}
