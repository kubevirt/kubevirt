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
 */

package alerts

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("alert expressions", func() {
	expectContains := func(expr string, substrings []string) {
		for _, substr := range substrings {
			Expect(expr).To(ContainSubstring(substr), "expression missing %q:\n%s", substr, expr)
		}
	}

	DescribeTable("componentDownWithReasonExpr",
		func(namespace, component string, expect []string) {
			expectContains(componentDownWithReasonExpr(namespace, component), expect)
		},
		Entry("api filters on correct pod and container", "kubevirt", "api", []string{
			"pod=~'virt-api-.*'",
			"container='virt-api'",
			"namespace='kubevirt'",
			"kube_pod_container_status_waiting_reason",
			"topk by(pod, namespace)",
			"max by(pod, namespace, reason)",
			"kubevirt_virt_api_ready_status{namespace='kubevirt'}",
			"or vector(0)) == 0",
		}),
		Entry("handler filters on correct pod and container", "test-ns", "handler", []string{
			"pod=~'virt-handler-.*'",
			"container='virt-handler'",
			"namespace='test-ns'",
			"kubevirt_virt_handler_ready_status{namespace='test-ns'}",
		}),
	)

	DescribeTable("componentDownFallbackExpr",
		func(namespace, component string, expect []string) {
			expectContains(componentDownFallbackExpr(namespace, component), expect)
		},
		Entry("controller fallback uses raw metrics and unless clause", "kubevirt", "controller", []string{
			"kube_pod_status_phase{pod=~'virt-controller-.*'",
			"phase='Running'",
			"namespace='kubevirt'",
			"or vector(0)",
			"unless on()",
			"kube_pod_container_status_waiting_reason{pod=~'virt-controller-.*'",
			"container='virt-controller'",
		}),
	)

	It("combines withReason and fallback branches in componentDownExpr", func() {
		expr := componentDownExpr("ci", "api")
		Expect(expr).To(ContainSubstring(") or ("))
		Expect(expr).To(ContainSubstring("topk by(pod, namespace)"))
		Expect(expr).To(ContainSubstring("vector(0)"))
	})

	It("joins kube_pod_info for node label in daemonSetDownExpr", func() {
		expr := daemonSetDownExpr("ci", "handler")
		Expect(expr).To(ContainSubstring("group_left(node)"))
		Expect(expr).To(ContainSubstring("kube_pod_info{namespace='ci'}"))
		Expect(expr).To(ContainSubstring(") or ("))
	})

	It("builds lowReadyAlertExpr with ready peer gate", func() {
		expectContains(lowReadyAlertExpr("ci", "api"), []string{
			"kubevirt_virt_api_ready_status{namespace='ci'}",
			"kube_pod_status_phase{pod=~'virt-api-.*'",
			"phase='Running'",
			"count(kubevirt_virt_api_ready_status{namespace='ci'} == 1) > 0",
		})
	})

	It("builds lowReadyWithNodeAlertExpr with node join", func() {
		expectContains(lowReadyWithNodeAlertExpr("ci", "handler"), []string{
			"kubevirt_virt_handler_ready_status{namespace='ci'}",
			"group_left(node) kube_pod_info{namespace='ci'}",
		})
	})

	It("builds noReadyAlertExpr with unless clause", func() {
		expectContains(noReadyAlertExpr("ci", "operator"), []string{
			"kubevirt_virt_operator_ready_status{namespace='ci'}",
			"kube_pod_status_ready{pod=~'virt-operator-.*'",
			"unless on(namespace)",
		})
	})
})
