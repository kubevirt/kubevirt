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

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Console", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("New VM with a serial console given", func() {

		It("should be returned that we are running cirros", func() {
			vm := tests.NewRandomVMWithPVC("disk-cirros")

			Expect(virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()).To(Succeed())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "checking http://169.254.169.254/2009-04-04/instance-id"},
			}, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should be returned that we are running fedora", func() {

			vm := tests.NewRandomVMWithEphemeralDiskHighMemory("kubevirt/fedora-cloud-registry-disk-demo:devel")

			Expect(virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()).To(Succeed())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "Welcome to"},
			}, 120*time.Second)
			Expect(err).ToNot(HaveOccurred())
		}, 140)

		It("should be able to reconnect to console multiple times", func() {
			vm := tests.NewRandomVMWithPVC("disk-alpine")

			Expect(virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()).To(Succeed())
			tests.WaitForSuccessfulVMStart(vm)

			for i := 0; i < 5; i++ {
				expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
				defer expecter.Close()
				Expect(err).ToNot(HaveOccurred())

				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "login"},
				}, 130*time.Second)
				Expect(err).ToNot(HaveOccurred())
			}
		}, 220)
	})
})
