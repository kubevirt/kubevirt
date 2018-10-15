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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("IOThreads", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
	})

	Context("IOThreads Policies", func() {

		It("Should honor shared ioThreadsPolicy", func() {
			policy := v1.IOThreadsPolicyShared
			vmi.Spec.Domain.IOThreadsPolicy = &policy

			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			listOptions := metav1.ListOptions{}

			Eventually(func() int {
				podList, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(listOptions)
				Expect(err).ToNot(HaveOccurred())
				return len(podList.Items)
			}, 75, 0.5).Should(Equal(1))

			Eventually(func() error {
				podList, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(listOptions)
				Expect(err).ToNot(HaveOccurred())
				for _, item := range podList.Items {
					if strings.HasPrefix(item.Name, vmi.ObjectMeta.GenerateName) {
						return nil
					}
				}
				return fmt.Errorf("Associated pod for VirtualMachineInstance '%s' not found", vmi.Name)
			}, 75, 0.5).Should(Succeed())

			getOptions := metav1.GetOptions{}
			var newVMI *v1.VirtualMachineInstance

			newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &getOptions)
			Expect(err).ToNot(HaveOccurred())

			domain := &api.Domain{}
			context := &api.ConverterContext{
				VirtualMachine: newVMI,
				UseEmulation:   true,
			}
			api.Convert_v1_VirtualMachine_To_api_Domain(newVMI, domain, context)

			expectedIOThreads := 1
			Expect(int(domain.Spec.IOThreads.IOThreads)).To(Equal(expectedIOThreads))

			Expect(len(newVMI.Spec.Domain.Devices.Disks)).To(Equal(1))
		})

	})
})
