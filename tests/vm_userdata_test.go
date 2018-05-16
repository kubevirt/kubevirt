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

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

const (
	sshAuthorizedKey = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key"
	fedoraPassword   = "fedora"
	expectedUserData = "printed from cloud-init userdata"
)

var _ = Describe("CloudInit UserData", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	LaunchVM := func(vm *v1.VirtualMachine) {
		By("Starting a VM")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())

		By("Waiting the VM start")
		_, ok := obj.(*v1.VirtualMachine)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
		Expect(tests.WaitForSuccessfulVMStart(obj)).ToNot(BeEmpty())
	}

	VerifyUserDataVM := func(vm *v1.VirtualMachine, commands []expect.Batcher, timeout time.Duration) {
		By("Expecting the VM console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
		defer expecter.Close()
		Expect(err).ToNot(HaveOccurred())

		By("Checking that the VM serial console output equals to expected one")
		resp, err := expecter.ExpectBatch(commands, timeout)
		log.DefaultLogger().Object(vm).Infof("%v", resp)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("A new VM", func() {
		Context("with cloudInitNoCloud userDataBase64 source", func() {
			It("should have cloud-init data", func(done Done) {
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)

				vm := tests.NewRandomVMWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), userData)
				LaunchVM(vm)
				VerifyUserDataVM(vm, []expect.Batcher{
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)
				close(done)
			}, 180)

			Context("with injected ssh-key", func() {
				It("should have ssh-key under authorized keys", func() {
					userData := fmt.Sprintf(
						"#cloud-config\npassword: %s\nchpasswd: { expire: False }\nssh_authorized_keys:\n  - %s",
						fedoraPassword,
						sshAuthorizedKey,
					)
					vm := tests.NewRandomVMWithEphemeralDiskAndUserdataHighMemory(tests.RegistryDiskFor(tests.RegistryDiskFedora), userData)

					LaunchVM(vm)

					VerifyUserDataVM(vm, []expect.Batcher{
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
			It("should process provided cloud-init data", func(done Done) {
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)

				vm := tests.NewRandomVMWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskCirros))
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       "disk1",
					VolumeName: "disk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
					Name: "disk1",
					VolumeSource: v1.VolumeSource{
						CloudInitNoCloud: &v1.CloudInitNoCloudSource{
							UserData: userData,
						},
					},
				})

				LaunchVM(vm)

				By("executing a user-data script")
				VerifyUserDataVM(vm, []expect.Batcher{
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)

				By("applying the hostname from meta-data")
				expecter, err := tests.LoggedInCirrosExpecter(vm)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "hostname\n"},
					&expect.BExp{R: vm.Name},
				}, time.Second*10)
				log.DefaultLogger().Object(vm).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
				close(done)
			}, 180)
		})

		It("should take user-data from k8s secret", func(done Done) {
			userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)
			vm := tests.NewRandomVMWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "")

			idx := 0
			for i, volume := range vm.Spec.Volumes {
				if volume.CloudInitNoCloud == nil {
					continue
				}
				idx = i

				secretID := fmt.Sprintf("%s-test-secret", vm.Name)
				spec := volume.CloudInitNoCloud
				spec.UserDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}

				// Store userdata as k8s secret
				By("Creating a user-data secret")
				secret := kubev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretID,
						Namespace: vm.GetObjectMeta().GetNamespace(),
					},
					Type: "Opaque",
					Data: map[string][]byte{
						"userdata": []byte(userData), // The client encrypts the secret for us
					},
				}
				_, err := virtClient.CoreV1().Secrets(vm.GetObjectMeta().GetNamespace()).Create(&secret)
				Expect(err).To(BeNil())
				break
			}
			LaunchVM(vm)
			VerifyUserDataVM(vm, []expect.Batcher{
				&expect.BExp{R: expectedUserData},
			}, time.Second*120)

			// Expect that the secret is not present on the vm itself
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Get(vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Volumes[idx].CloudInitNoCloud.UserData).To(BeEmpty())
			Expect(vm.Spec.Volumes[idx].CloudInitNoCloud.UserDataBase64).To(BeEmpty())

			close(done)
		}, 180)
	})
})
