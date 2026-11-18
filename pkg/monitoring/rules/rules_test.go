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

package rules_test

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/operator-observability-toolkit/pkg/testutil"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"
)

// namespaceRe matches "namespace" used as a PromQL label name — in label
// matchers, by/on/group_left/group_right clauses — but not as a substring
// of a metric or recording-rule name.
var namespaceRe = regexp.MustCompile(`\bnamespace\b`)

// validateAlertNamespaceLabel checks that every alert has a namespace
// label, either as a static label or derived from its PromQL expression.
func validateAlertNamespaceLabel(alert *promv1.Rule) []testutil.Problem {
	if _, hasNamespace := alert.Labels["namespace"]; hasNamespace {
		return nil
	}
	if namespaceRe.MatchString(alert.Expr.String()) {
		return nil
	}
	return []testutil.Problem{{
		ResourceName: alert.Alert,
		Description: "alert must have a namespace label " +
			"(add a static namespace label or ensure " +
			"the PromQL expression produces one)",
	}}
}

var _ = Describe("Rules Validation", func() {
	var linter *testutil.Linter

	BeforeEach(func() {
		Expect(rules.SetupRules("test-ns")).To(Succeed())
		linter = testutil.New()
	})

	It("Should validate alerts", func() {
		linter.AddCustomAlertValidations(
			testutil.ValidateAlertNameLength,
			testutil.ValidateAlertRunbookURLAnnotation,
			testutil.ValidateAlertHealthImpactLabel,
			testutil.ValidateAlertPartOfAndComponentLabels,
			validateAlertNamespaceLabel)

		problems := linter.LintAlerts(rules.ListAlerts())
		Expect(problems).To(BeEmpty())
	})

	It("Should validate recording rules", func() {
		problems := linter.LintRecordingRules(rules.ListRecordingRules())
		Expect(problems).To(BeEmpty())
	})
})
