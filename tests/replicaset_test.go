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

		It("should scale up to three replicas", func() {

			template := tests.NewRandomVM()
			rs := tests.NewRandomReplicaSetFromVM(template, 3)
			rs, err := virtClient.ReplicaSet(tests.NamespaceTestDefault).Create(rs)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int32 {
				res, err := virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return res.Status.Replicas
			}, 10, 1).Should(Equal(int32(3)))

			vms, err := virtClient.VM(tests.NamespaceTestDefault).List(v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vms.Items).To(HaveLen(3))
		})
	})

})
