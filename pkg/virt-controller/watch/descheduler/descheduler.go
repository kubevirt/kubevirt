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

package descheduler

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

// EvictOnlyAnnotation indicates pods whose eviction is not expected to be completed right away.
// Instead, an eviction request is expected to be intercepted by an external component which will initiate the
// eviction process for the pod.
const EvictOnlyAnnotation = "descheduler.alpha.kubernetes.io/request-evict-only"

// EvictionInProgressAnnotation indicates pods whose eviction was initiated by an external component.
const EvictionInProgressAnnotation = "descheduler.alpha.kubernetes.io/eviction-in-progress"

// EvictPodAnnotationKeyAlpha can be used to explicitly opt-in a pod to be eventually descheduled.
// The descheduler will only check the presence of the annotation and not its value.
const EvictPodAnnotationKeyAlpha = "descheduler.alpha.kubernetes.io/evict"

// EvictPodAnnotationKeyAlphaPreferNoEviction can be used to explicitly opt-out a pod to be eventually descheduled.
// The descheduler will only check the presence of the annotation and not its value.
const EvictPodAnnotationKeyAlphaPreferNoEviction = "descheduler.alpha.kubernetes.io/prefer-no-eviction"

func MarkEvictionInProgress(virtClient kubecli.KubevirtClient, sourcePod *k8sv1.Pod) (*k8sv1.Pod, error) {
	if _, exists := sourcePod.GetAnnotations()[EvictionInProgressAnnotation]; exists {
		return sourcePod, nil
	}

	patchSet := patch.New(
		patch.WithAdd(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(EvictionInProgressAnnotation)), "true"),
	)
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return nil, err
	}

	pod, err := virtClient.CoreV1().Pods(sourcePod.Namespace).Patch(context.Background(), sourcePod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		log.Log.Object(sourcePod).Errorf("failed to add %s pod annotation: %v", EvictionInProgressAnnotation, err)
		return nil, err
	}

	return pod, nil
}

func MarkEvictionCompleted(virtClient kubecli.KubevirtClient, sourcePod *k8sv1.Pod) (*k8sv1.Pod, error) {

	if value, exists := sourcePod.GetAnnotations()[EvictionInProgressAnnotation]; exists {
		patchSet := patch.New(
			patch.WithTest(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(EvictionInProgressAnnotation)), value),
			patch.WithRemove(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(EvictionInProgressAnnotation))),
		)
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return nil, err
		}

		pod, err := virtClient.CoreV1().Pods(sourcePod.Namespace).Patch(context.Background(), sourcePod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
		if err != nil {
			log.Log.Object(sourcePod).Errorf("failed to remove %s pod annotation : %v", EvictionInProgressAnnotation, err)
			return nil, err
		}
		return pod, nil
	}

	return sourcePod, nil
}
