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

		table.DescribeTable("should scale", func(startScale int, stopScale int) {

			template := tests.NewRandomVM()
			newRS := tests.NewRandomReplicaSetFromVM(template, int32(0))
			newRS, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Create(newRS)
			Expect(err).ToNot(HaveOccurred())

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

				//resourceVersion := rs.ObjectMeta.ResourceVersion
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() int32 {
					rs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(name, v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return rs.Status.Replicas
				}, 10, 1).Should(Equal(int32(scale)))

				vms, err := virtClient.VM(tests.NamespaceTestDefault).List(v12.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vms.Items).To(HaveLen(int(scale)))

				/*
					for _, vm := range vms.Items {
						tests.NewObjectEventWatcher(&vm).SinceResourceVersion(resourceVersion).Timeout(20 * time.Second).WaitFor(tests.NormalEvent, watch.SuccessfulCreateVirtualMachineReason)
						tests.NewObjectEventWatcher(&vm).SinceResourceVersion(resourceVersion).Timeout(20 * time.Second).WaitFor(tests.NormalEvent, watch.SuccessfulDeleteVirtualMachineReason)
					}
				*/
				time.Sleep(20)
			}

			doScale(newRS.ObjectMeta.Name, int32(startScale))
			doScale(newRS.ObjectMeta.Name, int32(stopScale))
			doScale(newRS.ObjectMeta.Name, int32(0))

		},
			table.FEntry("to three, to two and then to zero replicas", 3, 2),
			table.Entry("to four, to six and then to zero replicas", 5, 6),
		)
	})
})
