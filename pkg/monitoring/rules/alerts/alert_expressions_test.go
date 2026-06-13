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
	"strings"
	"testing"
)

func TestComponentDownWithReasonExpr(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		component string
		expect    []string
	}{
		{
			name:      "api expression filters on correct pod and container",
			namespace: "kubevirt",
			component: "api",
			expect: []string{
				"pod=~'virt-api-.*'",
				"container='virt-api'",
				"namespace='kubevirt'",
				"kube_pod_container_status_waiting_reason",
				"topk by(pod, namespace)",
				"max by(pod, namespace, reason)",
			},
		},
		{
			name:      "handler expression filters on correct pod and container",
			namespace: "test-ns",
			component: "handler",
			expect: []string{
				"pod=~'virt-handler-.*'",
				"container='virt-handler'",
				"namespace='test-ns'",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := componentDownWithReasonExpr(tc.namespace, tc.component)
			for _, substr := range tc.expect {
				if !strings.Contains(expr, substr) {
					t.Errorf("expected expression to contain %q, got:\n%s", substr, expr)
				}
			}
		})
	}
}

func TestComponentDownFallbackExpr(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		component string
		expect    []string
	}{
		{
			name:      "controller fallback uses raw metrics and unless clause",
			namespace: "kubevirt",
			component: "controller",
			expect: []string{
				"kube_pod_status_phase{pod=~'virt-controller-.*'",
				"phase='Running'",
				"namespace='kubevirt'",
				"or vector(0)",
				"unless on()",
				"kube_pod_container_status_waiting_reason{pod=~'virt-controller-.*'",
				"container='virt-controller'",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := componentDownFallbackExpr(tc.namespace, tc.component)
			for _, substr := range tc.expect {
				if !strings.Contains(expr, substr) {
					t.Errorf("expected expression to contain %q, got:\n%s", substr, expr)
				}
			}
		})
	}
}

func TestComponentDownExpr(t *testing.T) {
	expr := componentDownExpr("ci", "api")
	if !strings.Contains(expr, ") or (") {
		t.Error("expected componentDownExpr to combine two branches with 'or'")
	}
	if !strings.Contains(expr, "topk by(pod, namespace)") {
		t.Error("expected withReason branch in componentDownExpr")
	}
	if !strings.Contains(expr, "vector(0)") {
		t.Error("expected fallback branch in componentDownExpr")
	}
}

func TestDaemonSetDownExpr(t *testing.T) {
	expr := daemonSetDownExpr("ci", "handler")
	if !strings.Contains(expr, "group_left(node)") {
		t.Error("expected daemonSetDownExpr to join kube_pod_info for node label")
	}
	if !strings.Contains(expr, "kube_pod_info{namespace='ci'}") {
		t.Error("expected kube_pod_info with correct namespace")
	}
	if !strings.Contains(expr, ") or (") {
		t.Error("expected daemonSetDownExpr to combine two branches with 'or'")
	}
}

func TestLowReadyAlertExpr(t *testing.T) {
	expr := lowReadyAlertExpr("ci", "api")
	expect := []string{
		"kubevirt_virt_api_ready_status{namespace='ci'}",
		"kube_pod_status_phase{pod=~'virt-api-.*'",
		"phase='Running'",
		"count(kubevirt_virt_api_ready_status{namespace='ci'} == 1) > 0",
	}
	for _, substr := range expect {
		if !strings.Contains(expr, substr) {
			t.Errorf("expected expression to contain %q, got:\n%s", substr, expr)
		}
	}
}

func TestLowReadyWithNodeAlertExpr(t *testing.T) {
	expr := lowReadyWithNodeAlertExpr("ci", "handler")
	expect := []string{
		"kubevirt_virt_handler_ready_status{namespace='ci'}",
		"group_left(node) kube_pod_info{namespace='ci'}",
	}
	for _, substr := range expect {
		if !strings.Contains(expr, substr) {
			t.Errorf("expected expression to contain %q, got:\n%s", substr, expr)
		}
	}
}

func TestNoReadyAlertExpr(t *testing.T) {
	expr := noReadyAlertExpr("ci", "operator")
	expect := []string{
		"kubevirt_virt_operator_ready_status{namespace='ci'}",
		"kube_pod_status_ready{pod=~'virt-operator-.*'",
		"unless on(namespace)",
	}
	for _, substr := range expect {
		if !strings.Contains(expr, substr) {
			t.Errorf("expected expression to contain %q, got:\n%s", substr, expr)
		}
	}
}
