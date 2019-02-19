/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/tests"
)

const (
	sshAuthorizedKey = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key"
	fedoraPassword   = "fedora"
	expectedUserData = "printed from cloud-init userdata"
	testNetworkData  = "#Test networkData"
	testUserData     = "#cloud-config"
)

var _ = Describe("[rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component]CloudInit UserData", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	LaunchVMI := func(vmi *v1.VirtualMachineInstance) {
		By("Starting a VirtualMachineInstance")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
		Expect(err).To(BeNil())

		By("Waiting the VirtualMachineInstance start")
		_, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		Expect(tests.WaitForSuccessfulVMIStart(obj)).ToNot(BeEmpty())
	}

	VerifyUserDataVMI := func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
		By("Expecting the VirtualMachineInstance console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 10*time.Second)
		Expect(err).ToNot(HaveOccurred())
		defer expecter.Close()

		By("Checking that the VirtualMachineInstance serial console output equals to expected one")
		resp, err := expecter.ExpectBatch(commands, timeout)
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
		Expect(err).ToNot(HaveOccurred())
	}

	MountCloudInit := func(vmi *v1.VirtualMachineInstance, prompt string) {
		cmdCheck := "mount $(blkid  -L cidata) /mnt/\n"
		err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
			&expect.BSnd{S: "sudo su -\n"},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: cmdCheck},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: "0"},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	CheckCloudInitFile := func(vmi *v1.VirtualMachineInstance, prompt, testFile, testData string) {
		cmdCheck := "cat /mnt/" + testFile + "\n"
		err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
			&expect.BSnd{S: "sudo su -\n"},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: cmdCheck},
			&expect.BExp{R: testData},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("[rfe_id:151][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with cloudInitNoCloud userDataBase64 source", func() {
			It("[test_id:1615]should have cloud-init data", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), userData)
				LaunchVMI(vmi)
				VerifyUserDataVMI(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)
			})

			Context("with injected ssh-key", func() {
				It("[test_id:1616]should have ssh-key under authorized keys", func() {
					userData := fmt.Sprintf(
						"#cloud-config\npassword: %s\nchpasswd: { expire: False }\nssh_authorized_keys:\n  - %s",
						fedoraPassword,
						sshAuthorizedKey,
					)
					vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(tests.ContainerDiskFor(tests.ContainerDiskFedora), userData)

					LaunchVMI(vmi)

					VerifyUserDataVMI(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: "login:"},
						&expect.BSnd{S: "fedora\n"},
						&expect.BExp{R: "Password:"},
						&expect.BSnd{S: fedoraPassword + "\n"},
						&expect.BExp{R: "$"},
						&expect.BSnd{S: "cat /home/fedora/.ssh/authorized_keys\n"},
						&expect.BExp{R: "test-ssh-key"},
					}, time.Second*300)
				})
			})
		})

		Context("with cloudInitNoCloud userData source", func() {
			It("[test_id:1617]should process provided cloud-init data", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)

				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "disk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "disk1",
					VolumeSource: v1.VolumeSource{
						CloudInitNoCloud: &v1.CloudInitNoCloudSource{
							UserData: userData,
						},
					},
				})

				LaunchVMI(vmi)

				By("executing a user-data script")
				VerifyUserDataVMI(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)

				By("applying the hostname from meta-data")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "hostname\n"},
					&expect.BExp{R: dns.SanitizeHostname(vmi)},
				}, time.Second*10)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("[test_id:1618]should take user-data from k8s secret", func() {
			userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "")

			idx := 0
			for i, volume := range vmi.Spec.Volumes {
				if volume.CloudInitNoCloud == nil {
					continue
				}
				idx = i

				secretID := fmt.Sprintf("%s-test-secret", uuid.NewRandom().String())
				spec := volume.CloudInitNoCloud
				spec.UserDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}

				// Store userdata as k8s secret
				By("Creating a user-data secret")
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
						"userdata": []byte(userData), // The client encrypts the secret for us
					},
				}
				_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(&secret)
				Expect(err).To(BeNil())
				break
			}
			LaunchVMI(vmi)
			VerifyUserDataVMI(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: expectedUserData},
			}, time.Second*120)

			// Expect that the secret is not present on the vmi itself
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserData).To(BeEmpty())
			Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserDataBase64).To(BeEmpty())
		})

		Context("with cloudInitNoCloud networkData", func() {
			It("should have cloud-init network-config with NetworkData source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(
					tests.ContainerDiskFor(tests.ContainerDiskCirros), testUserData, testNetworkData, false)
				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

				By("mouting cloudinit iso")
				MountCloudInit(vmi, "#")

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "#", "network-config", testNetworkData)

				By("checking cloudinit user-data")
				CheckCloudInitFile(vmi, "#", "user-data", testUserData)
			})
			It("should have cloud-init network-config with NetworkDataBase64 source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(
					tests.ContainerDiskFor(tests.ContainerDiskCirros), testUserData, testNetworkData, true)
				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

				By("mouting cloudinit iso")
				MountCloudInit(vmi, "#")

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "#", "network-config", testNetworkData)

				By("checking cloudinit user-data")
				CheckCloudInitFile(vmi, "#", "user-data", testUserData)
			})
			It("should have cloud-init network-config from k8s secret", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(
					tests.ContainerDiskFor(tests.ContainerDiskCirros), "", "", false)

				idx := 0
				for i, volume := range vmi.Spec.Volumes {
					if volume.CloudInitNoCloud == nil {
						continue
					}
					idx = i

					secretID := fmt.Sprintf("%s-test-secret", uuid.NewRandom().String())
					spec := volume.CloudInitNoCloud
					spec.UserDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}
					spec.NetworkDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}

					// Store cloudinit data as k8s secret
					By("Creating a secret with user and network data")
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
							// The client encrypts the secret for us
							"userdata":    []byte(testUserData),
							"networkdata": []byte(testNetworkData),
						},
					}
					_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(&secret)
					Expect(err).To(BeNil())

					break
				}

				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

				By("mouting cloudinit iso")
				MountCloudInit(vmi, "#")

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "#", "network-config", testNetworkData)

				By("checking cloudinit user-data")
				CheckCloudInitFile(vmi, "#", "user-data", testUserData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserDataBase64).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.NetworkData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.NetworkDataBase64).To(BeEmpty())
			})

		})

	})
})
