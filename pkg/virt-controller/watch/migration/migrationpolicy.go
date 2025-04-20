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
 * Copyright The KubeVirt Authors.
 */

package migration

import (
	k8sv1 "k8s.io/api/core/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations/v1alpha1"
)

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

// matchPolicy returns the policy that is matched to the vmi, or nil of no policy is matched.
//
// Since every policy can specify VMI and Namespace labels to match to, matching is done by returning the most
// detailed policy, meaning the policy that matches the VMI and specifies the most labels that matched either
// the VMI or its namespace labels.
//
// If two policies are matched and have the same level of details (i.e. same number of matching labels) the matched
// policy is chosen by policies' names ordered by lexicographic order. The reason is to create a rather arbitrary yet
// deterministic way of matching policies.
func matchPolicy(policyList *v1alpha1.MigrationPolicyList, vmi *k6tv1.VirtualMachineInstance, vmiNamespace *k8sv1.Namespace) *v1alpha1.MigrationPolicy {
	var mathingPolicies []v1alpha1.MigrationPolicy
	bestScore := migrationPolicyMatchScore{}

	for _, policy := range policyList.Items {
		doesMatch, curScore := countMatchingLabels(&policy, vmi.Labels, vmiNamespace.Labels)

		if !doesMatch || curScore.lessThan(bestScore) {
			continue
		} else if curScore.greaterThan(bestScore) {
			bestScore = curScore
			mathingPolicies = []v1alpha1.MigrationPolicy{policy}
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
func countMatchingLabels(policy *v1alpha1.MigrationPolicy, vmiLabels, namespaceLabels map[string]string) (doesMatch bool, score migrationPolicyMatchScore) {
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

	areSelectorsAndLabelsNotNil := func(selector v1alpha1.LabelSelector, labels map[string]string) bool {
		return selector != nil && labels != nil
	}

	if areSelectorsAndLabelsNotNil(policy.Spec.Selectors.VirtualMachineInstanceSelector, vmiLabels) {
		matchingVMILabels = countLabelsHelper(policy.Spec.Selectors.VirtualMachineInstanceSelector, vmiLabels)
	}

	if doesMatch && areSelectorsAndLabelsNotNil(policy.Spec.Selectors.NamespaceSelector, vmiLabels) {
		matchingNSLabels = countLabelsHelper(policy.Spec.Selectors.NamespaceSelector, namespaceLabels)
	}

	if doesMatch {
		score = migrationPolicyMatchScore{matchingVMILabels: matchingVMILabels, matchingNSLabels: matchingNSLabels}
	}

	return doesMatch, score
}
