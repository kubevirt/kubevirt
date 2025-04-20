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

	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

func BeRunning() types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: "Running",
	}
}

func HaveSucceeded() types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: "Succeeded",
	}
}

func PendingPopulation() types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: "PendingPopulation",
	}
}

func WaitForFirstConsumer() types.GomegaMatcher {
	return Or(BeInPhase("WaitForFirstConsumer"), PendingPopulation())
}

func BeInPhase(phase interface{}) types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: phase,
	}
}

type phaseMatcher struct {
	expectedPhase interface{}
}

func (p phaseMatcher) Match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return false, nil
	}
	if helper.IsSlice(actual) {
		return helper.MatchElementsInSlice(actual, p.match)
	}
	return p.match(actual)
}

func (p phaseMatcher) match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return false, nil
	}
	phase, err := getCurrentPhase(actual)
	if err != nil {
		return false, err
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	if phase == expectedPhase {
		return true, nil
	}
	return false, nil
}

func (p phaseMatcher) FailureMessage(actual interface{}) (message string) {
	if helper.IsNil(actual) {
		return fmt.Sprintf("object does not exist")
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	if helper.IsSlice(actual) {
		return fmt.Sprintf("expected phases to be in %v but got %v", expectedPhase, collectPhasesForPrinting(actual))
	}

	objectInfo, err := getObjectKindAndName(actual)
	if err != nil {
		return err.Error()
	}
	phase, err := getCurrentPhase(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s expected phase is '%v' but got '%v'", objectInfo, expectedPhase, phase)
}

func (p phaseMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	if helper.IsNil(actual) {
		return fmt.Sprintf("object does not exist")
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	if helper.IsSlice(actual) {
		return fmt.Sprintf("expected phases to not be in %v but got %v", expectedPhase, collectPhasesForPrinting(actual))
	}
	objectInfo, err := getObjectKindAndName(actual)
	if err != nil {
		return err.Error()
	}
	phase, err := getCurrentPhase(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s expected phase '%v' to not match '%v'", objectInfo, expectedPhase, phase)
}

func getObjectKindAndName(actual interface{}) (string, error) {
	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return "", err
	}
	objectInfo := ""
	if u != nil {
		if u.GetKind() != "" {
			objectInfo = fmt.Sprintf("%s/", u.GetKind())
		}
		if u.GetName() != "" {
			objectInfo = fmt.Sprintf("%s%s", objectInfo, u.GetName())
		}
	}
	return objectInfo, nil
}

func getCurrentPhase(actual interface{}) (string, error) {
	actual = helper.ToPointer(actual)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(actual)
	if err != nil {
		return "", err
	}
	str, _, err := unstructured.NestedString(obj, "status", "phase")
	return str, err
}

func collectPhasesForPrinting(actual interface{}) (phases []string) {
	helper.IterateOverSlice(actual, func(value interface{}) bool {
		if helper.IsNil(value) {
			phases = append(phases, "nil")
			return true
		}
		phase, err := getCurrentPhase(value)
		if err != nil {
			phase = err.Error()
		}
		objectInfo, err := getObjectKindAndName(value)
		if err != nil {
			phase = err.Error()
		}
		if objectInfo != "" {
			phase = fmt.Sprintf("%s:%s", objectInfo, phase)
		}

		phases = append(phases, phase)
		return true
	})
	return
}

func getExpectedPhase(actual interface{}) string {
	return reflect.ValueOf(actual).String()
}

func HavePrintableStatus(status v1.VirtualMachinePrintableStatus) types.GomegaMatcher {
	return PointTo(MatchFields(IgnoreExtras, Fields{
		"Status": MatchFields(IgnoreExtras, Fields{
			"PrintableStatus": Equal(status),
		}),
	}))
}
