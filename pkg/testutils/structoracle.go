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

package testutils

import (
	"reflect"
	"strings"
)

// WithAllFieldsSet returns a pointer to a new instance of the type described by t
// with every pointer field set to a non-nil zero value, including nested structs.
// It is intended for use in tests that need a fully-populated struct to exercise
// field-coverage assertions.
func WithAllFieldsSet(t reflect.Type) interface{} {
	v := reflect.New(t).Elem()
	populateAllPointerFields(v)
	return v.Addr().Interface()
}

// populateAllPointerFields recursively initializes every nil pointer in v,
// descending into struct fields to cover nested types.
func populateAllPointerFields(v reflect.Value) {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		populateAllPointerFields(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanSet() {
				continue
			}
			if f.Kind() == reflect.Ptr || f.Kind() == reflect.Struct {
				populateAllPointerFields(f)
			}
		}
	}
}

// CopyByJSONTag copies fields from src (a pointer to a struct) into a new instance
// of dstType by matching JSON tag names, including nested structs. It is the
// reflection-based oracle for manual mapping functions whose contract is to transfer
// all fields that share a JSON tag name between two types. Nil pointer fields in
// src are skipped (the corresponding dst field retains its zero value).
func CopyByJSONTag(src interface{}, dstType reflect.Type) interface{} {
	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	dst := reflect.New(dstType)
	copyStructByJSONTag(srcVal, dst.Elem())
	return dst.Interface()
}

// copyStructByJSONTag copies fields from src to dst by matching JSON tag names,
// recursing into nested struct types.
func copyStructByJSONTag(src, dst reflect.Value) {
	dstFieldsByTag := make(map[string]int)
	for i := 0; i < dst.NumField(); i++ {
		if tag, _, _ := strings.Cut(dst.Type().Field(i).Tag.Get("json"), ","); tag != "" && tag != "-" {
			dstFieldsByTag[tag] = i
		}
	}

	for i := 0; i < src.NumField(); i++ {
		tag, _, _ := strings.Cut(src.Type().Field(i).Tag.Get("json"), ",")
		if tag == "" || tag == "-" {
			continue
		}
		di, ok := dstFieldsByTag[tag]
		if !ok {
			continue
		}

		srcField, dstField := src.Field(i), dst.Field(di)

		if srcField.Kind() == reflect.Ptr {
			if srcField.IsNil() {
				continue
			}
			if dstField.IsNil() {
				dstField.Set(reflect.New(dstField.Type().Elem()))
			}
			if srcField.Type().Elem().Kind() == reflect.Struct {
				copyStructByJSONTag(srcField.Elem(), dstField.Elem())
				continue
			}
		}

		if srcField.Kind() == reflect.Struct && dstField.Kind() == reflect.Struct {
			copyStructByJSONTag(srcField, dstField)
			continue
		}

		dstField.Set(srcField)
	}
}
