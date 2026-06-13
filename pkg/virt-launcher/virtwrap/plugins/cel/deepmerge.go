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

// deepmerge.go converts sparse objectVal results back into Go structs. After CEL
// evaluates a mutation expression, the result is an objectVal with only the fields
// the plugin set. deepMerge walks this objectVal and applies each field onto a copy
// of the original domain, leaving all other fields untouched. Nested objectVals are
// merged recursively; slices are replaced wholesale.

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// deepMerge applies the explicitly-set fields from a sparse objectVal onto a
// Go struct. Only fields present in the objectVal's map are written - all other
// fields in the base struct are left untouched.
func deepMerge(base reflect.Value, partial *objectVal) error {
	if base.Kind() == reflect.Ptr {
		if base.IsNil() {
			base.Set(reflect.New(base.Type().Elem()))
		}
		base = base.Elem()
	}

	typeName := base.Type().Name()

	for fieldName, val := range partial.fields {
		field := base.FieldByName(fieldName)
		if !field.IsValid() || !field.CanSet() {
			return fmt.Errorf("cannot set field %s on %s", fieldName, typeName)
		}
		if err := setField(field, val); err != nil {
			return fmt.Errorf("setting field %s on %s: %w", fieldName, typeName, err)
		}
	}
	return nil
}

func setField(field reflect.Value, val ref.Val) error {
	if val == nil || val == types.NullValue {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	if ov, ok := val.(*objectVal); ok {
		target := field
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			target = field.Elem()
		}
		return deepMerge(target, ov)
	}

	ft := field.Type()
	isPtr := ft.Kind() == reflect.Ptr
	if isPtr {
		ft = ft.Elem()
	}

	if ft.Kind() == reflect.Slice {
		return setSliceField(field, val)
	}

	rv, err := refValToReflect(val, ft)
	if err != nil {
		return err
	}

	if isPtr {
		ptr := reflect.New(ft)
		ptr.Elem().Set(rv)
		field.Set(ptr)
	} else {
		field.Set(rv)
	}
	return nil
}

func refValToReflect(val ref.Val, targetType reflect.Type) (reflect.Value, error) {
	native, err := val.ConvertToNative(targetType)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("converting to %v: %w", targetType, err)
	}
	return reflect.ValueOf(native), nil
}

func setSliceField(field reflect.Value, val ref.Val) error {
	iter, ok := val.(traits.Iterable)
	if !ok {
		return fmt.Errorf("list value does not support iteration")
	}
	it := iter.Iterator()

	ft := field.Type()
	elemType := ft.Elem()
	isElemPtr := elemType.Kind() == reflect.Ptr
	baseElemType := elemType
	if isElemPtr {
		baseElemType = elemType.Elem()
	}

	var elems []reflect.Value
	for it.HasNext() == types.True {
		elemVal := it.Next()
		var rv reflect.Value
		if ov, ok := elemVal.(*objectVal); ok {
			newElem := reflect.New(baseElemType).Elem()
			if err := deepMerge(newElem, ov); err != nil {
				return fmt.Errorf("merging list element: %w", err)
			}
			rv = newElem
		} else {
			var err error
			rv, err = refValToReflect(elemVal, baseElemType)
			if err != nil {
				return fmt.Errorf("converting list element: %w", err)
			}
		}

		if isElemPtr {
			ptr := reflect.New(baseElemType)
			ptr.Elem().Set(rv)
			elems = append(elems, ptr)
		} else {
			elems = append(elems, rv)
		}
	}

	slice := reflect.MakeSlice(ft, len(elems), len(elems))
	for i, elem := range elems {
		slice.Index(i).Set(elem)
	}
	field.Set(slice)
	return nil
}
