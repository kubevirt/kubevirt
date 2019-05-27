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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component]IgnitionData", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	LaunchVMI := func(vmi *v1.VirtualMachineInstance) {
		By("Starting a VirtualMachineInstance")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
		Expect(err).To(BeNil())

		By("Waiting the VirtualMachineInstance start")
		_, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		Expect(tests.WaitForSuccessfulVMIStart(obj)).ToNot(BeEmpty())
	}

	VerifyIgnitionDataVMI := func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
		By("Expecting the VirtualMachineInstance console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 10*time.Second)
		Expect(err).ToNot(HaveOccurred())
		defer expecter.Close()

		By("Checking that the VirtualMachineInstance serial console output equals to expected one")
		resp, err := expecter.ExpectBatch(commands, timeout)
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		if !tests.HasExperimentalIgnitionSupport() {
			Skip("ExperimentalIgnitionSupport feature gate is not enabled in kubevirt-config")
		}
	})

	Describe("[rfe_id:151][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with IgnitionData annotation", func() {
			Context("with injected data", func() {
				It("[test_id:1616]should have injected data under firmware directory", func() {
					vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(tests.ContainerDiskFor(tests.ContainerDiskFedora), "#!/bin/sh\n\necho fedora| passwd --stdin fedora\n")
					ignitionData := "ignition injected"
					vmi.Annotations = map[string]string{"kubevirt.io/ignitiondata": ignitionData}

					LaunchVMI(vmi)

					VerifyIgnitionDataVMI(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: "login:"},
						&expect.BSnd{S: "fedora\n"},
						&expect.BExp{R: "Password:"},
						&expect.BSnd{S: "fedora" + "\n"},
						&expect.BExp{R: "$"},
						&expect.BSnd{S: "ls /sys/firmware/qemu_fw_cfg/by_name/opt/com.coreos/config\n"},
						&expect.BExp{R: "raw"},
					}, time.Second*300)
				})
			})
		})

	})
})
