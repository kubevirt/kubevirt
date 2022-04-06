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
	"fmt"
	"time"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/client-go/discovery"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/version"
)

const (
	KubeVirtFinalizer string = "foregroundDeleteKubeVirt"

	ConditionReasonDeploymentFailedExisting = "ExistingDeployment"
	ConditionReasonDeploymentFailedError    = "DeploymentFailed"
	ConditionReasonDeletionFailedError      = "DeletionFailed"
	ConditionReasonDeploymentCreated        = "AllResourcesCreated"
	ConditionReasonDeploymentReady          = "AllComponentsReady"
	ConditionReasonDeploying                = "DeploymentInProgress"
	ConditionReasonUpdating                 = "UpdateInProgress"
	ConditionReasonDeleting                 = "DeletionInProgress"
)

func UpdateConditionsDeploying(kv *virtv1.KubeVirt) {
	removeCondition(kv, virtv1.KubeVirtConditionSynchronized)
	msg := fmt.Sprintf("Deploying version %s with registry %s",
		kv.Status.TargetKubeVirtVersion,
		kv.Status.TargetKubeVirtRegistry)
	updateCondition(kv, virtv1.KubeVirtConditionAvailable, k8sv1.ConditionFalse, ConditionReasonDeploying, msg)
	updateCondition(kv, virtv1.KubeVirtConditionProgressing, k8sv1.ConditionTrue, ConditionReasonDeploying, msg)
	updateCondition(kv, virtv1.KubeVirtConditionDegraded, k8sv1.ConditionFalse, ConditionReasonDeploying, msg)
}

func UpdateConditionsUpdating(kv *virtv1.KubeVirt) {
	removeCondition(kv, virtv1.KubeVirtConditionCreated)
	removeCondition(kv, virtv1.KubeVirtConditionSynchronized)
	msg := fmt.Sprintf("Transitioning from previous version %s with registry %s to target version %s using registry %s",
		kv.Status.ObservedKubeVirtVersion,
		kv.Status.ObservedKubeVirtRegistry,
		kv.Status.TargetKubeVirtVersion,
		kv.Status.TargetKubeVirtRegistry)
	updateCondition(kv, virtv1.KubeVirtConditionAvailable, k8sv1.ConditionTrue, ConditionReasonUpdating, msg)
	updateCondition(kv, virtv1.KubeVirtConditionProgressing, k8sv1.ConditionTrue, ConditionReasonUpdating, msg)
	updateCondition(kv, virtv1.KubeVirtConditionDegraded, k8sv1.ConditionTrue, ConditionReasonUpdating, msg)
}

func UpdateConditionsCreated(kv *virtv1.KubeVirt) {
	updateCondition(kv, virtv1.KubeVirtConditionCreated, k8sv1.ConditionTrue, ConditionReasonDeploymentCreated, "All resources were created.")
}

func UpdateConditionsAvailable(kv *virtv1.KubeVirt) {
	msg := "All components are ready."
	updateCondition(kv, virtv1.KubeVirtConditionAvailable, k8sv1.ConditionTrue, ConditionReasonDeploymentReady, msg)
	updateCondition(kv, virtv1.KubeVirtConditionProgressing, k8sv1.ConditionFalse, ConditionReasonDeploymentReady, msg)
	updateCondition(kv, virtv1.KubeVirtConditionDegraded, k8sv1.ConditionFalse, ConditionReasonDeploymentReady, msg)
}

func UpdateConditionsFailedExists(kv *virtv1.KubeVirt) {
	updateCondition(kv, virtv1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedExisting, "There is an active KubeVirt deployment")
	// don' t set any other conditions here, so HCO just ignores this KubeVirt CR
}

func UpdateConditionsFailedError(kv *virtv1.KubeVirt, err error) {
	msg := fmt.Sprintf("An error occurred during deployment: %v", err)
	updateCondition(kv, virtv1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedError, msg)
	updateCondition(kv, virtv1.KubeVirtConditionAvailable, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedError, msg)
	updateCondition(kv, virtv1.KubeVirtConditionProgressing, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedError, msg)
	updateCondition(kv, virtv1.KubeVirtConditionDegraded, k8sv1.ConditionTrue, ConditionReasonDeploymentFailedError, msg)
}

func UpdateConditionsDeleting(kv *virtv1.KubeVirt) {
	removeCondition(kv, virtv1.KubeVirtConditionCreated)
	removeCondition(kv, virtv1.KubeVirtConditionSynchronized)
	msg := fmt.Sprintf("Deletion was triggered")
	updateCondition(kv, virtv1.KubeVirtConditionAvailable, k8sv1.ConditionFalse, ConditionReasonDeleting, msg)
	updateCondition(kv, virtv1.KubeVirtConditionProgressing, k8sv1.ConditionFalse, ConditionReasonDeleting, msg)
	updateCondition(kv, virtv1.KubeVirtConditionDegraded, k8sv1.ConditionTrue, ConditionReasonDeleting, msg)
}

func UpdateConditionsDeletionFailed(kv *virtv1.KubeVirt, err error) {
	updateCondition(kv, virtv1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeletionFailedError, fmt.Sprintf("An error occurred during deletion: %v", err))
}

func updateCondition(kv *virtv1.KubeVirt, conditionType virtv1.KubeVirtConditionType, status k8sv1.ConditionStatus, reason string, message string) {
	condition, isNew := getCondition(kv, conditionType)
	condition.Status = status
	condition.Reason = reason
	condition.Message = message

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

func removeCondition(kv *virtv1.KubeVirt, conditionType virtv1.KubeVirtConditionType) {
	conditions := kv.Status.Conditions
	for i, condition := range conditions {
		if condition.Type == conditionType {
			conditions = append(conditions[:i], conditions[i+1:]...)
			kv.Status.Conditions = conditions
			return
		}
	}
}

func SetConditionTimestamps(kvOrig *virtv1.KubeVirt, kvUpdated *virtv1.KubeVirt) {
	now := metav1.Time{
		Time: time.Now(),
	}
	for i, c := range kvUpdated.Status.Conditions {
		if cOrig, created := getCondition(kvOrig, c.Type); !created {
			// check if condition was updated
			if cOrig.Status != c.Status ||
				cOrig.Reason != c.Reason ||
				cOrig.Message != c.Message {
				kvUpdated.Status.Conditions[i].LastProbeTime = now
				kvUpdated.Status.Conditions[i].LastTransitionTime = now
			}
			// do not update lastProbeTime only, will result in too many updates
		} else {
			// condition is new
			kvUpdated.Status.Conditions[i].LastProbeTime = now
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

func IsServiceMonitorEnabled(clientset kubecli.KubevirtClient) (bool, error) {
	_, apis, err := clientset.DiscoveryClient().ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return false, err
	}

	for _, api := range apis {
		if api.GroupVersion == promv1.SchemeGroupVersion.String() {
			for _, resource := range api.APIResources {
				if resource.Name == "servicemonitors" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// IsPrometheusRuleEnabled returns true if prometheusrules cr is defined
// and false otherwise.
func IsPrometheusRuleEnabled(clientset kubecli.KubevirtClient) (bool, error) {
	_, apis, err := clientset.DiscoveryClient().ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return false, err
	}

	for _, api := range apis {
		if api.GroupVersion == promv1.SchemeGroupVersion.String() {
			for _, resource := range api.APIResources {
				if resource.Name == "prometheusrules" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
