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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/version"
)

const (
	KubeVirtFinalizer string = "foregroundDeleteKubeVirt"
)

func UpdatePhase(kv *v1.KubeVirt, phase v1.KubeVirtPhase, clientset kubecli.KubevirtClient) error {
	var err error
	if kv.Status.Phase != phase {
		patchStr := fmt.Sprintf(`{"status":{"phase":"%s"}}`, phase)
		kv, err = clientset.KubeVirt(kv.Namespace).Patch(kv.Name, types.MergePatchType, []byte(patchStr))
	}
	return err
}

func UpdateCondition(kv *v1.KubeVirt, conditionType v1.KubeVirtConditionType, status k8sv1.ConditionStatus, reason string, message string, clientset kubecli.KubevirtClient) error {

	condition, isNew := getCondition(kv, conditionType)
	transition := false
	if !isNew && (condition.Status != status || condition.Reason != reason || condition.Message != message) {
		transition = true
	}

	condition.Status = status
	condition.Reason = reason
	condition.Message = message
	now := time.Now()
	condition.LastProbeTime = metav1.Time{
		Time: now,
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

	var condJson string
	bytes, err := json.Marshal(conditions)
	if err != nil {
		return err
	}
	condJson = string(bytes)

	patchStr := fmt.Sprintf(`{"status":{"conditions":%s}}`, condJson)
	_, err = clientset.KubeVirt(kv.Namespace).Patch(kv.Name, types.MergePatchType, []byte(patchStr))
	return err
}

func getCondition(kv *v1.KubeVirt, conditionType v1.KubeVirtConditionType) (*v1.KubeVirtCondition, bool) {
	for _, condition := range kv.Status.Conditions {
		if condition.Type == conditionType {
			return &condition, false
		}
	}
	condition := &v1.KubeVirtCondition{
		Type: conditionType,
	}
	return condition, true
}

func RemoveConditions(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) error {
	var conditions []struct{}
	var condJson string
	bytes, err := json.Marshal(conditions)
	if err != nil {
		return err
	}
	condJson = string(bytes)

	patchStr := fmt.Sprintf(`{"status":{"conditions":%s}}`, condJson)
	_, err = clientset.KubeVirt(kv.Namespace).Patch(kv.Name, types.MergePatchType, []byte(patchStr))
	return err
}

func AddFinalizer(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) error {
	if !HasFinalizer(kv) {
		kv.Finalizers = append(kv.Finalizers, KubeVirtFinalizer)
		return patchFinalizer(kv, clientset)
	}
	return nil
}

func RemoveFinalizer(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) error {
	kv.SetFinalizers([]string{})
	return patchFinalizer(kv, clientset)
}

func HasFinalizer(kv *v1.KubeVirt) bool {
	for _, f := range kv.GetFinalizers() {
		if f == KubeVirtFinalizer {
			return true
		}
	}
	return false
}

func patchFinalizer(kv *v1.KubeVirt, clientset kubecli.KubevirtClient) error {
	var finalizers string
	bytes, err := json.Marshal(kv.Finalizers)
	if err != nil {
		return err
	}
	finalizers = string(bytes)
	patchStr := fmt.Sprintf(`{"metadata":{"finalizers":%s}}`, finalizers)
	kv, err = clientset.KubeVirt(kv.Namespace).Patch(kv.Name, types.MergePatchType, []byte(patchStr))
	return err
}

func SetVersions(kv *v1.KubeVirt, config KubeVirtDeploymentConfig, clientset kubecli.KubevirtClient) error {
	// Note: for now we just set targetKubeVirtVersion and observedKubeVirtVersion to the tag of the operator image
	// In future this needs some more work...
	patchStr := fmt.Sprintf(`{"status":{"operatorVersion":"%s", "targetKubeVirtVersion":"%s", "observedKubeVirtVersion":"%s"}}`,
		version.Get().String(), config.ImageTag, config.ImageTag)
	kv, err := clientset.KubeVirt(kv.Namespace).Patch(kv.Name, types.MergePatchType, []byte(patchStr))
	return err
}
