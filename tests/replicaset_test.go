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
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/ginkgo/extensions/table"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"

	"time"

	"fmt"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VirtualMachineReplicaSet", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	doScale := func(name string, scale int32) {

		// Status updates can conflict with our desire to change the spec
		By(fmt.Sprintf("Scaling to %d", scale))
		var rs *v1.VirtualMachineReplicaSet
		for {
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			rs.Spec.Replicas = &scale
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Update(rs)
			if errors.IsConflict(err) {
				continue
			}
			break
		}

		Expect(err).ToNot(HaveOccurred())

		By("Checking the number of replicas")
		Eventually(func() int32 {
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rs.Status.Replicas
		}, 10, 1).Should(Equal(int32(scale)))

		vms, err := virtClient.VM(tests.NamespaceTestDefault).List(v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tests.NotDeleted(vms)).To(HaveLen(int(scale)))
	}

	newReplicaSet := func() *v1.VirtualMachineReplicaSet {
		By("Create a new VM replica set")
		template := tests.NewRandomVMWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskCirros))
		newRS := tests.NewRandomReplicaSetFromVM(template, int32(0))
		newRS, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Create(newRS)
		Expect(err).ToNot(HaveOccurred())
		return newRS
	}

	table.DescribeTable("should scale", func(startScale int, stopScale int) {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, int32(startScale))
		doScale(newRS.ObjectMeta.Name, int32(stopScale))
		doScale(newRS.ObjectMeta.Name, int32(0))

	},
		table.Entry("to three, to two and then to zero replicas", 3, 2),
		table.Entry("to five, to six and then to zero replicas", 5, 6),
	)

	It("should be rejected on POST if spec is invalid", func() {
		newRS := newReplicaSet()
		newRS.TypeMeta = v12.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineReplicaSet",
		}

		jsonBytes, err := json.Marshal(newRS)
		Expect(err).To(BeNil())

		// change the name of a required field (like domain) so validation will fail
		jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

		result := virtClient.RestClient().Post().Resource("virtualmachinereplicasets").Namespace(tests.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do()

		// Verify validation failed.
		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

	})
	It("should update readyReplicas once VMs are up", func() {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, 2)

		By("Checking the number of ready replicas")
		Eventually(func() int {
			rs, err := virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(newRS.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return int(rs.Status.ReadyReplicas)
		}, 120*time.Second, 1*time.Second).Should(Equal(2))
	})

	It("should remove VMs once it is marked for deletion", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)
		// Delete it
		By("Deleting the VM replica set")
		Expect(virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).Delete(newRS.ObjectMeta.Name, &v12.DeleteOptions{})).To(Succeed())
		// Wait until VMs are gone
		By("Waiting until all VMs are gone")
		Eventually(func() int {
			vms, err := virtClient.VM(newRS.ObjectMeta.Namespace).List(v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 60*time.Second, 1*time.Second).Should(BeZero())
	})

	It("should remove owner references on the VM if it is orphan deleted", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)

		// Check for owner reference
		vms, err := virtClient.VM(newRS.ObjectMeta.Namespace).List(v12.ListOptions{})
		Expect(vms.Items).To(HaveLen(2))
		for _, vm := range vms.Items {
			Expect(vm.OwnerReferences).ToNot(BeEmpty())
		}

		// Delete it
		By("Deleting the VM replica set with the 'orphan' deletion strategy")
		orphanPolicy := v12.DeletePropagationOrphan
		Expect(virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).
			Delete(newRS.ObjectMeta.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
		// Wait until the replica set is deleted
		By("Waiting until the replica set got deleted")
		Eventually(func() bool {
			_, err := virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).Get(newRS.ObjectMeta.Name, v12.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			return false
		}, 60*time.Second, 1*time.Second).Should(BeTrue())

		By("Checking if two VMs are orphaned and still exist")
		vms, err = virtClient.VM(newRS.ObjectMeta.Namespace).List(v12.ListOptions{})
		Expect(vms.Items).To(HaveLen(2))

		By("Checking a VM owner references")
		for _, vm := range vms.Items {
			Expect(vm.OwnerReferences).To(BeEmpty())
		}
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not scale when paused and scale when resume", func() {
		rs := newReplicaSet()
		// pause controller
		By("Pausing the replicaset")
		rs.Spec.Paused = true
		_, err = virtClient.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() v1.VMReplicaSetConditionType {
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(rs.Status.Conditions) > 0 {
				return rs.Status.Conditions[0].Type
			}
			return ""
		}, 10*time.Second, 1*time.Second).Should(Equal(v1.VMReplicaSetReplicaPaused))

		// set new replica count while still being paused
		By("Updating the number of replicas")
		rs.Spec.Replicas = tests.NewInt32(2)
		_, err = virtClient.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)
		Expect(err).ToNot(HaveOccurred())

		// make sure that we don't scale up
		By("Checking that the replicaset do not scale while it is paused")
		Consistently(func() int32 {
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			// Make sure that no failure happened, so that ensure that we don't scale because we are paused
			Expect(rs.Status.Conditions).To(HaveLen(1))
			return rs.Status.Replicas
		}, 3*time.Second, 1*time.Second).Should(Equal(int32(0)))

		// resume controller
		By("Resuming the replicaset")
		rs.Spec.Paused = false
		_, err = virtClient.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)
		Expect(err).ToNot(HaveOccurred())

		// Paused condition should disappear
		By("Checking that the pause condition disappeared from the replicaset")
		Eventually(func() int {
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(rs.Status.Conditions)
		}, 10*time.Second, 1*time.Second).Should(Equal(0))

		// Replicas should be created
		By("Checking that the missing replicas are now created")
		Eventually(func() int32 {
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rs.Status.Replicas
		}, 10*time.Second, 1*time.Second).Should(Equal(int32(2)))
	})
})
