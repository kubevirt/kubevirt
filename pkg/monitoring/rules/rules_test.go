/*
Copyright 2024 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rules

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/testutil"
)

var _ = Describe("Rules Validation", func() {
	var linter *testutil.Linter

	BeforeEach(func() {
		Expect(SetupRules("")).To(Succeed())
		linter = testutil.New()
	})

	It("Should validate alerts", func() {
		linter.AddCustomAlertValidations(
			testutil.ValidateAlertNameLength,
			testutil.ValidateAlertRunbookURLAnnotation,
			testutil.ValidateAlertHealthImpactLabel,
			testutil.ValidateAlertPartOfAndComponentLabels)

		alerts := ListAlerts()
		problems := linter.LintAlerts(alerts)
		Expect(problems).To(BeEmpty())
	})

	It("Should validate recording rules", func() {
		recordingRules := ListRecordingRules()
		problems := linter.LintRecordingRules(recordingRules)
		Expect(problems).To(BeEmpty())
	})
})
