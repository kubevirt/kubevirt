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

	"github.com/onsi/ginkgo/extensions/table"

	"k8s.io/apimachinery/pkg/api/errors"

	"time"

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

	Context("A valid VirtualMachineReplicaSet given", func() {

		doScale := func(name string, scale int32) {

			// Status updates can conflict with our desire to change the spec
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

			Eventually(func() int32 {
				rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(name, v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return rs.Status.Replicas
			}, 10, 1).Should(Equal(int32(scale)))

			vms, err := virtClient.VM(tests.NamespaceTestDefault).List(v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vms.Items).To(HaveLen(int(scale)))
		}

		newReplicaSet := func() *v1.VirtualMachineReplicaSet {
			template := tests.NewRandomVMWithEphemeralDisk("kubevirt/cirros-registry-disk-demo:devel")
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
			table.Entry("to four, to six and then to zero replicas", 5, 6),
		)

		It("should update readyReplicas once VMs are up", func() {
			newRS := newReplicaSet()
			doScale(newRS.ObjectMeta.Name, 2)
			Eventually(func() int {
				rs, err := virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(newRS.ObjectMeta.Name, v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return int(rs.Status.ReadyReplicas)
			}, 120*time.Second, 1*time.Second).Should(Equal(2))
		})

		It("should not scale when paused and scale when resume", func() {
			rs := newReplicaSet()
			// pause controller
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
			rs.Spec.Replicas = tests.NewInt32(2)
			_, err = virtClient.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)
			Expect(err).ToNot(HaveOccurred())

			// make sure that we don't scale up
			Consistently(func() int32 {
				rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				// Make sure that no failure happened, so that ensure that we don't scale because we are paused
				Expect(rs.Status.Conditions).To(HaveLen(1))
				return rs.Status.Replicas
			}, 3*time.Second, 1*time.Second).Should(Equal(int32(0)))

			// resume controller
			rs.Spec.Paused = false
			_, err = virtClient.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)
			Expect(err).ToNot(HaveOccurred())

			// Paused condition should disappear
			Eventually(func() int {
				rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(rs.Status.Conditions)
			}, 10*time.Second, 1*time.Second).Should(Equal(0))

			// Replicas should be created
			Eventually(func() int32 {
				rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return rs.Status.Replicas
			}, 10*time.Second, 1*time.Second).Should(Equal(int32(2)))
		})

	})
})
