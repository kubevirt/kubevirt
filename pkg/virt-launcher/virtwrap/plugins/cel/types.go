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

// types.go defines objectVal, a sparse CEL value type. When a plugin writes
// Domain{Title: "test"}, CEL calls NewValue which returns an objectVal containing
// only {Title: "test"}. This sparse representation is what makes partial mutation
// possible - deepMerge walks the objectVal and only overwrites the fields present.

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// objectValType implements ref.Type for objectVal values.
type objectValType string

func (t objectValType) HasTrait(trait int) bool {
	return trait == traits.FieldTesterType || trait == traits.IndexerType
}

func (t objectValType) TypeName() string {
	return string(t)
}

// objectVal is a sparse CEL value that only contains fields explicitly set
// in a CEL struct construction expression (e.g. Domain{Title: "test"}).
// Fields not mentioned are absent from the map, enabling deep merge to
// distinguish "explicitly set" from "zero-valued".
type objectVal struct {
	typeName string
	fields   map[string]ref.Val
}

func (v *objectVal) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("converting %s to native Go types is not supported", v.typeName)
}

func (v *objectVal) ConvertToType(typeValue ref.Type) ref.Val {
	if typeValue.TypeName() == v.typeName {
		return v
	}
	if typeValue == types.TypeType {
		return types.NewObjectType(v.typeName)
	}
	return types.NewErr("type conversion error from %s to %s", v.typeName, typeValue.TypeName())
}

func (v *objectVal) Equal(other ref.Val) ref.Val {
	return types.NewErr("Equal not supported for %s", v.typeName)
}

func (v *objectVal) Type() ref.Type {
	return objectValType(v.typeName)
}

func (v *objectVal) Value() any {
	return v
}

func (v *objectVal) Get(index ref.Val) ref.Val {
	fieldName, ok := index.Value().(string)
	if !ok {
		return types.NewErr("field name must be a string")
	}
	val, present := v.fields[fieldName]
	if !present {
		return types.NewErr("field '%s' is not set on %s", fieldName, v.typeName)
	}
	return val
}

func (v *objectVal) IsSet(field ref.Val) ref.Val {
	fieldName, ok := field.Value().(string)
	if !ok {
		return types.NewErr("field name must be a string")
	}
	_, present := v.fields[fieldName]
	return types.Bool(present)
}
