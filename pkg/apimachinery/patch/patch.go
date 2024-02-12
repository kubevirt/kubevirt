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

type PatchSet interface {
	Test(path string, value interface{})
	Add(path string, value interface{})
	Remove(path string)
	Replace(path string, value interface{})
	TestAndReplace(path string, oldVal, newVal interface{})
	AddOrReplace(path string, value interface{}, condition bool)
	IsEmpty() bool
	GeneratePayload() ([]byte, error)
}

type patchSet struct {
	patches []PatchOperation
}

func New() PatchSet {
	return &patchSet{}
}

func (p *patchSet) IsEmpty() bool {
	return len(p.patches) < 1
}

func (p *patchSet) addOp(op, path string, value interface{}) {
	p.patches = append(p.patches, PatchOperation{
		Op:    op,
		Path:  path,
		Value: value,
	})
}

func (p *patchSet) Test(path string, value interface{}) {
	p.addOp(PatchTestOp, path, value)
}

func (p *patchSet) Add(path string, value interface{}) {
	p.addOp(PatchAddOp, path, value)
}

func (p *patchSet) Remove(path string) {
	p.addOp(PatchRemoveOp, path, nil)
}

func (p *patchSet) Replace(path string, value interface{}) {
	p.addOp(PatchReplaceOp, path, value)
}

func (p *patchSet) TestAndReplace(path string, oldVal, newVal interface{}) {
	p.Test(path, oldVal)
	p.Replace(path, newVal)
}

func (p *patchSet) AddOrReplace(path string, value interface{}, condition bool) {
	if condition {
		p.Add(path, value)
	} else {
		p.Replace(path, value)
	}
}

func (p *patchSet) GeneratePayload() ([]byte, error) {
	return GeneratePatchPayload(p.patches...)
}

func GeneratePatchPayload(patches ...PatchOperation) ([]byte, error) {
	if len(patches) == 0 {
		return nil, nil
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
