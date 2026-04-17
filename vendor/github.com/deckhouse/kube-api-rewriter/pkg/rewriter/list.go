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
	"bytes"
	"errors"
	"strings"

	"github.com/tidwall/gjson"
)

// TODO merge this file into transformers.go

// RewriteResourceOrList is a helper to transform a single resource or a list of resources.
func RewriteResourceOrList(payload []byte, listKind string, transformFn func(singleObj []byte) ([]byte, error)) ([]byte, error) {
	kind := gjson.GetBytes(payload, "kind").String()

	// Not a list, transform a single resource.
	if kind != listKind {
		return transformFn(payload)
	}

	return RewriteArray(payload, "items", transformFn)
}

// RewriteResourceOrList2 is a helper to transform a single resource or a list of resources.
func RewriteResourceOrList2(payload []byte, transformFn func(singleObj []byte) ([]byte, error)) ([]byte, error) {
	kind := gjson.GetBytes(payload, "kind").String()
	if !strings.HasSuffix(kind, "List") {
		return transformFn(payload)
	}
	return RewriteArray(payload, "items", transformFn)
}

// SkipItem may be used by the transformFn to indicate that the item should be skipped from the result.
var SkipItem = errors.New("remove item from the result")

// RewriteArray gets array by path and transforms each item using transformFn.
// Use Root path to transform object itself.
// transformFn contract:
// return obj, nil -> obj is considered a replacement for the element.
// return nil, nil -> no transformation, element is added as-is.
// return any, SkipItem -> no transformation and no adding to the result.
// return any, err -> stop transformation, return error.
func RewriteArray(obj []byte, arrayPath string, transformFn func(item []byte) ([]byte, error)) ([]byte, error) {
	// Transform each item in list. Put back original items if transformFn returns nil bytes.
	items := GetBytes(obj, arrayPath).Array()
	if len(items) == 0 {
		return obj, nil
	}

	var rwrItems bytes.Buffer
	rwrItems.Grow(len(obj))
	// Start array
	rwrItems.WriteString(`[`)

	first := true
	for _, item := range items {

		rwrItem, err := transformFn([]byte(item.Raw))
		if err != nil {
			if errors.Is(err, SkipItem) {
				continue
			}
			return nil, err
		}

		// Prepend a comma for all elements except the first one.
		if first {
			first = false
		} else {
			rwrItems.WriteString(`,`)
		}

		// Put original item back to allow transformFn returns nil.
		if rwrItem == nil {
			rwrItem = []byte(item.Raw)
		}

		rwrItems.Write(rwrItem)
	}

	// Close array
	rwrItems.WriteString(`]`)
	return SetRawBytes(obj, arrayPath, rwrItems.Bytes())
}
