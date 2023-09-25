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

package tests_test

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"

	expect "github.com/google/goexpect"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[sig-compute]Guest Access Credentials", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient

	var (
		ExecutingBatchCmd func(*v1.VirtualMachineInstance, []expect.Batcher, time.Duration)
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	ExecutingBatchCmd = func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
		By("Checking that the VirtualMachineInstance serial console output equals to expected one")
		err := console.ExpectBatch(vmi, commands, timeout)
		Expect(err).ToNot(HaveOccurred())
	}

	Context("with qemu guest agent", func() {
		It("[test_id:6220]should propagate public ssh keys", func() {
			secretID := "my-pub-key"
			vmi := tests.NewRandomFedoraVMI()
			vmi.Namespace = util.NamespaceTestDefault
			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: secretID,
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
								Users: []string{"fedora"},
							},
						},
					},
				},
			}

			key1 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"
			key2 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key2"
			key3 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key3"

			By("Creating a secret with three ssh keys")
			secret := kubev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretID,
					Namespace: vmi.Namespace,
					Labels: map[string]string{
						util.SecretLabel: secretID,
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"my-key1": []byte(key1),
					"my-key2": []byte(key2),
					"my-key3": []byte(key3),
				},
			}
			_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Waiting on access credentials to sync")
			// this ensures the keys have propagated before we attempt to read
			Eventually(matcher.ThisVMI(vmi), 45*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAccessCredentialsSynchronized))

			By("Verifying all three pub ssh keys in secret are in VMI guest")
			ExecutingBatchCmd(vmi, []expect.Batcher{
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
			}, time.Second*180)
		})

		It("[test_id:6221]should propagate user password", func() {
			secretID := "my-user-pass"
			vmi := tests.NewRandomFedoraVMI()
			vmi.Namespace = util.NamespaceTestDefault

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					UserPassword: &v1.UserPasswordAccessCredential{
						Source: v1.UserPasswordAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: secretID,
							},
						},
						PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
						},
					},
				},
			}

			customPassword := "imadethisup"

			By("Creating a secret with custom password")
			secret := kubev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretID,
					Namespace: vmi.Namespace,
					Labels: map[string]string{
						util.SecretLabel: secretID,
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"fedora": []byte(customPassword),
				},
			}
			_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Verifying signin with custom password works")

			By("Waiting on access credentials to sync")
			// this ensures the keys have propagated before we attempt to read
			Eventually(matcher.ThisVMI(vmi), 45*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAccessCredentialsSynchronized))
			ExecutingBatchCmd(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "login:"},
				&expect.BSnd{S: "fedora\n"},
				&expect.BExp{R: "Password:"},
				&expect.BSnd{S: customPassword + "\n"},
				&expect.BExp{R: "\\$"},
			}, time.Second*180)
		})

		It("[test_id:6222]should update guest agent for public ssh keys", func() {
			secretID := "my-pub-key"
			vmi := tests.NewRandomFedoraVMIWithBlacklistGuestAgent("guest-exec")
			vmi.Namespace = util.NamespaceTestDefault
			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: secretID,
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
								Users: []string{"fedora"},
							},
						},
					},
				},
			}

			key1 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"

			By("Creating a secret with an ssh key")
			secret := kubev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretID,
					Namespace: vmi.Namespace,
					Labels: map[string]string{
						util.SecretLabel: secretID,
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"my-key1": []byte(key1),
				},
			}
			_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that denylisted commands triggered unsupported guest agent condition")
			Eventually(matcher.ThisVMI(vmi), 240*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceUnsupportedAgent))
		})

		It("[test_id:6223]should update guest agent for user password", func() {
			secretID := "my-user-pass"
			vmi := tests.NewRandomFedoraVMIWithBlacklistGuestAgent("guest-set-user-password")
			vmi.Namespace = util.NamespaceTestDefault

			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					UserPassword: &v1.UserPasswordAccessCredential{
						Source: v1.UserPasswordAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: secretID,
							},
						},
						PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
						},
					},
				},
			}

			customPassword := "imadethisup"

			By("Creating a secret with custom password")
			secret := kubev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretID,
					Namespace: vmi.Namespace,
					Labels: map[string]string{
						util.SecretLabel: secretID,
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"fedora": []byte(customPassword),
				},
			}
			_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that denylisted commands triggered unsupported guest agent condition")
			Eventually(matcher.ThisVMI(vmi), 240*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceUnsupportedAgent))
		})
	})
	Context("with secret and cloudInit propagation", func() {
		var vmi *v1.VirtualMachineInstance
		secretID := "my-pub-key"
		userData := fmt.Sprintf(
			"#cloud-config\npassword: %s\nchpasswd: { expire: False }\n",
			fedoraPassword,
		)

		key1 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"
		key2 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key2"
		key3 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key3"

		verifySSHKeys := func(vmi *v1.VirtualMachineInstance) {
			By("Verifying all three pub ssh keys in secret are in VMI guest")
			ExecutingBatchCmd(vmi, []expect.Batcher{
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
			}, time.Second*180)
		}

		BeforeEach(func() {
			vmi = tests.NewRandomVMIWithEphemeralDiskHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling))
			vmi.Namespace = util.NamespaceTestDefault

			By("Creating a secret with three ssh keys")
			secret := kubev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretID,
					Namespace: vmi.Namespace,
					Labels: map[string]string{
						util.SecretLabel: secretID,
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"my-key1": []byte(key1),
					"my-key2": []byte(key2),
					"my-key3": []byte(key3),
				},
			}
			_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:TODO]should have ssh-key under authorized keys added by NoCloud", func() {
			tests.AddCloudInitNoCloudData(vmi, "disk1", userData, "", false)
			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: secretID,
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			}
			tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
			verifySSHKeys(vmi)
		})
		It("[test_id:6224]should have ssh-key under authorized keys added by configDrive", func() {
			tests.AddCloudInitConfigDriveData(vmi, "disk1", userData, "", false)
			vmi.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: secretID,
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			}
			tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
			verifySSHKeys(vmi)
		})
	})
})
