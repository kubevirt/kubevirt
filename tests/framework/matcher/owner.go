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

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"

	"github.com/onsi/gomega/types"
)

func BeOwned() types.GomegaMatcher {
	return ownedMatcher{}
}

func HaveOwners() types.GomegaMatcher {
	return ownedMatcher{}
}

type ownedMatcher struct {
}

func (o ownedMatcher) Match(actual interface{}) (success bool, err error) {
	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return false, nil
	}
	if len(u.GetOwnerReferences()) > 0 {
		return true, nil
	}
	return false, nil
}

func (o ownedMatcher) FailureMessage(actual interface{}) (message string) {
	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected owner references to not be empty, but got '%v'", u)
}

func (o ownedMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected owner references to be empty, but got '%v'", u)
}
