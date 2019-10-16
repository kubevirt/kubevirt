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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:xxx][crit:medium][vendor:cnv-qe@redhat.com][level:component]SuspendResume", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("A valid VMI", func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
			tests.RunVMIAndExpectLaunch(vmi, 90)
		})

		AfterEach(func() {

		})

		It("Should report paused status on VMI", func() {
			By("Suspending VMI")
			virtClient.VirtualMachineInstance(vmi.Namespace).Suspend(vmi.Name)
			tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

			By("Resuming VMI")
			virtClient.VirtualMachineInstance(vmi.Namespace).Resume(vmi.Name)
			tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
		})

	})

	Context("A valid VM", func() {

		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vm = tests.NewRandomVMWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			vm = tests.StartVirtualMachine(vm)
		})

		AfterEach(func() {

		})

		It("Should report paused status on VM", func() {
			By("Suspending VMI")
			virtClient.VirtualMachineInstance(vm.Namespace).Suspend(vm.Name)
			tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

			By("Resuming VMI")
			virtClient.VirtualMachineInstance(vm.Namespace).Resume(vm.Name)
			tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)
		})

	})
})
