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

package infrastructure

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libwait"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/flags"
)

var _ = DescribeInfra("Start a VirtualMachineInstance", func() {

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("when the controller pod is not running and an election happens", func() {
		It("[test_id:4642]should succeed afterwards", func() {
			// This test needs at least 2 controller pods. Skip on single-replica.
			checks.SkipIfSingleReplica(virtClient)

			newLeaderPod := libinfra.GetNewLeaderPod(virtClient)
			Expect(newLeaderPod).NotTo(BeNil())

			// TODO: It can be race condition when newly deployed pod receive leadership, in this case we will need
			// to reduce Deployment replica before destroying the pod and to restore it after the test
			By("Destroying the leading controller pod")
			Eventually(func() string {
				leaderPodName := libinfra.GetLeader()

				Expect(virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Delete(context.Background(), leaderPodName, metav1.DeleteOptions{})).To(Succeed())

				Eventually(libinfra.GetLeader, 30*time.Second, 5*time.Second).ShouldNot(Equal(leaderPodName))

				leaderPod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), libinfra.GetLeader(), metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return leaderPod.Name
			}, 90*time.Second, 5*time.Second).Should(Equal(newLeaderPod.Name))

			Expect(matcher.ThisPod(newLeaderPod)()).To(matcher.HaveConditionTrue(k8sv1.PodReady))

			vmi := libvmi.NewAlpine()

			By("Starting a new VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Get()
			Expect(err).ToNot(HaveOccurred())
			vmiObj, ok := obj.(*v1.VirtualMachineInstance)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
			libwait.WaitForSuccessfulVMIStart(vmiObj)
		})
	})

})
