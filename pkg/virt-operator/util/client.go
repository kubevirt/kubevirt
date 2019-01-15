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

	"kubevirt.io/kubevirt/pkg/log"

	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func SetVersions(kv *virtv1.KubeVirt, config KubeVirtDeploymentConfig) {

	kv.Status.OperatorVersion = version.Get().String()

	// Note: for now we just set targetKubeVirtVersion and observedKubeVirtVersion to the tag of the operator image
	// In future this needs some more work...
	kv.Status.TargetKubeVirtVersion = config.ImageTag
	kv.Status.ObservedKubeVirtVersion = config.ImageTag

}

func UpdateScc(clientset kubecli.KubevirtClient, kv *virtv1.KubeVirt, add bool) error {

	secClient := clientset.SecClient()

	privScc, err := secClient.SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// we are mot on openshift?
			log.Log.V(4).Infof("unable to get scc, we are probably not on openshift: %v", err)
			return nil
		} else {
			return fmt.Errorf("unable to get scc: %v", err)
		}
	}

	var kubeVirtAccounts []string
	prefix := "system:serviceaccount"
	kubeVirtAccounts = append(kubeVirtAccounts, fmt.Sprintf("%s:%s:%s", prefix, kv.Namespace, "kubevirt-privileged"))
	kubeVirtAccounts = append(kubeVirtAccounts, fmt.Sprintf("%s:%s:%s", prefix, kv.Namespace, "kubevirt-apiserver"))
	kubeVirtAccounts = append(kubeVirtAccounts, fmt.Sprintf("%s:%s:%s", prefix, kv.Namespace, "kubevirt-controller"))

	modified := false
	users := privScc.Users
	for _, acc := range kubeVirtAccounts {
		if add {
			if !contains(users, acc) {
				users = append(users, acc)
				modified = true
			}
		} else {
			removed := false
			users, removed = remove(users, acc)
			modified = modified || removed
		}
	}
	if modified {
		privScc.Users = users
		_, err = secClient.SecurityContextConstraints().Update(privScc)
		if err != nil {
			return fmt.Errorf("unable to update scc: %v", err)
		}
	}

	return nil
}

func contains(users []string, user string) bool {
	for _, u := range users {
		if u == user {
			return true
		}
	}
	return false
}

func remove(users []string, user string) ([]string, bool) {
	var newUsers []string
	modified := false
	for _, u := range users {
		if u != user {
			newUsers = append(newUsers, u)
		} else {
			modified = true
		}
	}
	return newUsers, modified
}
