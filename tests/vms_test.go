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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = FDescribe("VirtualMachineSnapshot", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	newSnapshotWithVM := func(vm *v1.VirtualMachine) (*v1.VirtualMachineSnapshot, *v1.VirtualMachine) {
		By("Create a new VirtualMachineSnapshot")
		var currentVM *v1.VirtualMachine
		if vm == nil {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			currentVM = NewRandomVirtualMachine(vmi, false)

			currentVM, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(currentVM)
			Expect(err).ToNot(HaveOccurred())
		} else {
			currentVM = vm
		}

		newVMS := tests.NewRandomVirtualMachineSnapshot(currentVM)
		newVMS, err := virtClient.VirtualMachineSnapshot(tests.NamespaceTestDefault).Create(newVMS)
		Expect(err).ToNot(HaveOccurred())
		return newVMS, currentVM
	}

	newRestore := func(vm *v1.VirtualMachine, vms *v1.VirtualMachineSnapshot) *v1.VirtualMachineRestore {
		By("Create a new VirtualMachineRestore")
		if vm == nil || vms == nil{
			return nil
		}

		vmr := &v1.VirtualMachineRestore{
			ObjectMeta: v12.ObjectMeta{Name: vms.Spec.VirtualMachine, Namespace: vms.ObjectMeta.Namespace},
			Spec: v1.VirtualMachineRestoreSpec{
				VirtualMachineSnapshot: vms.Name,
			},
		}

		vmr, err := virtClient.VirtualMachineRestore(tests.NamespaceTestDefault).Create(vmr)
		Expect(err).ToNot(HaveOccurred())
		return vmr
	}

	Context("with valid VirtualMachine", func() {
		It("should snapshot VirtualMachine", func() {
			vms, vm := newSnapshotWithVM(nil)
			Eventually(func() types.UID {
				vms, _ = virtClient.VirtualMachineSnapshot(tests.NamespaceTestDefault).Get(vms.Name, v12.GetOptions{})
				if vms.Status.VirtualMachine != nil {
					return vms.Status.VirtualMachine.UID
				} else {
					return ""
				}
			}, 300*time.Second, 1*time.Second).Should(Equal(vm.UID))
		})

		It("should wait until VirtualMachine is not running", func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vm := NewRandomVirtualMachine(vmi, true)
			vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				currentVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return currentVM.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			vms, vm := newSnapshotWithVM(vm)
			vm.Spec.Running = false
			vm, err = virtClient.VirtualMachine(vm.Namespace).Update(vm)

			Eventually(func() types.UID {
				vms, _ = virtClient.VirtualMachineSnapshot(tests.NamespaceTestDefault).Get(vms.Name, v12.GetOptions{})
				if vms.Status.VirtualMachine != nil {
					return vms.Status.VirtualMachine.UID
				} else {
					return ""
				}
			}, 300*time.Second, 1*time.Second).Should(Equal(vm.UID))

		})

		It("should restore VirtualMachine", func() {
			vms, vm := newSnapshotWithVM(nil)
			Eventually(func() types.UID {
				vms, _ = virtClient.VirtualMachineSnapshot(tests.NamespaceTestDefault).Get(vms.Name, v12.GetOptions{})
				if vms.Status.VirtualMachine != nil {
					return vms.Status.VirtualMachine.UID
				} else {
					return ""
				}
			}, 300*time.Second, 1*time.Second).Should(Equal(vm.UID))

			updatedVM := vm.DeepCopy()
			updatedVM.Spec.Template.Spec.Hostname = "test"

			updatedVM, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Update(updatedVM)
			Expect(err).To(BeNil(), "should update VM")

			By("Restoring its state")
			vmr := newRestore(vm, vms)
			Eventually(func() *v12.Time {
				vmr, _ = virtClient.VirtualMachineRestore(tests.NamespaceTestDefault).Get(vmr.Name, v12.GetOptions{})
				return vmr.Status.RestoredOn
			}, 300*time.Second, 1*time.Second).ShouldNot(BeNil())

			updatedVM, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(updatedVM.Name, &v12.GetOptions{})
			Expect(err).To(BeNil())
			Expect(updatedVM.Spec.Template.Spec.Hostname).To(Equal(vm.Spec.Template.Spec.Hostname))
		})

		It("should restore VirtualMachine one it is running", func() {
			vms, vm := newSnapshotWithVM(nil)
			Eventually(func() types.UID {
				vms, _ = virtClient.VirtualMachineSnapshot(tests.NamespaceTestDefault).Get(vms.Name, v12.GetOptions{})
				if vms.Status.VirtualMachine != nil {
					return vms.Status.VirtualMachine.UID
				} else {
					return ""
				}
			}, 300*time.Second, 1*time.Second).Should(Equal(vm.UID))

			updatedVM := vm.DeepCopy()
			updatedVM.Spec.Template.Spec.Hostname = "test"
			updatedVM.Spec.Running = true

			updatedVM, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Update(updatedVM)
			Expect(err).To(BeNil(), "should update VM")

			By("Restoring its state")
			vmr := newRestore(vm, vms)
			Eventually(func() int {
				vmr, _ = virtClient.VirtualMachineRestore(tests.NamespaceTestDefault).Get(vmr.Name, v12.GetOptions{})
				return len(vmr.Status.Conditions)
			}, 300*time.Second, 1*time.Second).Should(Equal(1))

			Expect(vmr.Status.Conditions[0].Reason).To(Equal("VirtualMachineRunning"))
			updatedVM, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(updatedVM.Name, &v12.GetOptions{})
			Expect(err).To(BeNil())
			Expect(updatedVM.Spec.Template.Spec.Hostname).To(Equal("test"))

			updatedVM.Spec.Running = false
			updatedVM, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Update(updatedVM)
			Expect(err).To(BeNil(), "should update VM")

			By("Waiting until VM is not running")
			Eventually(func() *v12.Time {
				vmr, _ = virtClient.VirtualMachineRestore(tests.NamespaceTestDefault).Get(vmr.Name, v12.GetOptions{})
				return vmr.Status.RestoredOn
			}, 300*time.Second, 1*time.Second).ShouldNot(BeNil())

			updatedVM, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(updatedVM.Name, &v12.GetOptions{})
			Expect(err).To(BeNil())
			Expect(updatedVM.Spec.Template.Spec.Hostname).To(Equal(vm.Spec.Template.Spec.Hostname))
		})

		It("should not restore VirtualMachine without existing Snapshot", func() {
			vms, vm := newSnapshotWithVM(nil)
			Eventually(func() types.UID {
				vms, _ = virtClient.VirtualMachineSnapshot(tests.NamespaceTestDefault).Get(vms.Name, v12.GetOptions{})
				if vms.Status.VirtualMachine != nil {
					return vms.Status.VirtualMachine.UID
				} else {
					return ""
				}
			}, 300*time.Second, 1*time.Second).Should(Equal(vm.UID))

			virtClient.VirtualMachineSnapshot(tests.NamespaceTestDefault).Delete(vms.Name, &v12.DeleteOptions{})
			Expect(err).To(BeNil(), "should delete Snapshot")

			vmr := newRestore(vm, vms)
			Eventually(func() int {
				vmr, _ = virtClient.VirtualMachineRestore(tests.NamespaceTestDefault).Get(vmr.Name, v12.GetOptions{})
				return len(vmr.Status.Conditions)
			}, 300*time.Second, 1*time.Second).Should(Equal(1))
			By("Expecting it to fail due to missing snapshot")
			Expect(vmr.Status.Conditions[0].Reason).To(Equal("VirtualMachineHasNoSnapshot"), "should have snapshot")
		})

	})
})
