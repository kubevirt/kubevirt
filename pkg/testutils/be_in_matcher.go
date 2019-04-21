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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package testutils

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/matchers"
)

func isArrayOrSlice(a interface{}) bool {
	if a == nil {
		return false
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.Array, reflect.Slice:
		return true
	default:
		return false
	}
}

func isMap(a interface{}) bool {
	if a == nil {
		return false
	}
	return reflect.TypeOf(a).Kind() == reflect.Map
}

type BeInMatcher struct {
	Elements []interface{}
}

func (matcher *BeInMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, fmt.Errorf("BeIn matcher expects a type.")
	}

	length := len(matcher.Elements)
	valueAt := func(i int) interface{} {
		return matcher.Elements[i]
	}
	if length == 1 {
		element := matcher.Elements[0]
		switch {
		case isMap(element):
			value := reflect.ValueOf(element)
			length = value.Len()
			keys := value.MapKeys()
			valueAt = func(i int) interface{} {
				return value.MapIndex(keys[i]).Interface()
			}
		case isArrayOrSlice(element):
			value := reflect.ValueOf(element)
			length = value.Len()
			valueAt = func(i int) interface{} {
				return value.Index(i).Interface()
			}
		}
	}

	var lastError error
	for i := 0; i < length; i++ {
		matcher := &matchers.EqualMatcher{Expected: valueAt(i)}
		success, err := matcher.Match(actual)
		if err != nil {
			lastError = err
			continue
		}
		if success {
			return true, nil
		}
	}

	return false, lastError
}

func (matcher *BeInMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be in", matcher.Elements)
}

func (matcher *BeInMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be in", matcher.Elements)
}
