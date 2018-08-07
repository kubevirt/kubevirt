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

	k8sv1 "k8s.io/api/core/v1"
)

var _ = Describe("Console", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	RunVMIAndExpectConsoleOutput := func(vmi *v1.VirtualMachineInstance, expected string) {

		By("Creating a new VirtualMachineInstance")
		Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())

		By("Expecting the VirtualMachineInstance console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		defer expecter.Close()

		By("Checking that the console output equals to expected one")
		_, err = expecter.ExpectBatch([]expect.Batcher{
			&expect.BExp{R: expected},
		}, 120*time.Second)
		Expect(err).ToNot(HaveOccurred())
	}

	Describe("A new VirtualMachineInstance", func() {
		Context("with a serial console", func() {
			Context("with a cirros image", func() {
				It("should return that we are running cirros", func() {
					vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n")
					RunVMIAndExpectConsoleOutput(
						vmi,
						"login as 'cirros' user",
					)
				}, 140)
			})

			Context("with a fedora image", func() {
				It("should return that we are running fedora", func() {
					vmi := tests.NewRandomVMIWithEphemeralDiskHighMemory(tests.RegistryDiskFor(tests.RegistryDiskFedora))
					RunVMIAndExpectConsoleOutput(
						vmi,
						"Welcome to",
					)
				}, 140)
			})

			It("should be able to reconnect to console multiple times", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))

				By("Creating a new VirtualMachineInstance")
				Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())

				for i := 0; i < 5; i++ {
					By("Checking that the console output equals to expected one")
					err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: "login"},
					}, 160,
					)
					Expect(err).ToNot(HaveOccurred())
				}
			}, 220)
			It("should wait until the virtual machine is in running state and return a stream interface", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
				By("Creating a new VirtualMachineInstance")
				Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())

				_, err := virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, 30*time.Second)
				Expect(err).ToNot(HaveOccurred())
			}, 220)
			It("should fail waiting for the virtual machine instance to be running", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
				vmi.Spec.Affinity = &k8sv1.Affinity{
					NodeAffinity: &k8sv1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
							NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
								{
									MatchExpressions: []k8sv1.NodeSelectorRequirement{
										{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{"notexist"}},
									},
								},
							},
						},
					},
				}

				By("Creating a new VirtualMachineInstance")
				Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())

				_, err := virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, 30*time.Second)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Timeout trying to connect to the virtual machine instance"))
			}, 180)
			It("should fail waiting for the expecter", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
				vmi.Spec.Affinity = &k8sv1.Affinity{
					NodeAffinity: &k8sv1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
							NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
								{
									MatchExpressions: []k8sv1.NodeSelectorRequirement{
										{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{"notexist"}},
									},
								},
							},
						},
					},
				}

				By("Creating a new VirtualMachineInstance")
				Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())

				By("Expecting the VirtualMachineInstance console")
				_, _, err := tests.NewConsoleExpecter(virtClient, vmi, 30*time.Second)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Timeout trying to connect to the virtual machine instance"))
			}, 180)
		})
	})
})
