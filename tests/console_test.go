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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
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

	RunVMIAndWaitForStart := func(vmi *v1.VirtualMachineInstance) {
		By("Creating a new VirtualMachineInstance")
		Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())

		By("Waiting until it starts")
		tests.WaitForSuccessfulVMIStart(vmi)
	}

	ExpectConsoleOutput := func(vmi *v1.VirtualMachineInstance, expected string) {
		By("Expecting the VirtualMachineInstance console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			By("Closing the opened expecter")
			expecter.Close()
		}()

		By("Checking that the console output equals to expected one")
		_, err = expecter.ExpectBatch([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: expected},
		}, 120*time.Second)
		Expect(err).ToNot(HaveOccurred())
	}

	OpenConsole := func(vmi *v1.VirtualMachineInstance) (expect.Expecter, <-chan error) {
		By("Expecting the VirtualMachineInstance console")
		expecter, errChan, err := tests.NewConsoleExpecter(virtClient, vmi, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		return expecter, errChan
	}

	Describe("A new VirtualMachineInstance", func() {
		Context("with a serial console", func() {
			Context("with a cirros image", func() {

				It("should return that we are running cirros", func() {
					vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
					RunVMIAndWaitForStart(vmi)
					ExpectConsoleOutput(
						vmi,
						"login as 'cirros' user",
					)
				}, 140)
			})

			Context("with a fedora image", func() {
				It("should return that we are running fedora", func() {
					vmi := tests.NewRandomVMIWithEphemeralDiskHighMemory(tests.ContainerDiskFor(tests.ContainerDiskFedora))
					RunVMIAndWaitForStart(vmi)
					ExpectConsoleOutput(
						vmi,
						"Welcome to",
					)
				}, 140)
			})

			It("should be able to reconnect to console multiple times", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))

				RunVMIAndWaitForStart(vmi)

				for i := 0; i < 5; i++ {
					ExpectConsoleOutput(vmi, "login")
				}
			}, 220)

			It("should close console connection when new console connection is opened", func(done Done) {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))

				RunVMIAndWaitForStart(vmi)

				By("opening 1st console connection")
				expecter, errChan := OpenConsole(vmi)
				defer expecter.Close()

				By("expecting error on 1st console connection")
				go func() {
					defer GinkgoRecover()
					select {
					case receivedErr := <-errChan:
						Expect(receivedErr.Error()).To(ContainSubstring("close"))
						close(done)
					case <-time.After(60 * time.Second):
						Fail("timed out waiting for closed 1st connection")
					}
				}()

				By("opening 2nd console connection")
				ExpectConsoleOutput(vmi, "login")
			}, 220)

			It("should wait until the virtual machine is in running state and return a stream interface", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				By("Creating a new VirtualMachineInstance")
				_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, 30*time.Second)
				Expect(err).ToNot(HaveOccurred())
			}, 220)

			It("should fail waiting for the virtual machine instance to be running", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
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
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
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
