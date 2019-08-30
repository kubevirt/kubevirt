/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	conditions "github.com/openshift/custom-resource-status/conditions/v1"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

func (r *ReconcileCDI) isUpgrading(cr *cdiv1alpha1.CDI) bool {
	return cr.Status.ObservedVersion != "" && cr.Status.ObservedVersion != cr.Status.TargetVersion
}

// this is used for testing.  wish this a helper function in test file instead of member
func (r *ReconcileCDI) crSetVersion(cr *cdiv1alpha1.CDI, version, repo string) error {
	phase := cdiv1alpha1.CDIPhaseDeployed
	if version == "" {
		phase = cdiv1alpha1.CDIPhase("")
	}
	cr.Spec.ImageTag = version
	cr.Spec.ImageRegistry = repo
	cr.Status.ObservedVersion = version
	cr.Status.OperatorVersion = version
	cr.Status.TargetVersion = version
	return r.crUpdate(phase, cr)
}

func (r *ReconcileCDI) crInit(cr *cdiv1alpha1.CDI) error {
	cr.Finalizers = append(cr.Finalizers, finalizerName)
	cr.Status.OperatorVersion = r.namespacedArgs.DockerTag
	cr.Status.TargetVersion = r.namespacedArgs.DockerTag
	return r.crUpdate(cdiv1alpha1.CDIPhaseDeploying, cr)
}

func (r *ReconcileCDI) crError(cr *cdiv1alpha1.CDI) error {
	if cr.Status.Phase != cdiv1alpha1.CDIPhaseError {
		return r.crUpdate(cdiv1alpha1.CDIPhaseError, cr)
	}
	return nil
}

func (r *ReconcileCDI) crUpdate(phase cdiv1alpha1.CDIPhase, cr *cdiv1alpha1.CDI) error {
	cr.Status.Phase = phase
	return r.client.Update(context.TODO(), cr)
}

// GetConditionValues gets the conditions and put them into a map for easy comparison
func GetConditionValues(conditionList []conditions.Condition) map[conditions.ConditionType]corev1.ConditionStatus {
	result := make(map[conditions.ConditionType]corev1.ConditionStatus)
	for _, cond := range conditionList {
		result[cond.Type] = cond.Status
	}
	return result
}

// Compare condition maps and return true if any of the conditions changed, false otherwise.
func conditionsChanged(originalValues, newValues map[conditions.ConditionType]corev1.ConditionStatus) bool {
	if len(originalValues) != len(newValues) {
		return true
	}
	for k, v := range newValues {
		oldV, ok := originalValues[k]
		if !ok || oldV != v {
			return true
		}
	}
	return false
}

// MarkCrHealthyMessage marks the passed in CR as healthy. The CR object needs to be updated by the caller afterwards.
// Healthy means the following status conditions are set:
// ApplicationAvailable: true
// Progressing: false
// Degraded: false
func MarkCrHealthyMessage(cr *cdiv1alpha1.CDI, reason, message string) {
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:    conditions.ConditionAvailable,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionProgressing,
		Status: corev1.ConditionFalse,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionDegraded,
		Status: corev1.ConditionFalse,
	})
}

// MarkCrUpgradeHealingDegraded marks the passed CR as upgrading and degraded. The CR object needs to be updated by the caller afterwards.
// Failed means the following status conditions are set:
// ApplicationAvailable: true
// Progressing: true
// Degraded: true
func MarkCrUpgradeHealingDegraded(cr *cdiv1alpha1.CDI, reason, message string) {
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionAvailable,
		Status: corev1.ConditionTrue,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionProgressing,
		Status: corev1.ConditionTrue,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:    conditions.ConditionDegraded,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

// MarkCrFailed marks the passed CR as failed and requiring human intervention. The CR object needs to be updated by the caller afterwards.
// Failed means the following status conditions are set:
// ApplicationAvailable: false
// Progressing: false
// Degraded: true
func MarkCrFailed(cr *cdiv1alpha1.CDI, reason, message string) {
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionAvailable,
		Status: corev1.ConditionFalse,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionProgressing,
		Status: corev1.ConditionFalse,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:    conditions.ConditionDegraded,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

// MarkCrFailedHealing marks the passed CR as failed and healing. The CR object needs to be updated by the caller afterwards.
// FailedAndHealing means the following status conditions are set:
// ApplicationAvailable: false
// Progressing: true
// Degraded: true
func MarkCrFailedHealing(cr *cdiv1alpha1.CDI, reason, message string) {
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionAvailable,
		Status: corev1.ConditionFalse,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionProgressing,
		Status: corev1.ConditionTrue,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:    conditions.ConditionDegraded,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

// MarkCrDeploying marks the passed CR as currently deploying. The CR object needs to be updated by the caller afterwards.
// Deploying means the following status conditions are set:
// ApplicationAvailable: false
// Progressing: true
// Degraded: false
func MarkCrDeploying(cr *cdiv1alpha1.CDI, reason, message string) {
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionAvailable,
		Status: corev1.ConditionFalse,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:    conditions.ConditionProgressing,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
	conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
		Type:   conditions.ConditionDegraded,
		Status: corev1.ConditionFalse,
	})
}
