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

// TODO merge this file into transformers.go

// RewriteMapStringString transforms map[string]string value addressed by path.
func RewriteMapStringString(obj []byte, mapPath string, transformFn func(k, v string) (string, string)) ([]byte, error) {
	m := gjson.GetBytes(obj, mapPath).Map()
	if len(m) == 0 {
		return obj, nil
	}
	newMap := make(map[string]string, len(m))
	for k, v := range m {
		newK, newV := transformFn(k, v.String())
		newMap[newK] = newV
	}

	return sjson.SetBytes(obj, mapPath, newMap)
}
