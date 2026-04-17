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
	ClusterRoleKind            = "ClusterRole"
	ClusterRoleListKind        = "ClusterRoleList"
	RoleKind                   = "Role"
	RoleListKind               = "RoleList"
	RoleBindingKind            = "RoleBinding"
	RoleBindingListKind        = "RoleBindingList"
	ControllerRevisionKind     = "ControllerRevision"
	ControllerRevisionListKind = "ControllerRevisionList"
	ClusterRoleBindingKind     = "ClusterRoleBinding"
	ClusterRoleBindingListKind = "ClusterRoleBindingList"
	APIServiceKind             = "APIService"
	APIServiceListKind         = "APIServiceList"
)

func RewriteClusterRoleOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	if action == Rename {
		return RewriteResourceOrList(obj, ClusterRoleListKind, func(singleObj []byte) ([]byte, error) {
			return RewriteArray(singleObj, "rules", func(item []byte) ([]byte, error) {
				return RenameResourceRule(rules, item)
			})
		})
	}
	return RewriteResourceOrList(obj, ClusterRoleListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteArray(singleObj, "rules", func(item []byte) ([]byte, error) {
			return RestoreResourceRule(rules, item)
		})
	})
}

func RewriteRoleOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	if action == Rename {
		return RewriteResourceOrList(obj, RoleListKind, func(singleObj []byte) ([]byte, error) {
			return RewriteArray(singleObj, "rules", func(item []byte) ([]byte, error) {
				return RenameResourceRule(rules, item)
			})
		})
	}
	return RewriteResourceOrList(obj, RoleListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteArray(singleObj, "rules", func(item []byte) ([]byte, error) {
			return RestoreResourceRule(rules, item)
		})
	})
}

// RenameResourceRule renames apiGroups and resources in a single rule.
// Rule examples:
//   - apiGroups:
//   - original.group.io
//     resources:
//   - '*'
//     verbs:
//   - '*'
//   - apiGroups:
//   - original.group.io
//     resources:
//   - someresources
//   - someresources/finalizers
//   - someresources/status
//   - someresources/scale
//     verbs:
//   - watch
//   - list
//   - create
func RenameResourceRule(rules *RewriteRules, obj []byte) ([]byte, error) {
	var err error

	renameResources := false
	obj, err = TransformArrayOfStrings(obj, "apiGroups", func(apiGroup string) string {
		if rules.HasGroup(apiGroup) {
			renameResources = true
			return rules.RenameApiVersion(apiGroup)
		}
		if apiGroup == "*" {
			renameResources = true
		}
		return apiGroup
	})
	if err != nil {
		return nil, err
	}

	// Do not rename resources for unknown group.
	if !renameResources {
		return obj, nil
	}

	return TransformArrayOfStrings(obj, "resources", func(resourceType string) string {
		if resourceType == "*" || resourceType == "" {
			return resourceType
		}

		// Rename if there is rule for resourceType.
		_, resRule := rules.GroupResourceRules(resourceType)
		if resRule != nil {
			return rules.RenameResource(resourceType)
		}
		return resourceType
	})
}

// RestoreResourceRule restores apiGroups and resources in a single rule.
func RestoreResourceRule(rules *RewriteRules, obj []byte) ([]byte, error) {
	var err error

	restoreResources := false
	obj, err = TransformArrayOfStrings(obj, "apiGroups", func(apiGroup string) string {
		if rules.IsRenamedGroup(apiGroup) {
			restoreResources = true
			return rules.RestoreApiVersion(apiGroup)
		}
		if apiGroup == "*" {
			restoreResources = true
		}
		return apiGroup
	})
	if err != nil {
		return nil, err
	}

	// Do not rename resources for unknown group.
	if !restoreResources {
		return obj, nil
	}

	return TransformArrayOfStrings(obj, "resources", func(resourceType string) string {
		if resourceType == "*" || resourceType == "" {
			return resourceType
		}
		// Get rules for resource by restored resourceType.
		originalResourceType := rules.RestoreResource(resourceType)
		_, resRule := rules.GroupResourceRules(originalResourceType)
		if resRule != nil {
			// NOTE: subresource not trimmed.
			return originalResourceType
		}

		// No rules for resourceType, return as-is
		return resourceType
	})
}
