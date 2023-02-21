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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"

	"kubevirt.io/kubevirt/tests/libstorage"

	"kubevirt.io/kubevirt/tests/util"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/framework/checks"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
)

var _ = Describe("[sig-compute]vTPM", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("[rfe_id:5168][crit:high][vendor:cnv-qe@redhat.com][level:component] with TPM VMI option enabled", func() {
		It("[test_id:8607] should expose a functional emulated TPM which persists across migrations", func() {
			By("Creating a VMI with TPM enabled")
			vmi := tests.NewRandomFedoraVMI()
			vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{}
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

			By("Logging in as root")
			err = console.LoginToFedora(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring a TPM device is present")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /dev/tpm*\n"},
				&expect.BExp{R: "/dev/tpm0"},
			}, 300)).To(Succeed(), "Could not find a TPM device")

			By("Ensuring the TPM device is functional")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_pcrread sha256:15\n"},
				&expect.BExp{R: "0x0000000000000000000000000000000000000000000000000000000000000000"},
				&expect.BSnd{S: "tpm2_pcrextend 15:sha256=54d626e08c1c802b305dad30b7e54a82f102390cc92c7d4db112048935236e9c && echo 'do''ne'\n"},
				&expect.BExp{R: "done"},
				&expect.BSnd{S: "tpm2_pcrread sha256:15\n"},
				&expect.BExp{R: "0x1EE66777C372B96BC74AC4CB892E0879FA3CCF6A2F53DB1D00FD18B264797F49"},
			}, 300)).To(Succeed(), "PCR extension doesn't work correctly")

			By("Migrating the VMI")
			checks.SkipIfMigrationIsNotPossible()
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			By("Ensuring the TPM is still functional and its state carried over")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_pcrread sha256:15\n"},
				&expect.BExp{R: "0x1EE66777C372B96BC74AC4CB892E0879FA3CCF6A2F53DB1D00FD18B264797F49"},
			}, 300)).To(Succeed(), "Migrating broke the TPM")
		})
	})
})

var _ = Describe("[sig-storage]vTPM", decorators.SigStorage, func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("with persistent TPM VM option enabled", func() {
		restartVM := func(vm *v1.VirtualMachine) {
			By("Stopping the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			EventuallyWithOffset(1, func() error {
				_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(Succeed())

			By("Starting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			var vmi *v1.VirtualMachineInstance
			EventuallyWithOffset(1, func() error {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())
			libwait.WaitForSuccessfulVMIStartWithTimeout(vmi, 60)

			By("Logging in as root")
			err = console.LoginToFedora(vmi)
			Expect(err).ToNot(HaveOccurred())
		}

		migrateVMI := func(vmi *v1.VirtualMachineInstance) {
			By("Migrating the VMI")
			checks.SkipIfMigrationIsNotPossible()
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

		}

		checkTPM := func(vmi *v1.VirtualMachineInstance) {
			By("Ensuring the TPM is still functional and its state carried over")
			ExpectWithOffset(1, console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_unseal -Q --object-context=0x81010002\n"},
				&expect.BExp{R: "MYSECRET"},
			}, 300)).To(Succeed(), "the state of the TPM did not persist")
		}

		DescribeTable("[Serial]should persist TPM secrets across", Serial, func(ops ...string) {
			By("Setting the backend storage class to the default for RWX FS")
			storageClass, exists := libstorage.GetRWXFileSystemStorageClass()
			Expect(exists).To(BeTrue(), "No RWX FS storage class found")
			kv := util.GetCurrentKv(virtClient)
			kv.Spec.Configuration.VMStateStorageClass = storageClass
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

			By("Creating a VM with persistent TPM enabled")
			vmi := tests.NewRandomFedoraVMI()
			vmi.Namespace = util.NamespaceTestDefault
			vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
				Persistent: pointer.BoolPtr(true),
			}
			vm := tests.NewRandomVirtualMachine(vmi, true)
			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VM to start")
			Eventually(func() error {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())
			libwait.WaitForSuccessfulVMIStartWithTimeout(vmi, 60)

			By("Logging in as root")
			err = console.LoginToFedora(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Storing a secret into the TPM")
			// https://www.intel.com/content/www/us/en/developer/articles/code-sample/protecting-secret-data-and-keys-using-intel-platform-trust-technology.html
			// Not sealing against a set of PCRs, out of scope here, but should work with a carefully selected set (at least PCR1 was seen changing accross reboots)
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_createprimary -Q --hierarchy=o --key-context=prim.ctx\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo MYSECRET | tpm2_create --hash-algorithm=sha256 --public=seal.pub --private=seal.priv --sealing-input=- --parent-context=prim.ctx\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "tpm2_load -Q --parent-context=prim.ctx --public=seal.pub --private=seal.priv --name=seal.name --key-context=seal.ctx\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "tpm2_evictcontrol --hierarchy=o --object-context=seal.ctx 0x81010002\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "tpm2_unseal -Q --object-context=0x81010002\n"},
				&expect.BExp{R: "MYSECRET"},
			}, 300)).To(Succeed(), "failed to store secret into the TPM")

			for _, op := range ops {
				switch op {
				case "migrate":
					migrateVMI(vmi)
				case "restart":
					restartVM(vm)
				}
				checkTPM(vmi)
			}

			By("Stopping and removing the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = virtClient.VirtualMachine(util.NamespaceTestDefault).Delete(context.Background(), vm.Name, &k8smetav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("migration and restart", "migrate", "restart"),
			Entry("restart and migration", "restart", "migrate"),
		)
		It("[Serial]should remove persistent storage PVC if VMI is not owned by a VM", Serial, func() {
			By("Setting the backend storage class to the default for RWX FS")
			storageClass, exists := libstorage.GetRWXFileSystemStorageClass()
			Expect(exists).To(BeTrue(), "No RWX FS storage class found")
			kv := util.GetCurrentKv(virtClient)
			kv.Spec.Configuration.VMStateStorageClass = storageClass
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

			By("Creating a VMI with persistent TPM enabled")
			vmi := tests.NewRandomFedoraVMI()
			vmi.Namespace = util.NamespaceTestDefault
			vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
				Persistent: pointer.BoolPtr(true),
			}
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VMI to start")
			Eventually(func() error {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())
			libwait.WaitForSuccessfulVMIStartWithTimeout(vmi, 60)

			By("Removing the VMI")
			err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.Name, &k8smetav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the PVC gets deleted")
			Eventually(func() error {
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
				if !errors.IsNotFound(err) {
					return fmt.Errorf("VM %s not removed: %v", vmi.Name, err)
				}
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), backendstorage.PVCForVMI(vmi), k8smetav1.GetOptions{})
				if !errors.IsNotFound(err) {
					return fmt.Errorf("PVC %s not removed: %v", backendstorage.PVCForVMI(vmi), err)
				}
				return nil
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		})
	})
})
