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
	"context"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[Serial][rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]IgnitionData", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var LaunchVMI func(*v1.VirtualMachineInstance)

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		LaunchVMI = func(vmi *v1.VirtualMachineInstance) {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(util.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
			Expect(err).To(BeNil())

			By("Waiting the VirtualMachineInstance start")
			_, ok := obj.(*v1.VirtualMachineInstance)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
			Expect(tests.WaitForSuccessfulVMIStart(obj)).ToNot(BeEmpty())
		}
	})

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
					vmi := tests.NewRandomVMIWithEphemeralDiskHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling))

					ignitionData := "ignition injected"
					vmi.Annotations = map[string]string{v1.IgnitionAnnotation: ignitionData}

					LaunchVMI(vmi)

					Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: "login:"},
						&expect.BSnd{S: "fedora\n"},
						&expect.BExp{R: "Password:"},
						&expect.BSnd{S: "fedora" + "\n"},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: "ls /sys/firmware/qemu_fw_cfg/by_name/opt/com.coreos/config\n"},
						&expect.BExp{R: "raw"},
					}, 300)).To(Succeed())
				})
			})
		})

	})
})
