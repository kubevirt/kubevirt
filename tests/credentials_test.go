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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = FDescribe("Guest Access Credentials", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var (
		LaunchVMI         func(*v1.VirtualMachineInstance) *v1.VirtualMachineInstance
		ExecutingBatchCmd func(*v1.VirtualMachineInstance, []expect.Batcher, time.Duration)
	)

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		LaunchVMI = func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
			Expect(err).To(BeNil())

			By("Waiting the VirtualMachineInstance start")
			vmi, ok := obj.(*v1.VirtualMachineInstance)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
			// Warnings are okay. We'll receive a warning that the agent isn't connected
			// during bootup, but that is transient
			Expect(tests.WaitForSuccessfulVMIStartIgnoreWarnings(obj)).ToNot(BeEmpty())
			return vmi
		}

		ExecutingBatchCmd = func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
			By("Checking that the VirtualMachineInstance serial console output equals to expected one")
			err := console.ExpectBatch(vmi, commands, timeout)
			Expect(err).ToNot(HaveOccurred())
		}

	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("with qemu guest agent", func() {
		It("should propagate public ssh keys", func() {
			secretID := "my-pub-key"
			vmi := tests.NewRandomFedoraVMIWithGuestAgent()
			vmi.Namespace = tests.NamespaceTestDefault
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
						tests.SecretLabel: secretID,
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
			Expect(err).To(BeNil())

			LaunchVMI(vmi)

			By("Waiting for agent to connect")
			tests.WaitAgentConnected(virtClient, vmi)

			By("Waiting on access credentials to sync")
			// this ensures the keys have propagated before we attempt to read
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, cond := range vmi.Status.Conditions {
					if cond.Type == v1.VirtualMachineInstanceAccessCredentialsSynchronized && cond.Status == kubev1.ConditionTrue {
						return true
					}
				}
				return false
			}, 45*time.Second, time.Second).Should(BeTrue())

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

		It("should propagate user password", func() {
			secretID := "my-user-pass"
			vmi := tests.NewRandomFedoraVMIWithGuestAgent()
			vmi.Namespace = tests.NamespaceTestDefault

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
						tests.SecretLabel: secretID,
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"fedora": []byte(customPassword),
				},
			}
			_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			LaunchVMI(vmi)

			By("Waiting for agent to connect")
			tests.WaitAgentConnected(virtClient, vmi)

			By("Verifying signin with custom password works")

			By("Waiting on access credentials to sync")
			// this ensures the keys have propagated before we attempt to read
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, cond := range vmi.Status.Conditions {
					if cond.Type == v1.VirtualMachineInstanceAccessCredentialsSynchronized && cond.Status == kubev1.ConditionTrue {
						return true
					}
				}
				return false
			}, 45*time.Second, time.Second).Should(BeTrue())

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
	})
	Context("with secret and configDrive propagation", func() {
		It("should have ssh-key under authorized keys", func() {
			secretID := "my-pub-key"
			userData := fmt.Sprintf(
				"#cloud-config\npassword: %s\nchpasswd: { expire: False }\n",
				fedoraPassword,
			)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedora), userData)
			vmi.Namespace = tests.NamespaceTestDefault
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

			key1 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key1"
			key2 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key2"
			key3 := "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key3"

			By("Creating a secret with three ssh keys")
			secret := kubev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretID,
					Namespace: vmi.Namespace,
					Labels: map[string]string{
						tests.SecretLabel: secretID,
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
			Expect(err).To(BeNil())

			LaunchVMI(vmi)

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
	})
})
