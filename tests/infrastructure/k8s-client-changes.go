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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/libvmi/replicaset"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libreplicaset"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIGSerial("changes to the kubernetes client", func() {
	var (
		virtClient kubecli.KubevirtClient
		err        error
	)
	const replicas = 10
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})
	scheduledToRunning := func(vmis []v1.VirtualMachineInstance) time.Duration {
		var duration time.Duration
		for _, vmi := range vmis {
			start := metav1.Time{}
			stop := metav1.Time{}
			for _, timestamp := range vmi.Status.PhaseTransitionTimestamps {
				if timestamp.Phase == v1.Scheduled {
					start = timestamp.PhaseTransitionTimestamp
				} else if timestamp.Phase == v1.Running {
					stop = timestamp.PhaseTransitionTimestamp
				}
			}
			duration += stop.Sub(start.Time)
		}
		return duration
	}

	It("on the controller rate limiter should lead to delayed VMI starts", decorators.WgS390x, func() {
		By("first getting the basetime for a replicaset")
		replicaset := replicaset.New(libvmifact.NewAlpine(libvmi.WithResourceMemory("128Mi")), 0)
		replicaset, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).Create(context.Background(), replicaset, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		start := time.Now()
		libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, replicas)
		fastDuration := time.Since(start)
		libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 0)

		By("reducing the throughput on controller")
		originalKubeVirt := libkubevirt.GetCurrentKv(virtClient)
		originalKubeVirt.Spec.Configuration.ControllerConfiguration = &v1.ReloadableComponentConfiguration{
			RestClient: &v1.RESTClientConfiguration{
				RateLimiter: &v1.RateLimiter{
					TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
						Burst: 3,
						QPS:   2,
					},
				},
			},
		}
		config.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
		By("starting a replicaset with reduced throughput")
		start = time.Now()
		libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, replicas)
		slowDuration := time.Since(start)
		minExpectedDuration := 2 * fastDuration.Seconds()
		Expect(slowDuration.Seconds()).To(BeNumerically(">", minExpectedDuration))
	})

	It("on the virt handler rate limiter should lead to delayed VMI running states", func() {
		By("first getting the basetime for a replicaset")
		targetNode := libnode.GetAllSchedulableNodes(virtClient).Items[0]
		vmi := libvmi.New(
			libvmi.WithResourceMemory("1Mi"),
			libvmi.WithNodeSelectorFor(targetNode.Name),
		)

		replicaset := replicaset.New(vmi, 0)
		replicaset, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).Create(context.Background(), replicaset, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, replicas)
		Eventually(matcher.AllVMIs(replicaset.Namespace), 90*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))
		vmis, err := matcher.AllVMIs(replicaset.Namespace)()
		Expect(err).ToNot(HaveOccurred())
		fastDuration := scheduledToRunning(vmis)

		libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 0)
		Eventually(matcher.AllVMIs(replicaset.Namespace), 90*time.Second, 1*time.Second).Should(matcher.BeGone())

		By("reducing the throughput on handler")
		originalKubeVirt := libkubevirt.GetCurrentKv(virtClient)
		originalKubeVirt.Spec.Configuration.HandlerConfiguration = &v1.ReloadableComponentConfiguration{
			RestClient: &v1.RESTClientConfiguration{
				RateLimiter: &v1.RateLimiter{
					TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
						Burst: 1,
						QPS:   1,
					},
				},
			},
		}
		config.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)

		By("starting a replicaset with reduced throughput")
		libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, replicas)
		Eventually(matcher.AllVMIs(replicaset.Namespace), 180*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))
		vmis, err = matcher.AllVMIs(replicaset.Namespace)()
		Expect(err).ToNot(HaveOccurred())
		slowDuration := scheduledToRunning(vmis)
		minExpectedDuration := 1.5 * fastDuration.Seconds()
		Expect(slowDuration.Seconds()).To(BeNumerically(">", minExpectedDuration))
	})
}))
