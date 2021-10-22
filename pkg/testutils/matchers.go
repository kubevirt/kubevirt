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
	"strings"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

func HaveStatusCode(expected interface{}) types.GomegaMatcher {
	return &haveStatusCodeMatcher{
		expected: expected,
	}
}

type haveStatusCodeMatcher struct {
	expected   interface{}
	statusCode int
}

func (matcher *haveStatusCodeMatcher) Match(actual interface{}) (success bool, err error) {
	result, ok := actual.(rest.Result)
	if !ok {
		return false, fmt.Errorf("HaveStatusCode matcher expects a kubernetes rest client Result")
	}

	expectedStatusCode, ok := matcher.expected.(int)
	if !ok {
		return false, fmt.Errorf("Expected status code to be of type int")
	}

	result.StatusCode(&matcher.statusCode)

	if result.Error() != nil {
		matcher.statusCode = int(result.Error().(*errors.StatusError).Status().Code)
	}

	return matcher.statusCode == expectedStatusCode, nil
}

func (matcher *haveStatusCodeMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected status code \n\t%#v\not to be\n\t%#v", matcher.statusCode, matcher.expected)
}

func (matcher *haveStatusCodeMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected status code \n\t%#v\nnot to be\n\t%#v", matcher.statusCode, matcher.expected)
}

// In case we don't care about emitted events, we simply consume all of them and return.
func IgnoreEvents(recorder *record.FakeRecorder) {
loop:
	for {
		select {
		case <-recorder.Events:
		default:
			break loop
		}
	}
}

func ExpectEvent(recorder *record.FakeRecorder, reason string) {
	gomega.ExpectWithOffset(1, recorder.Events).To(gomega.Receive(gomega.ContainSubstring(reason)))
}

// ExpectEvents checks for given reasons in arbitrary order
func ExpectEvents(recorder *record.FakeRecorder, reasons ...string) {
	l := len(reasons)
	for x := 0; x < l; x++ {
		select {

		case e := <-recorder.Events:
			filtered := []string{}
			found := false
			for _, reason := range reasons {

				if strings.Contains(e, reason) && !found {
					found = true
					continue
				}
				filtered = append(filtered, reason)
			}

			gomega.ExpectWithOffset(1, found).To(gomega.BeTrue(), "Expected to match event reason '%s' with one of %v", e, reasons)
			reasons = filtered

		default:
			// There should be something, trigger an error
			gomega.ExpectWithOffset(1, recorder.Events).To(gomega.Receive())
		}
	}
}

func SatisfyAnyRegexp(regexps []string) types.GomegaMatcher {
	matchers := []types.GomegaMatcher{}
	for _, regexp := range regexps {
		matchers = append(matchers, gomega.MatchRegexp(regexp))
	}
	return gomega.SatisfyAny(matchers...)
}
