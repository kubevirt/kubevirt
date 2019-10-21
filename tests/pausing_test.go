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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package tests_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = FDescribe("[rfe_id:3064][crit:medium][vendor:cnv-qe@redhat.com][level:component]Pausing", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("A valid VMI", func() {

		var vmi *v1.VirtualMachineInstance

		runVMI := func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
			tests.RunVMIAndExpectLaunch(vmi, 90)
		}

		When("paused via API", func() {
			It("should signal paused state with condition", func() {
				runVMI()

				virtClient.VirtualMachineInstance(vmi.Namespace).Pause(vmi.Name)
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(vmi.Name)
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
			})
		})

		When("paused via virtctl", func() {
			It("[test_id:3079]should signal paused state with condition", func() {
				runVMI()
				command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", tests.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
			})

			It("[test_id:3080]should signal unpaused state with removed condition", func() {
				runVMI()
				command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", tests.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vmi", "--namespace", tests.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
			})
		})
	})

	Context("A valid VM", func() {

		var vm *v1.VirtualMachine

		runVM := func() {
			vm = tests.NewRandomVMWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			vm = tests.StartVirtualMachine(vm)
		}

		When("paused via API", func() {
			It("should signal paused state with condition", func() {

				runVM()

				virtClient.VirtualMachineInstance(vm.Namespace).Pause(vm.Name)
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				virtClient.VirtualMachineInstance(vm.Namespace).Unpause(vm.Name)
				tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)
			})

		})

		When("paused via virtctl", func() {

			It("[test_id:3059]should signal paused state with condition", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)
			})

			It("[test_id:3081]should gracefully handle pausing the VM again", func() {
				// TODO does not work yet
				//runVM()
				//command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
				//tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				//command = tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
			})

			It("[test_id:3088]should gracefully handle pausing the VMI again", func() {
				// TODO does not work yet
				//runVM()
				//command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
				//tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				//command = tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
			})

			It("[test_id:3060]should signal unpaused state with removed condition", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)
			})

			It("[test_id:3082]should gracefully handle unpausing again", func() {
				// TODO does not work yet
				//runVM()
				//command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
				//tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)
				//
				//command = tests.NewRepeatableVirtctlCommand("unpause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
				//tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)

				//command := tests.NewRepeatableVirtctlCommand("unpause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
			})
		})

		When("paused via virtctl", func() {
			It("[test_id:3085]should be stopped successfully", func() {

				runVM()

				By("pausing the VM")
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("stopping the VM")
				command = tests.NewRepeatableVirtctlCommand("stop", "--namespace", tests.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("checking deletion of VMI")
				Eventually(func() bool {
					_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
					if errors.IsNotFound(err) {
						return true
					}
					return false
				}, 300*time.Second, 1*time.Second).Should(BeTrue(), "The VMI did not disappear")

				By("checking status of VM")
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Ready
				}, 300*time.Second, 1*time.Second).Should(BeFalse())

			})
		})

		When("paused via virtctl", func() {
			It("[test_id:3086]should not be migrated", func() {

				// TODO does not work yet
				//runVM()
				//
				//By("pausing the VM")
				//command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//Expect(command()).To(Succeed())
				//tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)
				//
				//By("trying to migrate the VM")
				//command = tests.NewRepeatableVirtctlCommand("migrate", "--namespace", tests.NamespaceTestDefault, vm.Name)
				//// TODO check error message
				//Expect(command()).ToNot(Succeed())

			})
		})
	})
})
