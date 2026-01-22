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
 * Copyright The KubeVirt Authors.
 *
 */

package virtiofs

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage] ContainerPath virtiofs volumes", decorators.SigStorage, decorators.VirtioFS, func() {
	Context("With a ContainerPath volume pointing to non-existent path", func() {
		const (
			containerPathFilesystemName = "nonexistent-path"
			nonExistentPath             = "/this/path/does/not/exist"
		)

		It("Should set Synchronized=False with MissingVirtiofsContainers reason", func() {
			virtClient := kubevirt.Client()

			By("Creating a VMI with ContainerPath pointing to non-existent path")
			vmi := libvmifact.NewAlpine(
				libvmi.WithFilesystemContainerPath(containerPathFilesystemName, nonExistentPath),
			)

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to have Synchronized=False condition with MissingVirtiofsContainers reason")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, condition := range vmi.Status.Conditions {
					if condition.Type == v1.VirtualMachineInstanceSynchronized &&
						condition.Status == k8sv1.ConditionFalse &&
						condition.Reason == v1.MissingVirtiofsContainersReason {
						return true
					}
				}
				return false
			}, 120*time.Second, time.Second).Should(BeTrue(), "VMI should have Synchronized=False with MissingVirtiofsContainers reason")
		})
	})
})
