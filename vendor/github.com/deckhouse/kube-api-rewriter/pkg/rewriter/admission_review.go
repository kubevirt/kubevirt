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
	"encoding/base64"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// RewriteAdmissionReview rewrites AdmissionReview request and response.
// NOTE: only one rewrite direction is supported for now:
// - Restore object in AdmissionReview request.
// - Do nothing for AdmissionReview response.
func RewriteAdmissionReview(rules *RewriteRules, obj []byte) ([]byte, error) {
	if gjson.GetBytes(obj, "response").Exists() {
		return TransformObject(obj, "response", func(responseObj []byte) ([]byte, error) {
			return RenameAdmissionReviewResponse(rules, responseObj)
		})
	}

	request := gjson.GetBytes(obj, "request")
	if request.Exists() {
		newRequest, err := RestoreAdmissionReviewRequest(rules, []byte(request.Raw))
		if err != nil {
			return nil, err
		}
		if len(newRequest) > 0 {
			obj, err = sjson.SetRawBytes(obj, "request", newRequest)
			if err != nil {
				return nil, err
			}
		}
	}

	return obj, nil
}

// RenameAdmissionReviewResponse renames metadata in AdmissionReview response patch.
// AdmissionReview response example:
//
//	"response": {
//	   "uid": "<value from request.uid>",
//	   "allowed": true,
//	   "patchType": "JSONPatch",
//	   "patch": "W3sib3AiOiAiYWRkIiwgInBhdGgiOiAiL3NwZWMvcmVwbGljYXMiLCAidmFsdWUiOiAzfV0="
//	 }
//
// TODO rename annotations in AuditAnnotations field. (Ignore for now, as not used by the kubevirt).
func RenameAdmissionReviewResponse(rules *RewriteRules, obj []byte) ([]byte, error) {
	// Description for the AdmissionResponse.PatchType field: The type of Patch. Currently, we only allow "JSONPatch".
	patchType := gjson.GetBytes(obj, "patchType").String()
	if patchType != "JSONPatch" {
		return obj, nil
	}

	// Get decoded patch.
	b64Patch := gjson.GetBytes(obj, "patch").String()
	if b64Patch == "" {
		return obj, nil
	}

	patch, err := base64.StdEncoding.DecodeString(b64Patch)
	if err != nil {
		return nil, fmt.Errorf("decode base64 patch: %w", err)
	}

	rwrPatch, err := RenameMetadataPatch(rules, patch)
	if err != nil {
		return nil, fmt.Errorf("rename metadata patch: %w", err)
	}

	// Update patch field to base64 encoded rewritten patch.
	return sjson.SetBytes(obj, "patch", base64.StdEncoding.EncodeToString(rwrPatch))
}

// RestoreAdmissionReviewRequest restores apiVersion, kind and other fields in an AdmissionReview request.
// Only restoring is required, as AdmissionReview request only comes from API Server.
// Fields for AdmissionReview request:
//
//	kind, requestKind: - Fully-qualified group/version/kind of the incoming object
//	  kind - restore
//	  version
//	  group - restore
//	resource, requestResource - Fully-qualified group/version/kind of the resource being modified
//	  group - restore
//	  version
//	  resource - restore
//	object, oldObject - new and old objects being admitted, should be restored.
//
//	non-rewritable:
//	  uid - review uid, no rewrite
//	  subResource, requestSubResource - scale or status, no rewrite
//	  name
//	  namespace
//	  operation
//	  userInfo
//	  options
//	  dryRun
func RestoreAdmissionReviewRequest(rules *RewriteRules, obj []byte) ([]byte, error) {
	var err error

	// Rewrite "resource" field and find rules.
	{
		resourceObj := gjson.GetBytes(obj, "resource")
		group := resourceObj.Get("group")
		resource := resourceObj.Get("resource")
		// Ignore reviews for unknown renamed group.
		if !rules.IsRenamedGroup(group.String()) {
			return nil, nil
		}
		restoredResourceType := rules.RestoreResource(resource.String())
		obj, err = sjson.SetBytes(obj, "resource.resource", restoredResourceType)
		if err != nil {
			return nil, err
		}
		restoredGroup := rules.RestoreApiVersion(group.String())
		obj, err = sjson.SetBytes(obj, "resource.group", restoredGroup)
		if err != nil {
			return nil, err
		}
	}

	// Rewrite "requestResource" field.
	{
		fieldObj := gjson.GetBytes(obj, "requestResource")
		group := fieldObj.Get("group")
		resource := fieldObj.Get("resource")
		// Ignore reviews for unknown renamed group.
		if !rules.IsRenamedGroup(group.String()) {
			return nil, nil
		}
		restoredResourceType := rules.RestoreResource(resource.String())
		obj, err = sjson.SetBytes(obj, "requestResource.resource", restoredResourceType)
		if err != nil {
			return nil, err
		}
		restoredGroup := rules.RestoreApiVersion(group.String())
		obj, err = sjson.SetBytes(obj, "requestResource.group", restoredGroup)
		if err != nil {
			return nil, err
		}
	}

	// Check "subresource" field. No need to rewrite kind, requestKind, object and oldObject fields if subresource is set.
	{
		fieldObj := gjson.GetBytes(obj, "subresource")
		if fieldObj.Exists() && fieldObj.String() != "" {
			return obj, err
		}
	}

	// Rewrite "kind" field.
	{
		fieldObj := gjson.GetBytes(obj, "kind")
		kind := fieldObj.Get("kind")
		restoredKind := rules.RestoreKind(kind.String())
		obj, err = sjson.SetBytes(obj, "kind.kind", restoredKind)
		if err != nil {
			return nil, err
		}
		group := fieldObj.Get("group")
		restoredGroup := rules.RestoreApiVersion(group.String())
		obj, err = sjson.SetBytes(obj, "kind.group", restoredGroup)
		if err != nil {
			return nil, err
		}
	}

	// Rewrite "requestKind" field.
	{
		fieldObj := gjson.GetBytes(obj, "requestKind")
		kind := fieldObj.Get("kind")
		restoredKind := rules.RestoreKind(kind.String())
		obj, err = sjson.SetBytes(obj, "requestKind.kind", restoredKind)
		if err != nil {
			return nil, err
		}
		group := fieldObj.Get("group")
		restoredGroup := rules.RestoreApiVersion(group.String())
		obj, err = sjson.SetBytes(obj, "requestKind.group", restoredGroup)
		if err != nil {
			return nil, err
		}
	}

	// Rewrite "object" field.
	obj, err = TransformObject(obj, "object", func(objectObj []byte) ([]byte, error) {
		return RestoreAdmissionReviewObject(rules, objectObj)
	})
	if err != nil {
		return nil, fmt.Errorf("restore 'object': %w", err)
	}
	// Rewrite "object" field.
	obj, err = TransformObject(obj, "oldObject", func(objectObj []byte) ([]byte, error) {
		return RestoreAdmissionReviewObject(rules, objectObj)
	})
	if err != nil {
		return nil, fmt.Errorf("restore 'oldObject': %w", err)
	}

	return obj, nil
}

// RestoreAdmissionReviewObject fully restores object of known resource.
// TODO deduplicate with code in RewriteJSONPayload.
func RestoreAdmissionReviewObject(rules *RewriteRules, obj []byte) ([]byte, error) {
	var err error
	obj, err = RestoreResource(rules, obj)
	if err != nil {
		return nil, fmt.Errorf("restore resource group, kind: %w", err)
	}

	obj, err = TransformObject(obj, "metadata", func(metadataObj []byte) ([]byte, error) {
		return RewriteMetadata(rules, metadataObj, Restore)
	})
	if err != nil {
		return nil, fmt.Errorf("restore resource metadata: %w", err)
	}

	return obj, nil
}
