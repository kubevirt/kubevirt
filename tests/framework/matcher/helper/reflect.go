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
 */

package helper

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func IsNil(actual interface{}) bool {
	return actual == nil || (reflect.ValueOf(actual).Kind() == reflect.Ptr && reflect.ValueOf(actual).IsNil())
}

func IsSlice(actual interface{}) bool {
	val := reflect.ValueOf(actual)
	return val.Kind() == reflect.Array || val.Kind() == reflect.Slice
}

func IsStruct(actual interface{}) bool {
	val := reflect.ValueOf(actual)
	return val.Kind() == reflect.Struct
}

// ToPointer returns a new pointer to the provided interface{}, if the
// provided value is not already a pointer. if the original value is already a pointer it gets
// returned directly.
func ToPointer(actual interface{}) interface{} {
	if reflect.ValueOf(actual).Kind() != reflect.Ptr {
		p := reflect.New(reflect.TypeOf(actual))
		p.Elem().Set(reflect.ValueOf(actual))
		actual = p.Interface()
	}
	return actual
}

func DeferPointer(actual interface{}) interface{} {
	value := reflect.ValueOf(actual)
	if value.Kind() == reflect.Ptr {
		actual = value.Elem()
	}
	return actual
}

// IterateOverSlice iterates over the provides interface{} until all elements were visited
// or until the visior returns false
func IterateOverSlice(actual interface{}, visitor func(value interface{}) bool) {
	val := reflect.ValueOf(actual)
	for x := 0; x < val.Len(); x++ {
		if !visitor(val.Index(x).Interface()) {
			break
		}
	}
}

// MatchElementsInSlice applies a matcher individually to each element in the slice and returns as
// soon as the matcher fails on an element.
func MatchElementsInSlice(actual interface{}, matcher func(actual interface{}) (success bool, err error)) (bool, error) {
	var success bool
	var err error
	IterateOverSlice(actual, func(value interface{}) bool {
		success, err = matcher(value)
		return success
	})
	return success, err
}

func ToUnstructured(actual interface{}) (*unstructured.Unstructured, error) {
	if IsNil(actual) {
		return nil, fmt.Errorf("object does not exist")
	}
	actual = ToPointer(actual)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(actual)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}
