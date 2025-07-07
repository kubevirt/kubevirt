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
 * Copyright The KubeVirt Authors
 *
 */

package descheduler

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
)

// EvictOnlyAnnotation indicates pods whose eviction is not expected to be completed right away.
// Instead, an eviction request is expected to be intercepted by an external component which will initiate the
// eviction process for the pod.
const EvictOnlyAnnotation = "descheduler.alpha.kubernetes.io/request-evict-only"

// EvictionInProgressAnnotation indicates pods whose eviction was initiated by an external component.
const EvictionInProgressAnnotation = "descheduler.alpha.kubernetes.io/eviction-in-progress"

func MarkEvictionInProgress(virtClient kubecli.KubevirtClient, sourcePod *k8sv1.Pod) error {
	if _, exists := sourcePod.GetAnnotations()[EvictionInProgressAnnotation]; exists {
		return nil
	}

	patchSet := patch.New(
		patch.WithAdd(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(EvictionInProgressAnnotation)), "kubevirt"),
	)
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = virtClient.CoreV1().Pods(sourcePod.Namespace).Patch(context.Background(), sourcePod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		log.Log.Object(sourcePod).Errorf("failed to add %s pod annotation: %v", EvictionInProgressAnnotation, err)
		return err
	}

	return nil
}

func MarkSourcePodEvictionCompleted(virtClient kubecli.KubevirtClient, migration *virtv1.VirtualMachineInstanceMigration, podIndexer cache.Indexer) error {
	if migration.Status.MigrationState == nil || migration.Status.MigrationState.SourcePod == "" {
		return nil
	}

	podKey := controller.NamespacedKey(migration.Namespace, migration.Status.MigrationState.SourcePod)
	obj, exists, err := podIndexer.GetByKey(podKey)
	if !exists {
		log.Log.Warningf("source pod %s does not exist", migration.Status.MigrationState.SourcePod)
		return nil
	}
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to fetch source pod %s for namespace from cache.", migration.Status.MigrationState.SourcePod)
		return err
	}

	sourcePod := obj.(*k8sv1.Pod)
	if _, exists := sourcePod.GetAnnotations()[EvictionInProgressAnnotation]; exists {
		patchSet := patch.New(
			patch.WithTest(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(EvictionInProgressAnnotation)), "kubevirt"),
			patch.WithRemove(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(EvictionInProgressAnnotation))),
		)
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return err
		}

		_, err = virtClient.CoreV1().Pods(sourcePod.Namespace).Patch(context.Background(), sourcePod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
		if err != nil {
			log.Log.Object(sourcePod).Errorf("failed to remove %s pod annotation : %v", EvictionInProgressAnnotation, err)
			return err
		}
	}

	return nil
}
