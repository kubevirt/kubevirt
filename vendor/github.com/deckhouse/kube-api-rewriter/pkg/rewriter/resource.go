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

func RewriteCustomResourceOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	kind := gjson.GetBytes(obj, "kind").String()
	if action == Restore {
		kind = rules.RestoreKind(kind)
	}
	origGroupName, origResName, isList := rules.ResourceByKind(kind)
	if origGroupName == "" && origResName == "" {
		// Return as-is if kind is not in rules.
		return obj, nil
	}
	if isList {
		if action == Restore {
			return RestoreResourcesList(rules, obj)
		}

		return RenameResourcesList(rules, obj)
	}

	// Responses of GET, LIST, DELETE requests.
	// AdmissionReview requests from API Server.
	if action == Restore {
		return RestoreResource(rules, obj)
	}
	// CREATE, UPDATE, PATCH requests.
	// TODO need to implement for
	return RenameResource(rules, obj)
}

func RenameResourcesList(rules *RewriteRules, obj []byte) ([]byte, error) {
	obj, err := RenameAPIVersionAndKind(rules, obj)
	if err != nil {
		return nil, err
	}

	// Rewrite apiVersion and kind in each item.
	return RewriteArray(obj, "items", func(singleResource []byte) ([]byte, error) {
		return RenameResource(rules, singleResource)
	})
}

func RestoreResourcesList(rules *RewriteRules, obj []byte) ([]byte, error) {
	obj, err := RestoreAPIVersionAndKind(rules, obj)
	if err != nil {
		return nil, err
	}

	// Restore apiVersion and kind in each item.
	return RewriteArray(obj, "items", func(singleResource []byte) ([]byte, error) {
		return RestoreResource(rules, singleResource)
	})
}

func RenameResource(rules *RewriteRules, obj []byte) ([]byte, error) {
	obj, err := RenameAPIVersionAndKind(rules, obj)
	if err != nil {
		return nil, err
	}

	// Rewrite apiVersion in each managedFields.
	return RenameManagedFields(rules, obj)
}

func RestoreResource(rules *RewriteRules, obj []byte) ([]byte, error) {
	obj, err := RestoreAPIVersionAndKind(rules, obj)
	if err != nil {
		return nil, err
	}

	// Rewrite apiVersion in each managedFields.
	return RestoreManagedFields(rules, obj)
}

func RenameAPIVersionAndKind(rules *RewriteRules, obj []byte) ([]byte, error) {
	apiVersion := gjson.GetBytes(obj, "apiVersion").String()
	obj, err := sjson.SetBytes(obj, "apiVersion", rules.RenameApiVersion(apiVersion))
	if err != nil {
		return nil, err
	}

	kind := gjson.GetBytes(obj, "kind").String()
	return sjson.SetBytes(obj, "kind", rules.RenameKind(kind))
}

func RestoreAPIVersionAndKind(rules *RewriteRules, obj []byte) ([]byte, error) {
	apiVersion := gjson.GetBytes(obj, "apiVersion").String()
	apiVersion = rules.RestoreApiVersion(apiVersion)
	obj, err := sjson.SetBytes(obj, "apiVersion", apiVersion)
	if err != nil {
		return nil, err
	}

	kind := gjson.GetBytes(obj, "kind").String()
	return sjson.SetBytes(obj, "kind", rules.RestoreKind(kind))
}

func RewriteOwnerReferences(rules *RewriteRules, obj []byte, path string, action Action) ([]byte, error) {
	return RewriteArray(obj, path, func(ownerRefObj []byte) ([]byte, error) {
		return RewriteAPIVersionAndKind(rules, ownerRefObj, action)
	})
}

// RestoreManagedFields restores apiVersion in managedFields items.
//
// Example response from the server:
//
//	"metadata": {
//	  "managedFields":[
//	    { "apiVersion":"renamed.resource.group.io/v1", "fieldsType":"FieldsV1", "fieldsV1":{ ... }}, "manager": "Go-http-client", ...},
//	    { "apiVersion":"renamed.resource.group.io/v1", "fieldsType":"FieldsV1", "fieldsV1":{ ... }}, "manager": "kubectl-edit", ...}
//	  ],
func RestoreManagedFields(rules *RewriteRules, obj []byte) ([]byte, error) {
	return RewriteArray(obj, "metadata.managedFields", func(managedField []byte) ([]byte, error) {
		return TransformString(managedField, "apiVersion", func(apiVersion string) string {
			return rules.RestoreApiVersion(apiVersion)
		})
	})
}

// RenameManagedFields renames apiVersion in managedFields items.
//
// Example request from the client:
//
//	"metadata": {
//	  "managedFields":[
//	    { "apiVersion":"original.group.io/v1", "fieldsType":"FieldsV1", "fieldsV1":{ ... }}, "manager": "Go-http-client", ...},
//	    { "apiVersion":"original.group.io/v1", "fieldsType":"FieldsV1", "fieldsV1":{ ... }}, "manager": "kubectl-edit", ...}
//	  ],
func RenameManagedFields(rules *RewriteRules, obj []byte) ([]byte, error) {
	return RewriteArray(obj, "metadata.managedFields", func(managedField []byte) ([]byte, error) {
		return TransformString(managedField, "apiVersion", func(apiVersion string) string {
			return rules.RenameApiVersion(apiVersion)
		})
	})
}

func RenameResourcePatch(rules *RewriteRules, patch []byte) ([]byte, error) {
	return RenameMetadataPatch(rules, patch)
}
