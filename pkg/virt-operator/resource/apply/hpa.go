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

package apply

import (
	"context"
	"fmt"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

func (r *Reconciler) syncExportProxyHorizontalPodAutoscaler(deployment *appsv1.Deployment, desiredReplicas int32) error {
	// Skip HPA when the caller decided only one replica should run (same heuristic as virt-api).
	if desiredReplicas <= 1 {
		// Only Namespace/Name are needed for cache lookup and delete; avoid
		// probing custom.metrics.k8s.io on single-node clusters.
		lookupHPA := &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: deployment.Namespace,
				Name:      components.VirtExportProxyHPAName,
			},
		}
		obj, exists, _ := r.stores.HorizontalPodAutoscalerCache.Get(lookupHPA)
		if !exists {
			return nil
		}

		cachedHorizontalPodAutoscaler := obj.(*autoscalingv2.HorizontalPodAutoscaler)
		key, err := controller.KeyFunc(cachedHorizontalPodAutoscaler)
		if err != nil {
			return err
		}

		r.expectations.HorizontalPodAutoscaler.AddExpectedDeletion(r.kvKey, key)
		hpaClient := r.k8sClient.AutoscalingV2().HorizontalPodAutoscalers(deployment.Namespace)
		if err := hpaClient.Delete(context.Background(), lookupHPA.Name, metav1.DeleteOptions{}); err != nil {
			r.expectations.HorizontalPodAutoscaler.DeletionObserved(r.kvKey, key)
			return fmt.Errorf("unable to delete horizontalpodautoscaler %s: %w", lookupHPA.Name, err)
		}

		return nil
	}

	profile := r.resolveExportProxyHPAMetricsProfile(deployment.Namespace)
	horizontalPodAutoscaler := components.NewExportProxyHorizontalPodAutoscaler(deployment.Namespace, profile)
	return r.syncHorizontalPodAutoscaler(deployment.Namespace, horizontalPodAutoscaler)
}

func (r *Reconciler) syncHorizontalPodAutoscaler(namespace string, horizontalPodAutoscaler *autoscalingv2.HorizontalPodAutoscaler) error {
	kv := r.kv

	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)
	injectOperatorMetadata(kv, &horizontalPodAutoscaler.ObjectMeta, imageTag, imageRegistry, id, true)

	hpaClient := r.k8sClient.AutoscalingV2().HorizontalPodAutoscalers(namespace)

	obj, exists, _ := r.stores.HorizontalPodAutoscalerCache.Get(horizontalPodAutoscaler)
	if !exists {
		r.expectations.HorizontalPodAutoscaler.RaiseExpectations(r.kvKey, 1, 0)
		origHPA := horizontalPodAutoscaler
		horizontalPodAutoscaler, err := hpaClient.Create(context.Background(), horizontalPodAutoscaler, metav1.CreateOptions{})
		if err != nil {
			r.expectations.HorizontalPodAutoscaler.LowerExpectations(r.kvKey, 1, 0)
			log.Log.V(2).Infof("failed to create horizontalpodautoscaler %s: %+v", origHPA.Name, origHPA)
			return fmt.Errorf("unable to create horizontalpodautoscaler %s: %w", origHPA.Name, err)
		}
		log.Log.V(2).Infof("horizontalpodautoscaler %v created", horizontalPodAutoscaler.GetName())
		SetGeneration(&kv.Status.Generations, horizontalPodAutoscaler)

		return nil
	}

	cachedHorizontalPodAutoscaler := obj.(*autoscalingv2.HorizontalPodAutoscaler)
	modified := pointer.P(false)
	existingCopy := cachedHorizontalPodAutoscaler.DeepCopy()
	expectedGeneration := GetExpectedGeneration(horizontalPodAutoscaler, kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, horizontalPodAutoscaler.ObjectMeta)
	if !*modified &&
		equality.Semantic.DeepEqual(existingCopy.Spec, horizontalPodAutoscaler.Spec) &&
		existingCopy.ObjectMeta.Generation == expectedGeneration {
		log.Log.V(4).Infof("horizontalpodautoscaler %v is up-to-date", cachedHorizontalPodAutoscaler.GetName())
		return nil
	}

	patchBytes, err := patch.New(getPatchWithObjectMetaAndSpec([]patch.PatchOption{}, &horizontalPodAutoscaler.ObjectMeta, horizontalPodAutoscaler.Spec)...).GeneratePayload()
	if err != nil {
		return err
	}

	prePatchHPA := horizontalPodAutoscaler
	horizontalPodAutoscaler, err = hpaClient.Patch(context.Background(), horizontalPodAutoscaler.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		log.Log.V(2).Infof("failed to patch horizontalpodautoscaler %s: %+v", prePatchHPA.Name, prePatchHPA)
		return fmt.Errorf("unable to patch horizontalpodautoscaler %s: %w", prePatchHPA.Name, err)
	}

	SetGeneration(&kv.Status.Generations, horizontalPodAutoscaler)
	log.Log.V(2).Infof("horizontalpodautoscaler %v patched", horizontalPodAutoscaler.GetName())

	return nil
}
