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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package compute

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"

	expect "github.com/google/goexpect"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = SIGDescribe("Guest Access Credentials", func() {

	const (
		fedoraRunningTimeout    = 120
		guestAgentConnetTimeout = 2 * time.Minute
		denyListTimeout         = 2 * time.Minute
		fedoraPassword          = "fedora"
	)

	Context("with qemu guest agent", func() {
		withQuestAgentPropagationMethod := v1.SSHPublicKeyAccessCredentialPropagationMethod{
			QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
				Users: []string{"fedora"},
			},
		}

		withPassword := func(secretName string) libvmi.Option {
			return func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.AccessCredentials = append(vmi.Spec.AccessCredentials,
					v1.AccessCredential{
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
					})
			}
		}

		It("[test_id:6220]should propagate public ssh keys", func() {
			const secretID = "my-pub-key"
			vmi := libvmi.NewFedora(withSSHPK(secretID, withQuestAgentPropagationMethod))

			By("Creating a secret with three ssh keys")
			createNewSecret(secretID, map[string][]byte{
				"my-key1": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"),
				"my-key2": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key2"),
				"my-key3": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key3"),
			})

			vmi = tests.RunVMIAndExpectLaunch(vmi, fedoraRunningTimeout)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), guestAgentConnetTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Waiting on access credentials to sync")
			// this ensures the keys have propagated before we attempt to read
			Eventually(matcher.ThisVMI(vmi), denyListTimeout, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAccessCredentialsSynchronized))

			By("Verifying all three pub ssh keys in secret are in VMI guest")
			Expect(console.ExpectBatch(vmi,
				[]expect.Batcher{
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
		})

		It("[test_id:6221]should propagate user password", func() {
			const secretID = "my-user-pass"
			vmi := libvmi.NewFedora(withPassword(secretID))

			customPassword := "imadethisup"

			By("Creating a secret with custom password")
			createNewSecret(secretID, map[string][]byte{
				"fedora": []byte(customPassword),
			})

			vmi = tests.RunVMIAndExpectLaunch(vmi, fedoraRunningTimeout)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), guestAgentConnetTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Waiting on access credentials to sync")
			// this ensures the keys have propagated before we attempt to read
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

		It("[test_id:6222]should update guest agent for public ssh keys", func() {
			const secretID = "my-pub-key"
			vmi := libvmi.NewFedora(
				withSSHPK(secretID, withQuestAgentPropagationMethod),
				libvmi.WithCloudInitNoCloudUserData(
					tests.GetFedoraToolsGuestAgentBlacklistUserData("guest-exec"), false,
				),
			)

			By("Creating a secret with an ssh key")
			createNewSecret(secretID, map[string][]byte{
				"my-key1": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"),
			})

			vmi = tests.RunVMIAndExpectLaunch(vmi, fedoraRunningTimeout)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), guestAgentConnetTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that denylisted commands triggered unsupported guest agent condition")
			Eventually(matcher.ThisVMI(vmi), denyListTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceUnsupportedAgent))
		})

		It("[test_id:6223]should update guest agent for user password", func() {
			const secretID = "my-user-pass"
			vmi := libvmi.NewFedora(
				withPassword(secretID),
				libvmi.WithCloudInitNoCloudUserData(tests.GetFedoraToolsGuestAgentBlacklistUserData("guest-set-user-password"), false),
			)

			customPassword := "imadethisup"

			By("Creating a secret with custom password")
			createNewSecret(secretID, map[string][]byte{
				"fedora": []byte(customPassword),
			})
			vmi = tests.RunVMIAndExpectLaunch(vmi, fedoraRunningTimeout)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), guestAgentConnetTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that denylisted commands triggered unsupported guest agent condition")
			Eventually(matcher.ThisVMI(vmi), denyListTimeout, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceUnsupportedAgent))
		})
	})
	Context("with secret and cloudInit propagation", func() {
		const secretID = "my-pub-key"
		userData := fmt.Sprintf(
			"#cloud-config\npassword: %s\nchpasswd: { expire: False }\n",
			fedoraPassword,
		)

		verifySSHKeys := func(vmi *v1.VirtualMachineInstance) {
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
		}

		DescribeTable("should have ssh-key under authorized keys added ", func(propagationMethod v1.SSHPublicKeyAccessCredentialPropagationMethod) {
			By("Creating a secret with three ssh keys")
			createNewSecret(secretID, map[string][]byte{
				"my-key1": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"),
				"my-key2": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key2"),
				"my-key3": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key3"),
			})

			vmi := libvmi.NewFedora(
				libvmi.WithCloudInitConfigDriveData(userData, false),
				withSSHPK(secretID, propagationMethod))
			vmi = tests.RunVMIAndExpectLaunch(vmi, fedoraRunningTimeout)
			verifySSHKeys(vmi)
		},
			Entry("[test_id:6224]using configdrive", v1.SSHPublicKeyAccessCredentialPropagationMethod{
				ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
			}),
			Entry("using nocloud", v1.SSHPublicKeyAccessCredentialPropagationMethod{
				NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
			}),
		)
	})
})

func createNewSecret(secretID string, data map[string][]byte) {
	secret := &kubev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretID,
			Labels: map[string]string{
				util.SecretLabel: secretID,
			},
		},
		Type: "Opaque",
		Data: data,
	}
	_, err := kubevirt.Client().CoreV1().Secrets(util.NamespaceTestDefault).Create(context.Background(), secret, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
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
