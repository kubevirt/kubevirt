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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/goexpect"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("OfflineVirtualMachine", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("A valid OfflineVirtualMachine given", func() {

		newOfflineVirtualMachine := func(running bool) *v1.OfflineVirtualMachine {
			vmImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			template := tests.NewRandomVMWithEphemeralDiskAndUserdata(vmImage, "echo Hi\n")
			newOVM := NewRandomOfflineVirtualMachine(template, running)
			newOVM, err = virtClient.OfflineVirtualMachine(tests.NamespaceTestDefault).Create(newOVM)
			Expect(err).ToNot(HaveOccurred())
			return newOVM
		}

		startOVM := func(ovm *v1.OfflineVirtualMachine) *v1.OfflineVirtualMachine {
			By("Starting the VM")
			ovm, err = virtClient.OfflineVirtualMachine(ovm.Namespace).Get(ovm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedOVM := ovm.DeepCopy()
			updatedOVM.Spec.Running = true
			updatedOVM, err = virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Update(updatedOVM)
			Expect(err).ToNot(HaveOccurred())

			// Observe the VM created
			Eventually(func() error {
				_, err := virtClient.VM(updatedOVM.Namespace).Get(updatedOVM.Name, v12.GetOptions{})
				return err
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			By("OVM has the running condition")
			Eventually(func() bool {
				updatedOVM, err = virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Get(updatedOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return hasCondition(updatedOVM, v1.OfflineVirtualMachineRunning)
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			return updatedOVM
		}

		stopOVM := func(ovm *v1.OfflineVirtualMachine) *v1.OfflineVirtualMachine {
			By("Stopping the VM")
			ovm, err = virtClient.OfflineVirtualMachine(ovm.Namespace).Get(ovm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedOVM := ovm.DeepCopy()
			updatedOVM.Spec.Running = false
			updatedOVM, err := virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Update(updatedOVM)
			Expect(err).ToNot(HaveOccurred())

			// Observe the VM deleted
			Eventually(func() bool {
				_, err := virtClient.VM(updatedOVM.Namespace).Get(updatedOVM.Name, v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			By("OVM has not the running condition")
			Eventually(func() bool {
				updatedOVM, err = virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Get(updatedOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return hasCondition(updatedOVM, v1.OfflineVirtualMachineRunning)
			}, 120*time.Second, 1*time.Second).Should(BeFalse())

			return updatedOVM
		}

		It("should update OfflineVirtualMachine once VMs are up", func() {
			newOVM := newOfflineVirtualMachine(true)
			Eventually(func() v1.OfflineVirtualMachineConditionType {
				ovm, err := virtClient.OfflineVirtualMachine(tests.NamespaceTestDefault).Get(newOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return ovm.Status.Conditions[0].Type
			}, 120*time.Second, 1*time.Second).Should(Equal(v1.OfflineVirtualMachineRunning))
		})

		It("should remove VM once the OVM is marked for deletion", func() {
			newOVM := newOfflineVirtualMachine(true)
			// Create a offlinevm with vm
			// Delete it
			Expect(virtClient.OfflineVirtualMachine(newOVM.Namespace).Delete(newOVM.Name, &v12.DeleteOptions{})).To(Succeed())
			// Wait until VMs are gone
			Eventually(func() int {
				vms, err := virtClient.VM(newOVM.Namespace).List(v12.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(vms.Items)
			}, 120*time.Second, 2*time.Second).Should(BeZero())
		})

		It("should remove owner references on the VM if it is orphan deleted", func() {
			newOVM := newOfflineVirtualMachine(true)

			Eventually(func() []v12.OwnerReference {
				// Check for owner reference
				vm, _ := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
				return vm.OwnerReferences
			}, 120*time.Second, 1*time.Second).ShouldNot(BeEmpty())

			// Delete it
			orphanPolicy := v12.DeletePropagationOrphan
			Expect(virtClient.OfflineVirtualMachine(newOVM.Namespace).
				Delete(newOVM.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
			// Wait until the offlinevm is deleted
			Eventually(func() bool {
				_, err := virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			vm, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.OwnerReferences).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("shloud recreate VM if it gets deleted", func() {
			newOVM := newOfflineVirtualMachine(true)
			// Delete the VM
			Eventually(func() error {
				return virtClient.VM(newOVM.Namespace).Delete(newOVM.Name, &v12.DeleteOptions{})
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			Eventually(func() bool {
				_, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
				if errors.IsNotFound(err) {
					return false
				}
				return true
			}, 120*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("should stop VM if running set to false", func() {
			var currOVM *v1.OfflineVirtualMachine

			currOVM = newOfflineVirtualMachine(false)

			currOVM = startOVM(currOVM)
			currOVM = stopOVM(currOVM)

		})

		It("should start and stop VM multiple times", func() {
			var currOVM *v1.OfflineVirtualMachine

			currOVM = newOfflineVirtualMachine(false)

			// Start and stop VM multiple times
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				startOVM(currOVM)
				stopOVM(currOVM)
			}
		})

		It("should not update the VM spec if Running", func() {
			newOVM := newOfflineVirtualMachine(true)

			Eventually(func() bool {
				newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return hasCondition(newOVM, v1.OfflineVirtualMachineRunning)
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			By("Updating the OVM template spec")
			newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedOVM := newOVM.DeepCopy()
			updatedOVM.Spec.Template.Spec.Domain.Resources.Requests = v13.ResourceList{
				v13.ResourceMemory: resource.MustParse("4096Ki"),
			}
			updatedOVM, err := virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Update(updatedOVM)
			Expect(err).ToNot(HaveOccurred())

			By("Expecting the old VM spec still running")
			vm, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmMemory := vm.Spec.Domain.Resources.Requests.Memory()
			ovmMemory := newOVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmMemory.Cmp(*ovmMemory)).To(Equal(0))

			By("Restarting the OVM")
			stopOVM(newOVM)
			startOVM(newOVM)

			By("Expecting updated spec running")
			vm, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmMemory = vm.Spec.Domain.Resources.Requests.Memory()
			ovmMemory = updatedOVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmMemory.Cmp(*ovmMemory)).To(Equal(0))
		})

		It("should survive guest shutdown, multiple times", func() {
			newOVM := newOfflineVirtualMachine(false)

			for i := 0; i < 1; i++ {
				startOVM(newOVM)

				By("VM being in running phase")
				Eventually(func() bool {
					vm, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase == "Running"
				}, 200*time.Second, 1*time.Second).Should(BeTrue())

				vm, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
				expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Guest reboot")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "login as 'cirros' user. default password: 'gocubsgo'. use 'sudo' for root."},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "cirros login:"},
					&expect.BSnd{S: "cirros\n"},
					&expect.BExp{R: "Password:"},
					&expect.BSnd{S: "gocubsgo\n"},
					&expect.BExp{R: "$"},
					// keep the ordering!
					&expect.BSnd{S: "sudo poweroff"},
					&expect.BExp{R: "reboot: Power down"},
				}, 360*time.Second)
				Expect(err).ToNot(HaveOccurred())

				By("OVM does not have running condition")
				Eventually(func() bool {
					newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return hasCondition(newOVM, v1.OfflineVirtualMachineFailure)
				}, 120*time.Second, 1*time.Second).Should(BeTrue())
			}
		})
	})
})

// NewRandomOfflineVirtualMachine creates new OfflineVirtualMachine
func NewRandomOfflineVirtualMachine(vm *v1.VirtualMachine, running bool) *v1.OfflineVirtualMachine {
	name := vm.Name
	ovm := &v1.OfflineVirtualMachine{
		ObjectMeta: v12.ObjectMeta{Name: name},
		Spec: v1.OfflineVirtualMachineSpec{
			Running: running,
			Template: &v1.VMTemplateSpec{
				ObjectMeta: v12.ObjectMeta{
					Labels: map[string]string{"name": name},
					Name:   vm.ObjectMeta.Name,
				},
				Spec: vm.Spec,
			},
		},
	}
	return ovm
}

func hasCondition(ovm *v1.OfflineVirtualMachine, cond v1.OfflineVirtualMachineConditionType) bool {
	for _, c := range ovm.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}

	return false
}
