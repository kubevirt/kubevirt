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

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func RewriteAPIGroupAndKind(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteGVK(rules, obj, action, "apiGroup")
}

func RewriteAPIVersionAndKind(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteGVK(rules, obj, action, "apiVersion")
}

// RewriteGVK rewrites a "kind" field and a field with the group
// if there is the rule for these particular kind and group.
func RewriteGVK(rules *RewriteRules, obj []byte, action Action, gvFieldName string) ([]byte, error) {
	kind := gjson.GetBytes(obj, "kind").String()
	apiGroupVersion := gjson.GetBytes(obj, gvFieldName).String()

	rwrApiVersion := ""
	rwrKind := ""
	if action == Rename {
		// Rename if there is a rule for kind and group
		_, resourceRule := rules.KindRules(apiGroupVersion, kind)
		if resourceRule == nil {
			return obj, nil
		}
		rwrApiVersion = rules.RenameApiVersion(apiGroupVersion)
		rwrKind = rules.RenameKind(kind)
	}
	if action == Restore {
		// Restore if group is renamed and a rule can be found
		// for restored kind and group.
		if !rules.IsRenamedGroup(apiGroupVersion) {
			return obj, nil
		}
		rwrApiVersion = rules.RestoreApiVersion(apiGroupVersion)
		rwrKind = rules.RestoreKind(kind)
		_, resourceRule := rules.KindRules(rwrApiVersion, rwrKind)
		if resourceRule == nil {
			return obj, nil
		}
	}

	obj, err := sjson.SetBytes(obj, "kind", rwrKind)
	if err != nil {
		return nil, err
	}

	return sjson.SetBytes(obj, gvFieldName, rwrApiVersion)
}
