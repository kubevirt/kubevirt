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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("DataVolume Integration", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		if !tests.HasCDI() {
			Skip("Skip DataVolume tests when CDI is not present")
		}

		// TODO remove this once local storage provider is used
		//
		// In order to be able to test CDI with hostPath, we have to Temporarily
		// ensure just for these DataVolume tests that only a single node is scheduable.
		// This is the only way to guarantee both the import pod and VM pod land on the same node.
		tests.TaintAllButOne()
	})

	AfterEach(func() {
		tests.RemoveAllTaints()
	})

	runVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		By("Starting a VirtualMachineInstance with DataVolume")

		var obj *v1.VirtualMachineInstance
		var err error
		Eventually(func() error {
			obj, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the VirtualMachineInstance will start")
		tests.WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
		return obj
	}

	Describe("Starting a VirtualMachineInstance with a DataVolume", func() {
		Context("using Alpine import", func() {
			It("should be successfully started and stopped multiple times", func() {
				vmi := tests.NewRandomVMIWithDataVolume(tests.AlpineHttpUrl)

				num := 2
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					vmi := runVMIAndExpectLaunch(vmi, 120)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						expecter, err := tests.LoggedInAlpineExpecter(vmi)
						Expect(err).To(BeNil())
						expecter.Close()
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			})
		})
	})

	Describe("Starting a VirtualMachine with a DataVolume", func() {
		Context("using Alpine import", func() {
			It("should be successfully started and stopped multiple times", func() {
				vm := tests.NewRandomVMWithDataVolume(tests.AlpineHttpUrl)

				vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				num := 2
				By("Starting and stopping the VirtualMachine number of times")

				for i := 0; i < num; i++ {
					By(fmt.Sprintf("Doing run: %d", i))
					vm = tests.StartVirtualMachine(vm)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						expecter, err := tests.LoggedInAlpineExpecter(vmi)
						Expect(err).To(BeNil())
						expecter.Close()
					}

					vm = tests.StopVirtualMachine(vm)
				}
			})
		})

	})
})
