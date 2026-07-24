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

func lowReadyAlertExpr(namespace, component string) string {
	return fmt.Sprintf(
		"kubevirt_virt_%s_ready_status{namespace='%s'} == 0 "+
			"and on(pod, namespace) "+
			"kube_pod_status_phase{pod=~'virt-%s-.*', namespace='%s', phase='Running'} == 1 "+
			"and on() "+
			"count(kubevirt_virt_%s_ready_status{namespace='%s'} == 1) > 0",
		component, namespace, component, namespace, component, namespace,
	)
}

func lowReadyWithNodeAlertExpr(namespace, component string) string {
	return fmt.Sprintf(
		"(kubevirt_virt_%s_ready_status{namespace='%s'} == 0 "+
			"and on(pod, namespace) "+
			"kube_pod_status_phase{pod=~'virt-%s-.*', namespace='%s', phase='Running'} == 1) "+
			"* on(pod, namespace) group_left(node) kube_pod_info{namespace='%s'} "+
			"and on() "+
			"count(kubevirt_virt_%s_ready_status{namespace='%s'} == 1) > 0",
		component, namespace, component, namespace, namespace, component, namespace,
	)
}

func noReadyAlertExpr(namespace, component string) string {
	return fmt.Sprintf(
		"count by (namespace) (kubevirt_virt_%s_ready_status{namespace='%s'}) > 0 "+
			"unless on(namespace) "+
			"count by (namespace) ("+
			"kube_pod_status_ready{pod=~'virt-%s-.*', namespace='%s', condition='true'} "+
			"* on(pod, namespace) "+
			"kubevirt_virt_%s_ready_status{namespace='%s'} "+
			"== 1"+
			") > 0",
		component, namespace, component, namespace, component, namespace,
	)
}

// componentDownWithReasonExpr returns a per-pod expression that fires
// when a virt-* container has a waiting reason (ImagePullBackOff,
// CrashLoopBackOff, ErrImagePull, etc.).
//
// A container with a waiting reason is non-functional regardless of
// pod phase — CrashLoopBackOff pods can have phase=Running while the
// container restarts repeatedly. The metric only exists when a
// container is genuinely in a waiting state, so no phase guard is
// needed.
//
// Inner topk-by selects exactly one reason per pod, avoiding
// duplicate alerts when kube-state-metrics reports multiple
// reasons for the same failure (e.g. ErrImagePull + ImagePullBackOff).
// When values tie, topk's selection is deterministic per series
// fingerprint but not alphabetical. Outer max-by strips the
// container label so it doesn't leak into alert routing or grouping.
//
// Filters on container='virt-<component>' to ignore init containers
// and injected sidecars (e.g. service-mesh proxies).
func componentDownWithReasonExpr(namespace, component string) string {
	return fmt.Sprintf(
		"max by(pod, namespace, reason) (topk by(pod, namespace) (1, "+
			"kube_pod_container_status_waiting_reason"+
			"{pod=~'virt-%s-.*', container='virt-%s', namespace='%s'} > 0))",
		component, component, namespace,
	)
}

// componentDownFallbackExpr returns an instant-vector expression that
// fires when no virt-* pods are running AND the withReason branch
// produced nothing (i.e. no waiting_reason metrics exist — pods are
// entirely absent or haven't registered container status yet). Uses raw
// metrics instead of recording rules to stay consistent with the
// withReason branch.
func componentDownFallbackExpr(namespace, component string) string {
	return fmt.Sprintf(
		"(count(kube_pod_status_phase{pod=~'virt-%s-.*', namespace='%s', phase='Running'} == 1) or vector(0)) == 0 "+
			"unless on() "+
			"(count(kube_pod_container_status_waiting_reason{pod=~'virt-%s-.*', container='virt-%s', namespace='%s'} > 0) > 0)",
		component, namespace, component, component, namespace,
	)
}

// componentDownExpr builds an expression for Virt*Down alerts.
//
// Branch 1 (diagnostic): any virt-* pod has a container in a
// waiting state — fires per-pod with pod and reason labels.
// Covers CrashLoopBackOff (even with pod phase=Running),
// ImagePullBackOff, ErrImagePull, and any other waiting reason.
//
// Branch 2 (fallback): no pods are running AND branch 1 produced
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

// daemonSetDownExpr is like componentDownExpr but also joins kube_pod_info
// to include the node label, since virt-handler runs as a DaemonSet.
func daemonSetDownExpr(namespace, component string) string {
	withReason := fmt.Sprintf(
		"(%s) * on(pod, namespace) group_left(node) kube_pod_info{namespace='%s'}",
		componentDownWithReasonExpr(namespace, component), namespace,
	)

	return fmt.Sprintf("(%s) or (%s)",
		withReason,
		componentDownFallbackExpr(namespace, component),
	)
}

func getErrorRatio(ns, podName, errorCodeRegex string, durationInMinutes int) string {
	errorRatioQuery := "sum ( rate ( kubevirt_rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\",code=~\"%s\"} [%dm] ) )  / " +
		" sum ( rate ( kubevirt_rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\"} [%dm] ) )"
	return fmt.Sprintf(errorRatioQuery, ns, podName, errorCodeRegex, durationInMinutes, ns, podName, durationInMinutes)
}

func getRestCallsFailedWarning(failingCallsPercentage int, component string, durationInMinutes int) string {
	duration := fmt.Sprintf("%d minutes", durationInMinutes)

	const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
	return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
}
