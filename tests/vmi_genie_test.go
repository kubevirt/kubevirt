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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Genie", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	var detachedVMI *v1.VirtualMachineInstance

	tests.BeforeAll(func() {
		tests.BeforeTestCleanup()
	})

	Context("VirtualMachineInstance with cni flannel and ptp plugin interface", func() {
		AfterEach(func() {
			virtClient.VirtualMachineInstance("default").Delete(detachedVMI.Name, &v13.DeleteOptions{})
			fmt.Printf("Waiting for vmi %s in default namespace to be removed, this can take a while ...\n", detachedVMI.Name)
			EventuallyWithOffset(1, func() bool {
				return errors.IsNotFound(virtClient.VirtualMachineInstance("default").Delete(detachedVMI.Name, nil))
			}, 180*time.Second, 1*time.Second).
				Should(BeTrue())
		})

		It("should create a virtual machine with one interface", func() {
			detachedVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

			detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			detachedVMI.Spec.Networks = []v1.Network{
				{Name: "ptp", NetworkSource: v1.NetworkSource{
					Genie: &v1.GenieNetwork{NetworkName: "ptp"},
				}},
			}

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReadyWithNamespace(tests.NamespaceTestDefault, detachedVMI, tests.LoggedInCirrosExpecter)

			cmdCheck := fmt.Sprintf("ping %s -c 1 -w 5\n", "10.1.1.1")
			err = tests.CheckForTextExpecter(detachedVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: cmdCheck},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 180)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create a virtual machine with two interfaces", func() {
			detachedVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

			detachedVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
			detachedVMI.Spec.Networks = []v1.Network{
				{Name: "default", NetworkSource: v1.NetworkSource{
					Genie: &v1.GenieNetwork{NetworkName: "flannel"},
				}},
				{Name: "ptp", NetworkSource: v1.NetworkSource{
					Genie: &v1.GenieNetwork{NetworkName: "ptp"},
				}},
			}

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReadyWithNamespace(tests.NamespaceTestDefault, detachedVMI, tests.LoggedInCirrosExpecter)

			By("checking virtual machine instance as two interfaces")
			cmdCheck := fmt.Sprintf("ip link show %s\n", "eth0")
			err = tests.CheckForTextExpecter(detachedVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: cmdCheck},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 180)
			Expect(err).ToNot(HaveOccurred())

			cmdCheck = fmt.Sprintf("ip link show %s\n", "eth1")
			err = tests.CheckForTextExpecter(detachedVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: cmdCheck},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 180)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
