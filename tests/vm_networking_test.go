/*
 * This file is part of the kubevirt project
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

	"github.com/google/goexpect"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Networking", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	virtConfig, err := kubecli.GetKubevirtClientConfig()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("VirtualMachine With nodeNetwork definition given", func() {

		It("should connect via DHCP to the node network", func() {
			vm, err := tests.NewRandomVMWithEphemeralDiskAndUserdata("kubevirt/cirros-registry-disk-demo:devel", "noCloud", "#!/bin/bash\necho 'hello'\n")
			Expect(err).ToNot(HaveOccurred())

			// add node network
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{Type: "nodeNetwork"}}

			// Start VM
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			// Wait until the VM is booted, ping google and check if we can reach the internet
			expecter, _, err := tests.NewConsoleExpecter(virtConfig, vm, "", 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "cirros login: "},
				&expect.BSnd{S: "cirros\n"},
				&expect.BExp{R: "Password: "},
				&expect.BSnd{S: "cubswin:)\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "ping www.google.com -c 1 -w 5\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 90*time.Second)
			Expect(err).ToNot(HaveOccurred())
		}, 120)
	})

})
