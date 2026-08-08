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
	"fmt"
	"reflect"
	"strings"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/randfill"
)

const withAllFieldsSetMaxSeedAttempts = 50

// WithAllFieldsSet returns a pointer to a new instance of the type described by t
// with every field populated with random non-default values, including nested
// structs. It is intended for use in tests that need a fully-populated struct to
// exercise field-coverage assertions.
func WithAllFieldsSet(t reflect.Type) interface{} {
	// Seeds are deterministic: this loop asserts that structRandfill's Funcs
	// produce all-non-default values, panicking if they don't.
	for seed := int64(0); seed < withAllFieldsSetMaxSeedAttempts; seed++ {
		v := reflect.New(t).Elem()
		structRandfill(seed).Fill(v.Addr().Interface())
		if areAllFieldsNonDefault(v) {
			return v.Addr().Interface()
		}
	}
	// if you hit this panic, you likely need to add a custom fill function
	// for the type in structRandfill that returns a non-default value for the type
	panic(fmt.Sprintf("testutils.WithAllFieldsSet: could not populate non-default values for %v", t))
}

func structRandfill(seed int64) *randfill.Filler {
	return randfill.NewWithSeed(seed).NilChance(0).Funcs(
		func(b *bool, c randfill.Continue) { *b = true },
		func(t **metav1.Time, c randfill.Continue) {
			if *t == nil {
				*t = &metav1.Time{}
			}
			(*t).RandFill(c.Rand)
		},
		func(q **resource.Quantity, c randfill.Continue) {
			*q = resource.NewQuantity(1, resource.DecimalSI)
		},
	)
}

func areAllFieldsNonDefault(v reflect.Value) bool {
	zero := reflect.Zero(v.Type())
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).PkgPath != "" {
			continue
		}
		if apiequality.Semantic.DeepEqual(v.Field(i).Interface(), zero.Field(i).Interface()) {
			return false
		}
	}
	return true
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

		if srcField.Type() == dstField.Type() {
			dstField.Set(srcField)
			continue
		}

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
