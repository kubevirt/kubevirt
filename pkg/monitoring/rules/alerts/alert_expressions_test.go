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
	const installNamespaceSelector = "namespace='kubevirt'"
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
			installNamespaceSelector,
			"kube_pod_container_status_waiting_reason",
			"topk by(pod, namespace)",
			"max by(pod, namespace, reason)",
			"kube_pod_status_ready{pod=~'virt-api-.*', namespace='kubevirt', condition='true'}",
			"or vector(0)) == 0",
		}),
		Entry("operator filters on correct pod and container", "test-ns", "operator", []string{
			"pod=~'virt-operator-.*'",
			"container='virt-operator'",
			"namespace='test-ns'",
			"kubevirt_virt_operator_ready_status{namespace='test-ns'}",
		}),
	)

	DescribeTable("componentDownFallbackExpr",
		func(namespace, component string, expect []string) {
			expectContains(componentDownFallbackExpr(namespace, component), expect)
		},
		Entry("controller fallback uses existing pods_running recording rule and unless clause", "kubevirt", "controller", []string{
			"cluster:kubevirt_virt_controller_pods_running:count",
			"unless on()",
			"kube_pod_container_status_waiting_reason{pod=~'virt-controller-.*'",
			"container='virt-controller'",
			installNamespaceSelector,
		}),
	)

	It("combines withReason and fallback branches in componentDownExpr", func() {
		expr := componentDownExpr("ci", "api")
		Expect(expr).To(ContainSubstring(") or ("))
		Expect(expr).To(ContainSubstring("topk by(pod, namespace)"))
		Expect(expr).To(ContainSubstring("vector(0)"))
	})
})
