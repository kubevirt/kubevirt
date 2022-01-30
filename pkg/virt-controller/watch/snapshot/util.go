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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package snapshot

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
)

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Log.Infof("%s took %s", name, elapsed)
}

func cacheKeyFunc(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func newReadyCondition(status corev1.ConditionStatus, reason string) snapshotv1.Condition {
	return snapshotv1.Condition{
		Type:               snapshotv1.ConditionReady,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func newProgressingCondition(status corev1.ConditionStatus, reason string) snapshotv1.Condition {
	return snapshotv1.Condition{
		Type:               snapshotv1.ConditionProgressing,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func newFailureCondition(status corev1.ConditionStatus, reason string) snapshotv1.Condition {
	return snapshotv1.Condition{
		Type:               snapshotv1.ConditionFailure,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func updateCondition(conditions []snapshotv1.Condition, c snapshotv1.Condition, includeReason bool) []snapshotv1.Condition {
	found := false
	for i := range conditions {
		if conditions[i].Type == c.Type {
			if conditions[i].Status != c.Status || (includeReason && conditions[i].Reason != c.Reason) {
				conditions[i] = c
			}
			found = true
			break
		}
	}

	if !found {
		conditions = append(conditions, c)
	}

	return conditions
}

func processWorkItem(queue workqueue.RateLimitingInterface, handler func(string) (time.Duration, error)) bool {
	obj, shutdown := queue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if requeueAfter, err := handler(key); requeueAfter > 0 || err != nil {
			if requeueAfter > 0 {
				queue.AddAfter(key, requeueAfter)
			} else {
				queue.AddRateLimited(key)
			}

			return err
		}

		queue.Forget(obj)

		return nil

	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func podsUsingPVCs(podInformer cache.SharedIndexInformer, namespace string, pvcNames sets.String) ([]corev1.Pod, error) {
	var pods []corev1.Pod

	if pvcNames.Len() < 1 {
		return pods, nil
	}

	objs, err := podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil, fmt.Errorf("expected Pod, got %T", obj)
		}

		for _, volume := range pod.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil &&
				pvcNames.Has(volume.PersistentVolumeClaim.ClaimName) {
				pods = append(pods, *pod)
			}
		}
	}

	return pods, nil
}

func getFailureDeadline(vmSnapshot *snapshotv1.VirtualMachineSnapshot) time.Duration {
	failureDeadline := snapshotv1.DefaultFailureDeadline
	if vmSnapshot.Spec.FailureDeadline != nil {
		failureDeadline = vmSnapshot.Spec.FailureDeadline.Duration
	}

	return failureDeadline
}

func timeUntilDeadline(vmSnapshot *snapshotv1.VirtualMachineSnapshot) time.Duration {
	failureDeadline := getFailureDeadline(vmSnapshot)
	// No Deadline set by user
	if failureDeadline == 0 {
		return failureDeadline
	}
	deadline := vmSnapshot.CreationTimestamp.Add(failureDeadline)
	return time.Until(deadline)
}
