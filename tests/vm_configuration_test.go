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

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Configurations", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("New VM with different cpu topologies give", func() {

		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vm = tests.NewRandomVMWithDirectLun(2, false)
		})
		It("should report 3 cpu cores", func() {
			vm.Spec.Domain.CPU = &v1.CPU{
				Cores: 3,
			}

			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "Welcome to Alpine"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "login"},
				&expect.BSnd{S: "root\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
				&expect.BExp{R: "3"},
			}, 250*time.Second)

			Expect(err).ToNot(HaveOccurred())
		}, 300)
	})
})
