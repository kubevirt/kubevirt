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

package infrastructure

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe(SIGSerial("Start a VirtualMachineInstance", func() {
	Context("when the controller pod is not running and an election happens", func() {
		It("[test_id:4642]should elect a new controller pod", func() {
			virtClient := kubevirt.Client()
			By("Deleting the virt-controller leader pod")
			leaderPodName := libinfra.GetLeader()
			Expect(k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).Delete(
				context.Background(), leaderPodName,
				metav1.DeleteOptions{})).To(Succeed())

			By("Expecting a new leader to get elected")
			Eventually(libinfra.GetLeader, 30*time.Second, 5*time.Second).ShouldNot(Equal(leaderPodName))

			By("Starting a new VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(
				context.Background(), libvmifact.NewGuestless(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)
		})
	})
}))
