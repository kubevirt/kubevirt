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

package matcher

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

func Exist() types.GomegaMatcher {
	return existMatcher{}
}

func BeGone() types.GomegaMatcher {
	return goneMatcher{}
}

type existMatcher struct {
}

func (e existMatcher) Match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return false, nil
	}
	return true, nil
}

func formatObject(actual interface{}) string {
	if helper.IsNil(actual) {
		return fmt.Sprintf("%v", actual)
	}
	if !helper.IsStruct(helper.DeferPointer(actual)) {
		return fmt.Sprintf("%v", actual)
	}
	obj := reflect.ValueOf(helper.ToPointer(actual)).Elem()
	metadata := obj.FieldByName("ObjectMeta")
	if metadata.IsZero() {
		return fmt.Sprintf("%v", actual)
	}

	// Optional
	status := obj.FieldByName("Status")

	// Too much data to display and is only helpful in later stages
	metadata.FieldByName("ManagedFields").SetZero()

	return fmt.Sprintf("%s\nmetadata: %s \nstatus: %s", reflect.TypeOf(actual), format.Object(metadata.Interface(), 0), format.Object(status.Interface(), 0))
}

func (e existMatcher) FailureMessage(actual interface{}) (message string) {

	return fmt.Sprintf("expected object to still exist, but it is gone: %s", formatObject(actual))
}

func (e existMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected object to be gone, but it still exists: %s", formatObject(actual))
}

type goneMatcher struct {
}

func (g goneMatcher) Match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return true, nil
	}
	if helper.IsSlice(actual) && reflect.ValueOf(actual).Len() == 0 {
		return true, nil
	}
	return false, nil
}

func (g goneMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected object to be gone, but it still exists: %s", formatObject(actual))
}

func (g goneMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected object to still exist, but it is gone: %s", formatObject(actual))
}
