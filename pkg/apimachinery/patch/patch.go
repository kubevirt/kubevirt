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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package patch

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

const (
	PatchReplaceOp = "replace"
	PatchTestOp    = "test"
	PatchAddOp     = "add"
	PatchRemoveOp  = "remove"
)

func GeneratePatchPayload(patches ...PatchOperation) ([]byte, error) {
	if len(patches) == 0 {
		return nil, fmt.Errorf("list of patches is empty")
	}

	payloadBytes, err := json.Marshal(patches)
	if err != nil {
		return nil, err
	}

	return payloadBytes, nil
}

func GenerateTestReplacePatch(path string, oldValue, newValue interface{}) ([]byte, error) {
	return GeneratePatchPayload(
		PatchOperation{
			Op:    PatchTestOp,
			Path:  path,
			Value: oldValue,
		},
		PatchOperation{
			Op:    PatchReplaceOp,
			Path:  path,
			Value: newValue,
		},
	)
}

func UnmarshalPatch(patch []byte) ([]PatchOperation, error) {
	var p []PatchOperation
	err := json.Unmarshal(patch, &p)

	return p, err
}

func EscapeJSONPointer(ptr string) string {
	s := strings.ReplaceAll(ptr, "~", "~0")
	return strings.ReplaceAll(s, "/", "~1")
}
