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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "github.com/onsi/ginkgo/extensions/table"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"

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

		newOfflineVirtualMachine := func() *v1.OfflineVirtualMachine {
			template := tests.NewRandomVMWithEphemeralDisk("kubevirt/cirros-registry-disk-demo:devel")
			newOVM := NewRandomOfflineVirtualMachine(template, true)
			newOVM, err = virtClient.OfflineVirtualMachine(tests.NamespaceTestDefault).Create(newOVM)
			Expect(err).ToNot(HaveOccurred())
			return newOVM
		}

		It("should update readyReplicas once VMs are up", func() {
			newOVM := newOfflineVirtualMachine()
			Eventually(func() v1.OfflineVirtualMachineConditionType {
				ovm, err := virtClient.OfflineVirtualMachine(tests.NamespaceTestDefault).Get(newOVM.ObjectMeta.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return ovm.Status.Conditions[0].Type
			}, 120*time.Second, 1*time.Second).Should(Equal("OVMRunning"))
		})

		It("should remove VM once the OVM is marked for deletion", func() {
			newOVM := newOfflineVirtualMachine()
			// Create a offlinevm with vm
			// Delete it
			Expect(virtClient.OfflineVirtualMachine(newOVM.ObjectMeta.Namespace).Delete(newOVM.ObjectMeta.Name, &v12.DeleteOptions{})).To(Succeed())
			// Wait until VMs are gone
			Eventually(func() int {
				vms, err := virtClient.VM(newOVM.ObjectMeta.Namespace).List(v12.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(vms.Items)
			}, 60*time.Second, 1*time.Second).Should(BeZero())
		})

		It("should remove owner references on the VM if it is orphan deleted", func() {
			newOVM := newOfflineVirtualMachine()

			// Check for owner reference
			vms, err := virtClient.VM(newOVM.ObjectMeta.Namespace).List(v12.ListOptions{})
			Expect(vms.Items).To(HaveLen(1))
			for _, vm := range vms.Items {
				Expect(vm.OwnerReferences).ToNot(BeEmpty())
			}

			// Delete it
			orphanPolicy := v12.DeletePropagationOrphan
			Expect(virtClient.OfflineVirtualMachine(newOVM.ObjectMeta.Namespace).
				Delete(newOVM.ObjectMeta.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
			// Wait until the replica set is deleted
			Eventually(func() bool {
				_, err := virtClient.OfflineVirtualMachine(newOVM.ObjectMeta.Namespace).Get(newOVM.ObjectMeta.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			vms, err = virtClient.VM(newOVM.ObjectMeta.Namespace).List(v12.ListOptions{})
			Expect(vms.Items).To(HaveLen(2))
			for _, vm := range vms.Items {
				Expect(vm.OwnerReferences).To(BeEmpty())
			}
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

// NewRandomOfflineVirtualMachine creates new OfflineVirtualMachine
func NewRandomOfflineVirtualMachine(vm *v1.VirtualMachine, running bool) *v1.OfflineVirtualMachine {
	name := "offlinevm" + rand.String(5)
	ovm := &v1.OfflineVirtualMachine{
		ObjectMeta: v12.ObjectMeta{Name: "offlinevm" + rand.String(5)},
		Spec: v1.OfflineVirtualMachineSpec{
			Selector: &v12.LabelSelector{
				MatchLabels: map[string]string{"name": name},
			},
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
