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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package virt_operator

import (
	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	operatorv1helpers "github.com/openshift/library-go/pkg/operator/v1helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

func syncClusterOperator(namespace string, version string, done bool) error {
	name := "virt-operator"

	cfgv1client, err := getCfgV1Client()
	if err != nil {
		return err
	}

	co, err := cfgv1client.ClusterOperators().Get(name, metav1.GetOptions{})
	if err != nil {
		// Not found - create the resource
		if errors.IsNotFound(err) {
			co := &configv1.ClusterOperator{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterOperator",
					APIVersion: "config.openshift.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Status: configv1.ClusterOperatorStatus{
					RelatedObjects: []configv1.ObjectReference{
						{
							Group:    "",
							Resource: "namespaces",
							Name:     namespace,
						},
					},
				},
			}

			co, err = cfgv1client.ClusterOperators().Create(co)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	prevConditions := co.Status.Conditions
	co.Status.Conditions = updateConditions(prevConditions, done)

	operatorv1helpers.SetOperandVersion(&co.Status.Versions, configv1.OperandVersion{Name: "operator", Version: version})

	if conditionsEquals(co.Status.Conditions, prevConditions) {
		return nil
	}

	_, err = cfgv1client.ClusterOperators().UpdateStatus(co)
	if err != nil {
		return err
	}

	return nil
}

func updateConditions(conditions []configv1.ClusterOperatorStatusCondition, done bool) []configv1.ClusterOperatorStatusCondition {
	// FIXME: switch from deprecated Failing to newer Degraded
	// FIXME: actually implement the expected semantics of these conditions
	conditions = []configv1.ClusterOperatorStatusCondition{
		{
			Type:   configv1.OperatorAvailable,
			Status: configv1.ConditionFalse,
		}, {
			Type:   configv1.OperatorProgressing,
			Status: configv1.ConditionFalse,
		}, {
			Type:   configv1.OperatorFailing,
			Status: configv1.ConditionFalse,
		},
	}
	if done {
		conditions[0].Status = configv1.ConditionTrue
	} else {
		conditions[1].Status = configv1.ConditionTrue
	}
	return conditions
}

func conditionsEquals(conditions []configv1.ClusterOperatorStatusCondition, prev []configv1.ClusterOperatorStatusCondition) bool {
	// FIXME: only update when something has changed
	return false
}

func getCfgV1Client() (*configv1client.ConfigV1Client, error) {
	c, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return nil, err
	}

	cfgv1client, err := configv1client.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return cfgv1client, nil
}
