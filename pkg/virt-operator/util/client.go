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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package util

import (
	"time"

	secv1 "github.com/openshift/api/security/v1"
	"k8s.io/client-go/discovery"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/version"
)

const (
	KubeVirtFinalizer string = "foregroundDeleteKubeVirt"
)

func UpdateCondition(kv *virtv1.KubeVirt, conditionType virtv1.KubeVirtConditionType, status k8sv1.ConditionStatus, reason string, message string) {

	condition, isNew := getCondition(kv, conditionType)
	transition := false
	if !isNew && (condition.Status != status || condition.Reason != reason || condition.Message != message) {
		transition = true
	}

	condition.Status = status
	condition.Reason = reason
	condition.Message = message

	now := time.Now()
	if isNew || transition {
		condition.LastProbeTime = metav1.Time{
			Time: now,
		}
	}
	if transition {
		condition.LastTransitionTime = metav1.Time{
			Time: now,
		}
	}

	conditions := kv.Status.Conditions
	if isNew {
		conditions = append(conditions, *condition)
	} else {
		for i := range conditions {
			if conditions[i].Type == conditionType {
				conditions[i] = *condition
				break
			}
		}
	}

	kv.Status.Conditions = conditions

}

func getCondition(kv *virtv1.KubeVirt, conditionType virtv1.KubeVirtConditionType) (*virtv1.KubeVirtCondition, bool) {
	for _, condition := range kv.Status.Conditions {
		if condition.Type == conditionType {
			return &condition, false
		}
	}
	condition := &virtv1.KubeVirtCondition{
		Type: conditionType,
	}
	return condition, true
}

func RemoveCondition(kv *virtv1.KubeVirt, conditionType virtv1.KubeVirtConditionType) {
	conditions := kv.Status.Conditions
	for i, condition := range conditions {
		if condition.Type == conditionType {
			conditions = append(conditions[:i], conditions[i+1:]...)
			kv.Status.Conditions = conditions
			return
		}
	}
}

func AddFinalizer(kv *virtv1.KubeVirt) {
	if !hasFinalizer(kv) {
		kv.Finalizers = append(kv.Finalizers, KubeVirtFinalizer)
	}
}

func hasFinalizer(kv *virtv1.KubeVirt) bool {
	for _, f := range kv.GetFinalizers() {
		if f == KubeVirtFinalizer {
			return true
		}
	}
	return false
}

func SetOperatorVersion(kv *virtv1.KubeVirt) {
	kv.Status.OperatorVersion = version.Get().String()
}

func IsOnOpenshift(clientset kubecli.KubevirtClient) (bool, error) {

	apis, err := clientset.DiscoveryClient().ServerResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return false, err
	}

	// In case of an error, check if security.openshift.io is the reason (unlikely).
	// If it is, we are obviously on an openshift cluster.
	// Otherwise we can do a positive check.
	if discovery.IsGroupDiscoveryFailedError(err) {
		e := err.(*discovery.ErrGroupDiscoveryFailed)
		if _, exists := e.Groups[secv1.GroupVersion]; exists {
			return true, nil
		}
	}

	for _, api := range apis {
		if api.GroupVersion == secv1.GroupVersion.String() {
			for _, resource := range api.APIResources {
				if resource.Name == "securitycontextconstraints" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
