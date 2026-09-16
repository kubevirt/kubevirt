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

package virtcontroller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	poolv1 "kubevirt.io/api/pool/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VM Pool Stats Collector", func() {
	newVMPool := func(name, namespace, uid string, desired *int32, replicas, readyReplicas int32) *poolv1.VirtualMachinePool {
		return &poolv1.VirtualMachinePool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       types.UID(uid),
			},
			Spec: poolv1.VirtualMachinePoolSpec{
				Replicas: desired,
			},
			Status: poolv1.VirtualMachinePoolStatus{
				Replicas:      replicas,
				ReadyReplicas: readyReplicas,
			},
		}
	}

	It("should emit no series when there are no pools", func() {
		Expect(reportVMPoolStats(nil)).To(BeEmpty())
		Expect(reportVMPoolStats([]*poolv1.VirtualMachinePool{})).To(BeEmpty())
	})

	It("should emit info and replica gauges without putting replica counts on info labels", func() {
		pool := newVMPool("pool-1", "ns-1", "uid-1", pointer.P(int32(3)), 2, 1)

		results := reportVMPoolStats([]*poolv1.VirtualMachinePool{pool})
		Expect(results).To(HaveLen(6))

		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_info"))
		Expect(results[0].Value).To(Equal(1.0))
		Expect(results[0].Labels).To(Equal([]string{"ns-1", "pool-1", "uid-1"}))

		Expect(results[1].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_desired_replicas"))
		Expect(results[1].Value).To(Equal(3.0))
		Expect(results[1].Labels).To(Equal([]string{"ns-1", "pool-1"}))

		Expect(results[2].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_replicas"))
		Expect(results[2].Value).To(Equal(2.0))
		Expect(results[2].Labels).To(Equal([]string{"ns-1", "pool-1"}))

		Expect(results[3].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_ready_replicas"))
		Expect(results[3].Value).To(Equal(1.0))
		Expect(results[3].Labels).To(Equal([]string{"ns-1", "pool-1"}))

		Expect(results[4].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_paused"))
		Expect(results[4].Value).To(Equal(0.0))
		Expect(results[4].Labels).To(Equal([]string{"ns-1", "pool-1"}))

		Expect(results[5].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_replica_failure"))
		Expect(results[5].Value).To(Equal(0.0))
		Expect(results[5].Labels).To(Equal([]string{"ns-1", "pool-1"}))
	})

	It("should default desired replicas to 1 when spec.replicas is unset", func() {
		pool := newVMPool("pool-1", "ns-1", "uid-1", nil, 0, 0)

		results := reportVMPoolStats([]*poolv1.VirtualMachinePool{pool})
		Expect(results[1].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_desired_replicas"))
		Expect(results[1].Value).To(Equal(1.0))
	})

	It("should emit one series set per pool and drop deleted objects from the list", func() {
		live := newVMPool("pool-live", "ns-1", "uid-live", pointer.P(int32(2)), 2, 2)
		done := newVMPool("pool-done", "ns-1", "uid-done", pointer.P(int32(0)), 0, 0)

		results := reportVMPoolStats([]*poolv1.VirtualMachinePool{live, done})
		Expect(results).To(HaveLen(12))
		Expect(results[0].Labels[2]).To(Equal("uid-live"))
		Expect(results[6].Labels[2]).To(Equal("uid-done"))

		remaining := reportVMPoolStats([]*poolv1.VirtualMachinePool{done})
		Expect(remaining).To(HaveLen(6))
		Expect(remaining[0].Labels[2]).To(Equal("uid-done"))
	})

	It("should set paused from spec.paused", func() {
		pool := newVMPool("pool-1", "ns-1", "uid-1", pointer.P(int32(1)), 1, 1)
		pool.Spec.Paused = true

		results := reportVMPoolStats([]*poolv1.VirtualMachinePool{pool})
		Expect(results[4].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_paused"))
		Expect(results[4].Value).To(Equal(1.0))
	})

	DescribeTable("should set replica_failure from ReplicaFailure condition", func(status k8sv1.ConditionStatus, expected float64) {
		pool := newVMPool("pool-1", "ns-1", "uid-1", pointer.P(int32(1)), 0, 0)
		if status != "" {
			pool.Status.Conditions = []poolv1.VirtualMachinePoolCondition{{
				Type:   poolv1.VirtualMachinePoolReplicaFailure,
				Status: status,
			}}
		}

		results := reportVMPoolStats([]*poolv1.VirtualMachinePool{pool})
		Expect(results[5].Metric.GetOpts().Name).To(Equal("kubevirt_vmpool_replica_failure"))
		Expect(results[5].Value).To(Equal(expected))
	},
		Entry("absent", k8sv1.ConditionStatus(""), 0.0),
		Entry("true", k8sv1.ConditionTrue, 1.0),
		Entry("false", k8sv1.ConditionFalse, 0.0),
	)
})
