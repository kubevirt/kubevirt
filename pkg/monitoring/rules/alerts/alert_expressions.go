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

import "fmt"

// componentDownWithReasonExpr reports one waiting reason per pod only when
// no healthy peer remains. API readiness comes from kube-state-metrics because
// these releases do not export kubevirt_virt_api_ready_status.
func componentDownWithReasonExpr(namespace, component string) string {
	waitingReason := fmt.Sprintf(
		"max by(pod, namespace, reason) (topk by(pod, namespace) (1, "+
			"kube_pod_container_status_waiting_reason"+
			"{pod=~'virt-%s-.*', container='virt-%s', namespace='%s'} > 0))",
		component, component, namespace,
	)
	ready := fmt.Sprintf("kubevirt_virt_%s_ready_status{namespace='%s'}", component, namespace)
	if component == "api" {
		ready = fmt.Sprintf("kube_pod_status_ready{pod=~'virt-api-.*', namespace='%s', condition='true'}", namespace)
	}
	return fmt.Sprintf(
		"(%s) and on() ((count(%s == 1) or vector(0)) == 0)",
		waitingReason, ready,
	)
}

// componentDownFallbackExpr retains the release branch's recording-rule fallback
// when kube-state-metrics does not report a container waiting reason.
// Only virt-controller has a pods_running recording rule on this branch.
func componentDownFallbackExpr(namespace, component string) string {
	running := fmt.Sprintf("kubevirt_virt_%s_up", component)
	if component == "controller" {
		running = "cluster:kubevirt_virt_controller_pods_running:count"
	}
	return fmt.Sprintf(
		"%s == 0 "+
			"unless on() "+
			"(count(kube_pod_container_status_waiting_reason{pod=~'virt-%s-.*', container='virt-%s', namespace='%s'} > 0) > 0)",
		running, component, component, namespace,
	)
}

// componentDownExpr builds an expression for Virt*Down alerts.
//
// Branch 1 (diagnostic): virt-* pods have a container in a waiting
// state and no ready peer remains — fires per-pod with pod and reason
// labels. Covers CrashLoopBackOff (even with pod phase=Running),
// ImagePullBackOff, ErrImagePull, and any other waiting reason when
// the component is fully unavailable.
//
// Branch 2 (fallback): no pods are up AND branch 1 produced
// nothing — covers pods entirely absent, or pods in Failed/Unknown
// state without a waiting reason.
//
// The fallback suppresses itself whenever any waiting_reason metric
// exists, so exactly one branch fires at a time.
func componentDownExpr(namespace, component string) string {
	return fmt.Sprintf("(%s) or (%s)",
		componentDownWithReasonExpr(namespace, component),
		componentDownFallbackExpr(namespace, component),
	)
}
