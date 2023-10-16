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
 * Copyright 2017-2023 Red Hat, Inc.
 *
 */

package vmi_configuration

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"kubevirt.io/kubevirt/tests/compute"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = compute.SIGDescribe("Configurations", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU spec", func() {
		var cpuVmi *v1.VirtualMachineInstance
		var nodes *k8sv1.NodeList

		parseCPUNiceName := func(name string) string {
			updatedCPUName := strings.Replace(name, "\n", "", -1)
			if strings.Contains(updatedCPUName, ":") {
				updatedCPUName = strings.Split(name, ":")[1]

			}
			updatedCPUName = strings.Replace(updatedCPUName, " ", "", 1)
			updatedCPUName = strings.Replace(updatedCPUName, "(", "", -1)
			updatedCPUName = strings.Replace(updatedCPUName, ")", "", -1)

			updatedCPUName = strings.Split(updatedCPUName, "-")[0]
			updatedCPUName = strings.Split(updatedCPUName, "_")[0]

			for i, char := range updatedCPUName {
				if unicode.IsUpper(char) && i != 0 {
					updatedCPUName = strings.Split(updatedCPUName, string(char))[0]
				}
			}
			return updatedCPUName
		}

		BeforeEach(func() {
			nodes = libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

			cpuVmi = libvmi.NewCirros()
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model defined", func() {
			It("[test_id:1678]should report defined CPU model", func() {
				supportedCPUs := tests.GetSupportedCPUModels(*nodes)
				Expect(supportedCPUs).ToNot(BeEmpty())
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Model: supportedCPUs[0],
				}
				niceName := parseCPUNiceName(supportedCPUs[0])

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep %s /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)).To(Succeed())
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model equals to passthrough", func() {
			It("[test_id:1679]should report exactly the same model as node CPU", func() {
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Model: "host-passthrough",
				}

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Checking the CPU model under the guest OS")
				output := tests.RunCommandOnVmiPod(cpuVmi, []string{"grep", "-m1", "model name", "/proc/cpuinfo"})

				niceName := parseCPUNiceName(output)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep '%s' /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)).To(Succeed())
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model not defined", func() {
			It("[test_id:1680]should report CPU model from libvirt capabilities", func() {
				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				output := tests.RunCommandOnVmiPod(cpuVmi, []string{"grep", "-m1", "model name", "/proc/cpuinfo"})

				niceName := parseCPUNiceName(output)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				err = console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep '%s' /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when CPU features defined", func() {
			It("[test_id:3123]should start a Virtual Machine with matching features", func() {
				supportedCPUFeatures := tests.GetSupportedCPUFeatures(*nodes)
				Expect(supportedCPUFeatures).ToNot(BeEmpty())
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{
							Name: supportedCPUFeatures[0],
						},
					},
				}

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())
			})
		})
	})

})
