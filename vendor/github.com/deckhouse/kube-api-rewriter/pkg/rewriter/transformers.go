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
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// TransformString transforms string value addressed by path.
func TransformString(obj []byte, path string, transformFn func(field string) string) ([]byte, error) {
	pathStr := gjson.GetBytes(obj, path)
	if !pathStr.Exists() {
		return obj, nil
	}
	rwrString := transformFn(pathStr.String())
	return sjson.SetBytes(obj, path, rwrString)
}

// TransformObject transforms object value addressed by path.
func TransformObject(obj []byte, path string, transformFn func(item []byte) ([]byte, error)) ([]byte, error) {
	pathObj := gjson.GetBytes(obj, path)
	if !pathObj.IsObject() {
		return obj, nil
	}
	rwrObj, err := transformFn([]byte(pathObj.Raw))
	if err != nil {
		return nil, err
	}
	return sjson.SetRawBytes(obj, path, rwrObj)
}

// TransformArrayOfStrings transforms array value addressed by path.
func TransformArrayOfStrings(obj []byte, arrayPath string, transformFn func(item string) string) ([]byte, error) {
	// Transform each item in list. Put back original items if transformFn returns nil bytes.
	items := gjson.GetBytes(obj, arrayPath).Array()
	if len(items) == 0 {
		return obj, nil
	}
	rwrItems := make([]string, len(items))
	for i, item := range items {
		rwrItems[i] = transformFn(item.String())
	}

	return sjson.SetBytes(obj, arrayPath, rwrItems)
}

// TransformPatch treats obj as a JSON patch or Merge patch and calls
// a corresponding transformFn.
func TransformPatch(
	obj []byte,
	transformMerge func(mergePatch []byte) ([]byte, error),
	transformJSON func(jsonPatch []byte) ([]byte, error)) ([]byte, error) {
	if len(obj) == 0 {
		return obj, nil
	}
	// Merge patch for Kubernetes resource is always starts with the curly bracket.
	if string(obj[0]) == "{" && transformMerge != nil {
		return transformMerge(obj)
	}

	// JSON patch should start with the square bracket.
	if string(obj[0]) == "[" && transformJSON != nil {
		return RewriteArray(obj, Root, transformJSON)
	}

	// Return patch as-is in other cases.
	return obj, nil
}

// Helpers for traversing JSON objects with support for root path.
// gjson supports @this, but sjson don't, so unique alias is used.

const Root = "@ROOT"

func GetBytes(obj []byte, path string) gjson.Result {
	if path == Root {
		return gjson.ParseBytes(obj)
	}
	return gjson.GetBytes(obj, path)
}

func SetBytes(obj []byte, path string, value interface{}) ([]byte, error) {
	if path == Root {
		return json.Marshal(value)
	}
	return sjson.SetBytes(obj, path, value)
}

func SetRawBytes(obj []byte, path string, value []byte) ([]byte, error) {
	if path == Root {
		return value, nil
	}
	return sjson.SetRawBytes(obj, path, value)
}
