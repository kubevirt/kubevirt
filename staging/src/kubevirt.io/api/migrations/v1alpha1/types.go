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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

// MigrationPolicy holds migration policy (i.e. configurations) to apply to a VM or group of VMs
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
// +genclient:nonNamespaced
type MigrationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MigrationPolicySpec `json:"spec" valid:"required"`
	// +nullable
	Status MigrationPolicyStatus `json:"status,omitempty"`
}

type MigrationPolicySpec struct {
	Selectors *Selectors `json:"selectors"`

	//+optional
	AllowAutoConverge *bool `json:"allowAutoConverge,omitempty"`
	//+optional
	BandwidthPerMigration *resource.Quantity `json:"bandwidthPerMigration,omitempty"`
	//+optional
	CompletionTimeoutPerGiB *int64 `json:"completionTimeoutPerGiB,omitempty"`
	//+optional
	AllowPostCopy *bool `json:"allowPostCopy,omitempty"`
}

type Selectors struct {
	//+optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	//+optional
	VirtualMachineInstanceSelector *metav1.LabelSelector `json:"virtualMachineInstanceSelector,omitempty"`
}

type MigrationPolicyStatus struct {
}

// MigrationPolicyList is a list of MigrationPolicy
//
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MigrationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=atomic
	Items []MigrationPolicy `json:"items"`
}

// GetMigrationConfByPolicy returns a new migration configuration. The new configuration attributes will be overridden
// by the migration policy if the specified attributes were defined for this policy. Otherwise they wouldn't change.
// The boolean returned value indicates if any changes were made to the configurations.
func (m *MigrationPolicy) GetMigrationConfByPolicy(clusterMigrationConfigurations *k6tv1.MigrationConfiguration) (changed bool, err error) {
	policySpec := m.Spec
	changed = false

	if policySpec.AllowAutoConverge != nil {
		changed = true
		*clusterMigrationConfigurations.AllowAutoConverge = *policySpec.AllowAutoConverge
	}
	if policySpec.BandwidthPerMigration != nil {
		changed = true
		*clusterMigrationConfigurations.BandwidthPerMigration = *policySpec.BandwidthPerMigration
	}
	if policySpec.CompletionTimeoutPerGiB != nil {
		changed = true
		*clusterMigrationConfigurations.CompletionTimeoutPerGiB = *policySpec.CompletionTimeoutPerGiB
	}
	if policySpec.AllowPostCopy != nil {
		changed = true
		*clusterMigrationConfigurations.AllowPostCopy = *policySpec.AllowPostCopy
	}

	return changed, nil
}

// +k8s:openapi-gen=false
type migrationPolicyMatchScore struct {
	matchingVMILabels int
	matchingNSLabels  int
}

func (score migrationPolicyMatchScore) equals(otherScore migrationPolicyMatchScore) bool {
	return score.matchingVMILabels == otherScore.matchingVMILabels &&
		score.matchingNSLabels == otherScore.matchingNSLabels
}

func (score migrationPolicyMatchScore) greaterThan(otherScore migrationPolicyMatchScore) bool {
	thisTotalScore := score.matchingNSLabels + score.matchingVMILabels
	otherTotalScore := otherScore.matchingNSLabels + otherScore.matchingVMILabels

	if thisTotalScore == otherTotalScore {
		return score.matchingVMILabels > otherScore.matchingVMILabels
	}

	return thisTotalScore > otherTotalScore
}

func (score migrationPolicyMatchScore) lessThan(otherScore migrationPolicyMatchScore) bool {
	return !score.equals(otherScore) && !score.greaterThan(otherScore)
}

// MatchPolicy returns the policy that is matched to the vmi, or nil of no policy is matched.
//
// Since every policy can specify VMI and Namespace labels to match to, matching is done by returning the most
// detailed policy, meaning the policy that matches the VMI and specifies the most labels that matched either
// the VMI or its namespace labels.
//
// If two policies are matched and have the same level of details (i.e. same number of matching labels) the matched
// policy is chosen by policies' names ordered by lexicographic order. The reason is to create a rather arbitrary yet
// deterministic way of matching policies.
func (list *MigrationPolicyList) MatchPolicy(vmi *k6tv1.VirtualMachineInstance, vmiNamespace *k8sv1.Namespace) *MigrationPolicy {
	var mathingPolicies []MigrationPolicy
	bestScore := migrationPolicyMatchScore{}

	for _, policy := range list.Items {
		doesMatch, curScore := countMatchingLabels(&policy, vmi.Labels, vmiNamespace.Labels)

		if !doesMatch || curScore.lessThan(bestScore) {
			continue
		} else if curScore.greaterThan(bestScore) {
			bestScore = curScore
			mathingPolicies = []MigrationPolicy{policy}
		} else {
			mathingPolicies = append(mathingPolicies, policy)
		}
	}

	if len(mathingPolicies) == 0 {
		return nil
	} else if len(mathingPolicies) == 1 {
		return &mathingPolicies[0]
	}

	// If more than one policy is matched with the same number of matching labels it will be chosen by policies names'
	// lexicographic order
	firstPolicyNameLexicographicOrder := mathingPolicies[0].Name
	var firstPolicyNameLexicographicOrderIdx int

	for idx, matchingPolicy := range mathingPolicies {
		if matchingPolicy.Name < firstPolicyNameLexicographicOrder {
			firstPolicyNameLexicographicOrder = matchingPolicy.Name
			firstPolicyNameLexicographicOrderIdx = idx
		}
	}

	return &mathingPolicies[firstPolicyNameLexicographicOrderIdx]
}

// countMatchingLabels checks if a policy matches to a VMI and the number of matching labels.
// In the case that doesMatch is false, matchingLabels needs to be dismissed and not counted on.
func countMatchingLabels(policy *MigrationPolicy, vmiLabels, namespaceLabels map[string]string) (doesMatch bool, score migrationPolicyMatchScore) {
	var matchingVMILabels, matchingNSLabels int
	doesMatch = true

	if policy.Spec.Selectors == nil {
		return false, score
	}

	countLabelsHelper := func(policyLabels, labelsToMatch map[string]string) (matchingLabels int) {
		for policyKey, policyValue := range policyLabels {
			value, exists := labelsToMatch[policyKey]
			if exists && value == policyValue {
				matchingLabels++
			} else {
				doesMatch = false
				return
			}
		}
		return matchingLabels
	}

	areSelectorsAndLabelsNotNil := func(selector *metav1.LabelSelector, labels map[string]string) bool {
		return selector != nil && selector.MatchLabels != nil && labels != nil
	}

	if areSelectorsAndLabelsNotNil(policy.Spec.Selectors.VirtualMachineInstanceSelector, vmiLabels) {
		matchingVMILabels = countLabelsHelper(policy.Spec.Selectors.VirtualMachineInstanceSelector.MatchLabels, vmiLabels)
	}

	if doesMatch && areSelectorsAndLabelsNotNil(policy.Spec.Selectors.NamespaceSelector, vmiLabels) {
		matchingNSLabels = countLabelsHelper(policy.Spec.Selectors.NamespaceSelector.MatchLabels, namespaceLabels)
	}

	if doesMatch {
		score = migrationPolicyMatchScore{matchingVMILabels: matchingVMILabels, matchingNSLabels: matchingNSLabels}
	}

	return doesMatch, score
}
