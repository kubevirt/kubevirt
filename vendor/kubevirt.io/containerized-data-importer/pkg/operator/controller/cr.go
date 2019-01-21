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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

var (
	conditionReady = cdiv1alpha1.CDICondition{
		Type:    cdiv1alpha1.CDIConditionRunning,
		Status:  corev1.ConditionTrue,
		Reason:  "All deployments running and ready",
		Message: "Have fun!",
	}
)

func (r *ReconcileCDI) crInit(cr *cdiv1alpha1.CDI) error {
	cr.Finalizers = append(cr.Finalizers, finalizerName)
	cr.Status.OperatorVersion = r.namespacedArgs.DockerTag
	cr.Status.TargetVersion = r.namespacedArgs.DockerTag
	if err := r.crUpdate(cdiv1alpha1.CDIPhaseDeploying, cr); err != nil {
		return err
	}
	return nil
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

func (r *ReconcileCDI) conditionUpdate(condition cdiv1alpha1.CDICondition, cr *cdiv1alpha1.CDI) error {
	condition.LastProbeTime = metav1.Time{Time: time.Now()}
	condition.LastTransitionTime = condition.LastProbeTime

	i := -1
	for j, c := range cr.Status.Conditions {
		if c.Type == condition.Type {
			i = j
			break
		}
	}

	if i >= 0 {
		c := cr.Status.Conditions[i]
		c.LastProbeTime = condition.LastProbeTime
		c.LastTransitionTime = condition.LastTransitionTime

		if c == condition {
			return nil
		}

		cr.Status.Conditions[i] = condition

	} else {
		cr.Status.Conditions = append(cr.Status.Conditions, condition)
	}

	return r.crUpdate(cr.Status.Phase, cr)
}

func (r *ReconcileCDI) conditionRemove(conditionType cdiv1alpha1.CDIConditionType, cr *cdiv1alpha1.CDI) error {
	i := -1
	for j, c := range cr.Status.Conditions {
		if conditionType == c.Type {
			i = j
			break
		}
	}

	if i >= 0 {
		cr.Status.Conditions = append(cr.Status.Conditions[:i], cr.Status.Conditions[i+1:]...)

		return r.crUpdate(cr.Status.Phase, cr)
	}

	return nil
}
