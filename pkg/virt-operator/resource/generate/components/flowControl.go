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
	virtv1 "kubevirt.io/api/core/v1"
)

func newKubeVirtFlowSchema() *flowcontrol.FlowSchema {
	return &flowcontrol.FlowSchemaSpec{
		PriorityLevelConfiguration: "workload-low",
		MatchingPrecedence:         1000, // 1000 makes KubeVirt less important than k8s control-plane but more importnant than user traffic
		DistinguisherMethod: &flowcontrol.FlowDistinguisherMethod{
			FlowDistinguisherMethodType: flowcontrol.FlowDistinguisherMethodByUserType,
		},
		Rules: []flowcontrol.PolicyRulesWithSubjects{
			ResourceRules: []ResourcePolicyRule{
				APIGroups: []string{
					v1.KubeVirtGroupVersionKind.Group,
				},
				ClusterScope: true,
				Namespaces: []string{
					"'*'",
				},
				Resource: []string{
					v1.VirtualMachineGroupVersionKind.Kind,
					v1.VirtualMachineInstanceGroupVersionKind.Kind,
					v1.VirtualMachineInstanceReplicaSetGroupVersionKind.King,
					v1.VirtualMachineInstancePresetGroupVersionKind.Kind,
					vi.VirtualMachineInstanceMigrationGroupVersionKind.Kind,
				},
				Verbs: []string{
					"'*'",
				},
			},
			Subjects: []flowcontrol.Subject{
				Kind: flowcontrol.SubjectKindServiceAccount,
				ServiceAccount: &flowcontrol.ServiceAccountSubject{
					Name:      "'.'",
					Namespace: "kubevirt",
				},
			},
		},
	}
}
