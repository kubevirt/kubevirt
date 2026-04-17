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
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func RewriteMetadata(rules *RewriteRules, metadataObj []byte, action Action) ([]byte, error) {
	metadataObj, err := RewriteLabelsMap(rules, metadataObj, "labels", action)
	if err != nil {
		return nil, err
	}
	metadataObj, err = RewriteAnnotationsMap(rules, metadataObj, "annotations", action)
	if err != nil {
		return nil, err
	}
	metadataObj, err = RewriteFinalizers(rules, metadataObj, "finalizers", action)
	if err != nil {
		return nil, err
	}
	return RewriteOwnerReferences(rules, metadataObj, "ownerReferences", action)
}

// RenameMetadataPatch transforms known metadata fields in patches.
// Example:
// - merge patch on metadata:
// {"metadata": { "labels": {"kubevirt.io/schedulable": "false", "cpumanager": "false"}, "annotations": {"kubevirt.io/heartbeat": "2024-06-07T23:27:53Z"}}}
// - JSON patch on metadata:
// [{"op":"test", "path":"/metadata/labels", "value":{"label":"value"}},
//
//	{"op":"replace", "path":"/metadata/labels", "value":{"label":"newValue"}}]
func RenameMetadataPatch(rules *RewriteRules, patch []byte) ([]byte, error) {
	return TransformPatch(patch,
		func(mergePatch []byte) ([]byte, error) {
			return TransformObject(mergePatch, "metadata", func(metadataObj []byte) ([]byte, error) {
				return RewriteMetadata(rules, metadataObj, Rename)
			})
		},
		func(jsonPatch []byte) ([]byte, error) {
			path := gjson.GetBytes(jsonPatch, "path").String()
			switch path {
			case "/metadata/labels":
				return RewriteLabelsMap(rules, jsonPatch, "value", Rename)
			case "/metadata/annotations":
				return RewriteAnnotationsMap(rules, jsonPatch, "value", Rename)
			case "/metadata/finalizers":
				return RewriteFinalizers(rules, jsonPatch, "value", Rename)
			case "/metadata/ownerReferences":
				return RewriteOwnerReferences(rules, jsonPatch, "value", Rename)
			case "/metadata":
				return TransformObject(jsonPatch, "value", func(metadataObj []byte) ([]byte, error) {
					return RewriteMetadata(rules, metadataObj, Rename)
				})
			}

			encLabel, found := strings.CutPrefix(path, "/metadata/labels/")
			if found {
				label := decodeJSONPatchPath(encLabel)
				rwrLabel := rules.LabelsRewriter().Rewrite(label, Rename)
				if label != rwrLabel {
					return sjson.SetBytes(jsonPatch, "path", "/metadata/labels/"+encodeJSONPatchPath(rwrLabel))
				}
			}

			encAnno, found := strings.CutPrefix(path, "/metadata/annotations/")
			if found {
				anno := decodeJSONPatchPath(encAnno)
				rwrAnno := rules.AnnotationsRewriter().Rewrite(anno, Rename)
				if anno != rwrAnno {
					return sjson.SetBytes(jsonPatch, "path", "/metadata/annotations/"+encodeJSONPatchPath(rwrAnno))
				}
			}

			encFin, found := strings.CutPrefix(path, "/metadata/finalizers/")
			if found {
				fin := decodeJSONPatchPath(encFin)
				rwrFin := rules.FinalizersRewriter().Rewrite(fin, Rename)
				if fin != rwrFin {
					return sjson.SetBytes(jsonPatch, "path", "/metadata/finalizers/"+encodeJSONPatchPath(rwrFin))
				}
			}

			return jsonPatch, nil
		})
}

func RewriteLabelsMap(rules *RewriteRules, obj []byte, path string, action Action) ([]byte, error) {
	return RewriteMapStringString(obj, path, func(k, v string) (string, string) {
		return rules.LabelsRewriter().RewriteNameValue(k, v, action)
	})
}

func RewriteAnnotationsMap(rules *RewriteRules, obj []byte, path string, action Action) ([]byte, error) {
	return RewriteMapStringString(obj, path, func(k, v string) (string, string) {
		return rules.AnnotationsRewriter().RewriteNameValue(k, v, action)
	})
}

func RewriteFinalizers(rules *RewriteRules, obj []byte, path string, action Action) ([]byte, error) {
	return TransformArrayOfStrings(obj, path, func(finalizer string) string {
		return rules.FinalizersRewriter().Rewrite(finalizer, action)
	})
}

const (
	tildeChar        = "~"
	tildePlaceholder = "~0"
	slashChar        = "/"
	slashPlaceholder = "~1"
)

// decodeJSONPatchPath restores ~ and / from ~0 and ~1.
// See https://jsonpatch.com/#json-pointer
func decodeJSONPatchPath(path string) string {
	// Restore / first to prevent tilde doubling.
	res := strings.Replace(path, slashPlaceholder, slashChar, -1)
	return strings.Replace(res, tildePlaceholder, tildeChar, -1)
}

// encodeJSONPatchPath replaces ~ and / to ~0 and ~1.
// See https://jsonpatch.com/#json-pointer
func encodeJSONPatchPath(path string) string {
	// Replace ~ first to prevent tilde doubling.
	res := strings.Replace(path, tildeChar, tildePlaceholder, -1)
	return strings.Replace(res, slashChar, slashPlaceholder, -1)
}
