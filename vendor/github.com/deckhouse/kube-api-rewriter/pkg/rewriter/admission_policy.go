/*
Copyright 2024 Flant JSC

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

package rewriter

const (
	ValidatingAdmissionPolicyKind            = "ValidatingAdmissionPolicy"
	ValidatingAdmissionPolicyListKind        = "ValidatingAdmissionPolicyList"
	ValidatingAdmissionPolicyBindingKind     = "ValidatingAdmissionPolicyBinding"
	ValidatingAdmissionPolicyBindingListKind = "ValidatingAdmissionPolicyBindingList"
)

// renames apiGroups and resources in a single resourceRule.
// Rule examples:
//	resourceRules:
//	- apiGroups:
//	    - ""
//	  apiVersions:
//      - '*'
//    operations:
//      - '*'
//    resources:
//      - nodes
//    scope: '*'

func RewriteValidatingAdmissionPolicyOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	if action == Rename {
		return RewriteResourceOrList(obj, ValidatingAdmissionPolicyListKind, func(singleObj []byte) ([]byte, error) {
			return RewriteArray(singleObj, "spec.matchConstraints.resourceRules", func(item []byte) ([]byte, error) {
				return RenameResourceRule(rules, item)
			})
		})
	}
	return RewriteResourceOrList(obj, ValidatingAdmissionPolicyListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteArray(singleObj, "spec.matchConstraints.resourceRules", func(item []byte) ([]byte, error) {
			return RestoreResourceRule(rules, item)
		})
	})
}

func RewriteValidatingAdmissionPolicyBindingOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	if action == Rename {
		return RewriteResourceOrList(obj, ValidatingAdmissionPolicyBindingListKind, func(singleObj []byte) ([]byte, error) {
			return RewriteArray(singleObj, "spec.matchResources.resourceRules", func(item []byte) ([]byte, error) {
				return RenameResourceRule(rules, item)
			})
		})
	}
	return RewriteResourceOrList(obj, ValidatingAdmissionPolicyBindingListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteArray(singleObj, "spec.matchResources.resourceRules", func(item []byte) ([]byte, error) {
			return RestoreResourceRule(rules, item)
		})
	})
}
