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

package template

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/klog/v2"
)

const (
	// debugLogLevel is the klog verbosity level for debug messages about non-parameterizable fields
	debugLogLevel = 5
)

type stringTransformer = func(string) (string, bool, error)

// visitValue recursively visits all string fields in the provided value and calls the
// visitor function on them. The visitor function can be used to modify the value of string fields.
func visitValue(val reflect.Value, tf stringTransformer) error {
	// Substitution on nil values is not possible.
	if val.Kind() == reflect.Chan || val.Kind() == reflect.Func || val.Kind() == reflect.Interface ||
		val.Kind() == reflect.Ptr || val.Kind() == reflect.Map || val.Kind() == reflect.Slice {
		if val.IsNil() {
			return nil
		}
	}

	return visitNonNilValue(val, tf)
}

func visitNonNilValue(val reflect.Value, tf stringTransformer) error {
	switch val.Kind() {
	case reflect.Pointer, reflect.Interface:
		return visitValue(val.Elem(), tf)
	case reflect.Slice, reflect.Array:
		return visitSliceArray(val, tf)
	case reflect.Struct:
		return visitStruct(val, tf)
	case reflect.Map:
		return visitMap(val, tf)
	case reflect.String:
		if !val.CanSet() {
			return fmt.Errorf("unable to set String value '%v'", val)
		}
		if s, asString, err := tf(val.String()); err != nil {
			return err
		} else if !asString {
			return fmt.Errorf("attempted to set String field to non-string value '%v'", s)
		} else {
			val.SetString(s)
		}
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		klog.V(debugLogLevel).Infof("Ignoring non-parameterizable field type '%s': %v", val.Kind(), val)
	}

	return nil
}

func visitSliceArray(val reflect.Value, tf stringTransformer) error {
	elementType := val.Type().Elem()
	for i := range val.Len() {
		if newVal, err := visitUnsettableValues(elementType, val.Index(i), tf); err != nil {
			return err
		} else {
			val.Index(i).Set(newVal)
		}
	}

	return nil
}

func visitStruct(val reflect.Value, tf stringTransformer) error {
	for i := range val.NumField() {
		field := val.Field(i)
		// Skip unexported fields as they cannot be set
		if field.Kind() != reflect.Pointer && field.Kind() != reflect.Interface && !field.CanSet() {
			klog.V(debugLogLevel).Infof("Ignoring unexported field '%s'", field.String())
			continue
		}
		if err := visitValue(field, tf); err != nil {
			return err
		}
	}

	return nil
}

func visitMap(val reflect.Value, tf stringTransformer) error {
	valueType := val.Type().Elem()
	lenMapKeys := len(val.MapKeys())
	deletes := make([]reflect.Value, 0, lenMapKeys)
	updates := make(map[any]reflect.Value, lenMapKeys)

	for _, oldKey := range val.MapKeys() {
		newKey, err := visitUnsettableValues(oldKey.Type(), oldKey, tf)
		if err != nil {
			return err
		}
		oldValue := val.MapIndex(oldKey)
		newValue, err := visitUnsettableValues(valueType, oldValue, tf)
		if err != nil {
			return err
		}
		updates[newKey.Interface()] = newValue
		if !reflect.DeepEqual(oldKey.Interface(), newKey.Interface()) {
			deletes = append(deletes, oldKey)
		}
	}

	// Delete old keys first, then add new keys to prevent key collision issues.
	// If a transformed key collides with an existing key in the map, deleting after
	// updating could remove the newly set value instead of the old one.
	for _, k := range deletes {
		val.SetMapIndex(k, reflect.Value{})
	}
	for k, v := range updates {
		val.SetMapIndex(reflect.ValueOf(k), v)
	}

	return nil
}

// visitUnsettableValues creates a copy of the existing value and returns the modified result.
func visitUnsettableValues(typeOf reflect.Type, existing reflect.Value, tf stringTransformer) (reflect.Value, error) {
	val := reflect.New(typeOf).Elem()
	// If the value type is interface, we must resolve it to a concrete value prior to setting it back.
	if existing.CanInterface() {
		existing = reflect.ValueOf(existing.Interface())
	}

	if existing.Kind() == reflect.String {
		if s, asString, err := tf(existing.String()); err != nil {
			return reflect.Value{}, err
		} else if asString {
			val = reflect.ValueOf(s)
		} else {
			var data any
			if err := json.Unmarshal([]byte(s), &data); err != nil {
				// The result of the substitution may have been an unquoted string value,
				// which is an error when decoding in json(only "true", "false", and numeric
				// values can be unquoted), so try wrapping the value in quotes so it will be
				// properly converted to a string type during decoding.
				val = reflect.ValueOf(s)
			} else {
				if data == nil {
					return reflect.Value{}, fmt.Errorf("cannot assign nil value to target type %v", typeOf)
				}
				if !reflect.TypeOf(data).AssignableTo(typeOf) {
					return reflect.Value{}, fmt.Errorf("substituted value type %T is not assignable to target type %v", data, typeOf)
				}
				val = reflect.ValueOf(data)
			}
		}

		return val, nil
	}

	if existing.IsValid() && existing.Kind() != reflect.Invalid {
		val.Set(existing)
	}
	if err := visitValue(val, tf); err != nil {
		return reflect.Value{}, err
	}

	return val, nil
}
