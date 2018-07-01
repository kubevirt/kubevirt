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

var _ = Describe("VirtualMachineInstanceReplicaSet", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	doScale := func(name string, scale int32) {

		// Status updates can conflict with our desire to change the spec
		By(fmt.Sprintf("Scaling to %d", scale))
		var rs *v1.VirtualMachineInstanceReplicaSet
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
		}, 90*time.Second, time.Second).Should(Equal(int32(scale)))

		vmis, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tests.NotDeleted(vmis)).To(HaveLen(int(scale)))
	}

	newReplicaSet := func() *v1.VirtualMachineInstanceReplicaSet {
		By("Create a new VirtualMachineInstance replica set")
		template := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskCirros))
		newRS := tests.NewRandomReplicaSetFromVMI(template, int32(0))
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
			Kind:       "VirtualMachineInstanceReplicaSet",
		}

		jsonBytes, err := json.Marshal(newRS)
		Expect(err).To(BeNil())

		// change the name of a required field (like domain) so validation will fail
		jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

		result := virtClient.RestClient().Post().Resource("virtualmachineinstancereplicasets").Namespace(tests.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do()

		// Verify validation failed.
		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

	})
	It("should reject POST if validation webhoook deems the spec is invalid", func() {
		newRS := newReplicaSet()
		newRS.TypeMeta = v12.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceReplicaSet",
		}

		// Add a disk that doesn't map to a volume.
		// This should get rejected which tells us the webhook validator is working.
		newRS.Spec.Template.Spec.Domain.Devices.Disks = append(newRS.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
			Name:       "testdisk",
			VolumeName: "testvolume",
		})

		result := virtClient.RestClient().Post().Resource("virtualmachineinstancereplicasets").Namespace(tests.NamespaceTestDefault).Body(newRS).Do()

		// Verify validation failed.
		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

		reviewResponse := &v12.Status{}
		body, _ := result.Raw()
		err = json.Unmarshal(body, reviewResponse)
		Expect(err).To(BeNil())

		Expect(len(reviewResponse.Details.Causes)).To(Equal(1))
		Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[1].volumeName"))
	})
	It("should update readyReplicas once VMIs are up", func() {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, 2)

		By("Checking the number of ready replicas")
		Eventually(func() int {
			rs, err := virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(newRS.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return int(rs.Status.ReadyReplicas)
		}, 120*time.Second, 1*time.Second).Should(Equal(2))
	})

	It("should remove VMIs once it is marked for deletion", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)
		// Delete it
		By("Deleting the VirtualMachineInstance replica set")
		Expect(virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).Delete(newRS.ObjectMeta.Name, &v12.DeleteOptions{})).To(Succeed())
		// Wait until VMIs are gone
		By("Waiting until all VMIs are gone")
		Eventually(func() int {
			vmis, err := virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(&v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vmis.Items)
		}, 120*time.Second, 1*time.Second).Should(BeZero())
	})

	It("should remove owner references on the VirtualMachineInstance if it is orphan deleted", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)

		// Check for owner reference
		vmis, err := virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(&v12.ListOptions{})
		Expect(vmis.Items).To(HaveLen(2))
		for _, vmi := range vmis.Items {
			Expect(vmi.OwnerReferences).ToNot(BeEmpty())
		}

		// Delete it
		By("Deleting the VirtualMachineInstance replica set with the 'orphan' deletion strategy")
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

		By("Checking if two VMIs are orphaned and still exist")
		vmis, err = virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(&v12.ListOptions{})
		Expect(vmis.Items).To(HaveLen(2))

		By("Checking a VirtualMachineInstance owner references")
		for _, vmi := range vmis.Items {
			Expect(vmi.OwnerReferences).To(BeEmpty())
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

		Eventually(func() v1.VirtualMachineInstanceReplicaSetConditionType {
			rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(rs.Status.Conditions) > 0 {
				return rs.Status.Conditions[0].Type
			}
			return ""
		}, 10*time.Second, 1*time.Second).Should(Equal(v1.VirtualMachineInstanceReplicaSetReplicaPaused))

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

	It("should remove the finished VM", func() {
		By("Creating new replica set")
		rs := newReplicaSet()
		doScale(rs.ObjectMeta.Name, int32(2))

		vmis, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmis.Items).ToNot(BeEmpty())

		vmi := vmis.Items[0]
		pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(&vmi))
		Expect(err).ToNot(HaveOccurred())
		Expect(len(pods.Items)).To(Equal(1))
		pod := pods.Items[0]

		By("Deleting one of the RS VMS pods")
		err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Delete(pod.Name, &v12.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking that the VM dissapeared")
		Eventually(func() bool {
			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &v12.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			return false
		}, 120*time.Second, time.Second).Should(Equal(true))

		By("Checking number of RS VM's")
		vmis, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vmis.Items)).Should(Equal(2))
	})
})
