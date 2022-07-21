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
package components

import (
	flowcontrol "k8s.io/api/flowcontrol/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
)

const (
	kubevirtFlowSchemaName = "kubevirt-control-plane"
)

func NewKubeVirtFlowSchema() *flowcontrol.FlowSchema {
	kubevirtResources := flowcontrol.ResourcePolicyRule{
		APIGroups: []string{
			virtv1.KubeVirtGroupVersionKind.Group,
		},
		ClusterScope: true,
		Namespaces: []string{
			"*",
		},
		Resources: []string{
			virtv1.VirtualMachineGroupVersionKind.Kind,
			virtv1.VirtualMachineInstanceGroupVersionKind.Kind,
			virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind,
			virtv1.VirtualMachineInstancePresetGroupVersionKind.Kind,
			virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Kind,
		},
		Verbs: []string{
			"*",
		},
	}

	kubevirtSubject := flowcontrol.Subject{
		Kind: flowcontrol.SubjectKindServiceAccount,
		ServiceAccount: &flowcontrol.ServiceAccountSubject{
			Name:      "*",
			Namespace: "kubevirt",
		},
	}
	kubevirtPolicy := flowcontrol.PolicyRulesWithSubjects{
		ResourceRules: []flowcontrol.ResourcePolicyRule{
			kubevirtResources,
		},
		Subjects: []flowcontrol.Subject{
			kubevirtSubject,
		},
	}

	schemaSpec := flowcontrol.FlowSchemaSpec{
		PriorityLevelConfiguration: flowcontrol.PriorityLevelConfigurationReference{
			Name: "workload-low",
		},
		MatchingPrecedence: 1000, // 1000 makes KubeVirt less important than k8s control-plane but more importnant than user traffic
		DistinguisherMethod: &flowcontrol.FlowDistinguisherMethod{
			Type: flowcontrol.FlowDistinguisherMethodByUserType,
		},
		Rules: []flowcontrol.PolicyRulesWithSubjects{
			kubevirtPolicy,
		},
	}

	return &flowcontrol.FlowSchema{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "flowschema/v1beta2",
			Kind:       "FlowSchema",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kubevirtFlowSchemaName,
			Labels: map[string]string{
				virtv1.AppLabel:     kubevirtFlowSchemaName,
				virtv1.AppNameLabel: kubevirtFlowSchemaName,
			},
		},
		Spec: schemaSpec,
	}
}
