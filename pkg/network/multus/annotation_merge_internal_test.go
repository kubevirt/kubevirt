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
 *
 */

package multus

import (
	"reflect"
	"strings"
	"testing"
)

// TestOwnedJSONFieldsMatchStructTags pins the owned-field name lists to the
// struct tags: modeling a new field without extending the list (or extending
// the list without the field) fails here, instead of silently treating an
// owned field as external and emitting it twice.
func TestOwnedJSONFieldsMatchStructTags(t *testing.T) {
	testCases := []struct {
		ownedFields []string
		structType  reflect.Type
	}{
		{elementOwnedJSONFields, reflect.TypeFor[networkSelectionElement]()},
		{cniArgsOwnedJSONFields, reflect.TypeFor[cniArgs]()},
	}

	for _, testCase := range testCases {
		expected := jsonTagNames(testCase.structType)
		if !reflect.DeepEqual(testCase.ownedFields, expected) {
			t.Errorf("owned fields %v do not match the json tags %v of %s",
				testCase.ownedFields, expected, testCase.structType)
		}
	}
}

func jsonTagNames(structType reflect.Type) []string {
	names := make([]string, 0, structType.NumField())
	for field := range structType.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue // the unexported extras field carries no tag
		}
		names = append(names, name)
	}

	return names
}
