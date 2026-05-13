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

package cel

import (
	"fmt"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// sparseProvider wraps a types.Provider, overriding only NewValue to return
// sparse objectVal values. All type discovery delegates to the underlying
// provider (backed by ext.NativeTypes).
type sparseProvider struct {
	delegate types.Provider
}

func (p *sparseProvider) EnumValue(enumName string) ref.Val {
	return p.delegate.EnumValue(enumName)
}

func (p *sparseProvider) FindIdent(identName string) (ref.Val, bool) {
	return p.delegate.FindIdent(identName)
}

func (p *sparseProvider) FindStructType(typeName string) (*types.Type, bool) {
	return p.delegate.FindStructType(typeName)
}

func (p *sparseProvider) FindStructFieldNames(typeName string) ([]string, bool) {
	return p.delegate.FindStructFieldNames(typeName)
}

// FindStructFieldType delegates type lookup to NativeTypes but wraps the field
// accessors to handle both value types that coexist at runtime:
//   - objectVal (from NewValue): sparse mutation results - read from field map
//   - nativeObj (from NativeTypes adapter): Go struct inputs like domainSpec -
//     delegate to the original accessor which uses reflect
func (p *sparseProvider) FindStructFieldType(typeName, fieldName string) (*types.FieldType, bool) {
	ft, found := p.delegate.FindStructFieldType(typeName, fieldName)
	if !found {
		return nil, false
	}
	return &types.FieldType{
		Type: ft.Type,
		IsSet: func(target any) bool {
			if ov, ok := target.(*objectVal); ok {
				_, present := ov.fields[fieldName]
				return present
			}
			return ft.IsSet(target)
		},
		GetFrom: func(target any) (any, error) {
			if ov, ok := target.(*objectVal); ok {
				val, present := ov.fields[fieldName]
				if !present {
					return nil, fmt.Errorf("field '%s' not set", fieldName)
				}
				return val, nil
			}
			return ft.GetFrom(target)
		},
	}, true
}

// NewValue returns a sparse objectVal that only tracks the explicitly
// provided fields. This is the only method that differs from NativeTypes,
// which would create a full zero-valued Go struct via reflect.New().
func (p *sparseProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	return &objectVal{
		typeName: typeName,
		fields:   fields,
	}
}
