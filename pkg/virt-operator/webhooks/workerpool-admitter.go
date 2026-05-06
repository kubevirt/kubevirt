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

package webhooks

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workerv1 "kubevirt.io/api/worker/v1alpha1"
)

type WorkerPoolAdmitter struct{}

func (admitter *WorkerPoolAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	pool := &workerv1.WorkerPool{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pool); err != nil {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("failed to decode WorkerPool: %v", err),
				Code:    http.StatusBadRequest,
			},
		}
	}

	var causes []metav1.StatusCause

	if pool.Spec.VirtHandlerImage == "" && pool.Spec.VirtLauncherImage == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "pool must specify at least one of virtHandlerImage or virtLauncherImage",
			Field:   "spec",
		})
	}

	if len(pool.Spec.Selector.DeviceNames) == 0 && pool.Spec.Selector.VMLabels == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "pool selector must define at least one of deviceNames or vmLabels",
			Field:   "spec.selector",
		})
	}

	if len(pool.Spec.NodeSelector) == 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "pool must specify a non-empty nodeSelector",
			Field:   "spec.nodeSelector",
		})
	}

	if len(causes) > 0 {
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: "WorkerPool validation failed",
				Code:    http.StatusUnprocessableEntity,
				Details: &metav1.StatusDetails{
					Causes: causes,
				},
			},
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func isSubset(subset, superset map[string]string) bool {
	for k, v := range subset {
		if superset[k] != v {
			return false
		}
	}
	return true
}

func WarnOverlappingWorkerPools(pool *workerv1.WorkerPool, existingPools []workerv1.WorkerPool) []string {
	var warnings []string
	for _, other := range existingPools {
		if other.Name == pool.Name {
			continue
		}

		for _, dn := range pool.Spec.Selector.DeviceNames {
			if slices.Contains(other.Spec.Selector.DeviceNames, dn) {
				warnings = append(warnings, fmt.Sprintf(
					"pools %q and %q have overlapping deviceName %q; first-match-wins (alphabetical) applies",
					pool.Name, other.Name, dn))
			}
		}

		if pool.Spec.Selector.VMLabels != nil && other.Spec.Selector.VMLabels != nil {
			lp := pool.Spec.Selector.VMLabels.MatchLabels
			lo := other.Spec.Selector.VMLabels.MatchLabels
			if isSubset(lp, lo) || isSubset(lo, lp) {
				warnings = append(warnings, fmt.Sprintf(
					"pools %q and %q have overlapping vmLabels; first-match-wins (alphabetical) applies",
					pool.Name, other.Name))
			}
		}

		if maps.Equal(pool.Spec.NodeSelector, other.Spec.NodeSelector) {
			warnings = append(warnings, fmt.Sprintf(
				"pools %q and %q have identical nodeSelector; multiple virt-handler DaemonSets may target the same nodes",
				pool.Name, other.Name))
		} else if isSubset(pool.Spec.NodeSelector, other.Spec.NodeSelector) || isSubset(other.Spec.NodeSelector, pool.Spec.NodeSelector) {
			warnings = append(warnings, fmt.Sprintf(
				"pools %q and %q have overlapping nodeSelector; multiple virt-handler DaemonSets may target the same nodes",
				pool.Name, other.Name))
		}
	}
	return warnings
}
