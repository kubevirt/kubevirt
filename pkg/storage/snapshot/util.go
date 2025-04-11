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

package snapshot

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/client-go/kubecli"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
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

func hasConditionType(conditions []snapshotv1.Condition, condType snapshotv1.ConditionType) bool {
	for _, cond := range conditions {
		if cond.Type == condType {
			return true
		}
	}
	return false
}

func updateCondition(conditions []snapshotv1.Condition, c snapshotv1.Condition) []snapshotv1.Condition {
	found := false
	for i := range conditions {
		if conditions[i].Type == c.Type {
			if conditions[i].Status != c.Status || conditions[i].Reason != c.Reason {
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

func getSimplifiedMetaObject(meta metav1.ObjectMeta) *metav1.ObjectMeta {
	result := meta.DeepCopy()
	result.ManagedFields = nil

	return result
}

func GetSnapshotContents(vmSnapshot *snapshotv1.VirtualMachineSnapshot, client kubecli.KubevirtClient) (*snapshotv1.VirtualMachineSnapshotContent, error) {
	if vmSnapshot == nil {
		return nil, fmt.Errorf("VirtualMachineSnapshot is nil")
	}

	if vmSnapshot.Status == nil || vmSnapshot.Status.VirtualMachineSnapshotContentName == nil {
		return nil, fmt.Errorf("VirtualMachineSnapshot %s has nil contents name", vmSnapshot.Name)
	}

	vmSnapshotContentName := *vmSnapshot.Status.VirtualMachineSnapshotContentName

	return client.VirtualMachineSnapshotContent(vmSnapshot.Namespace).Get(context.Background(), vmSnapshotContentName, metav1.GetOptions{})
}
