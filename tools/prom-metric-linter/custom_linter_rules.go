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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil/promlint"
	dto "github.com/prometheus/client_model/go"
)

func CustomLinterRules(problems []promlint.Problem, mf *dto.MetricFamily, operatorName string, subOperatorName string) []promlint.Problem {
	// Check metric prefix
	nameParts := strings.Split(*mf.Name, "_")

	if nameParts[0] != operatorName {
		problems = append(problems, promlint.Problem{
			Metric: *mf.Name,
			Text:   fmt.Sprintf("name need to start with '%s_'", operatorName),
		})
	} else if operatorName != subOperatorName && nameParts[1] != subOperatorName {
		problems = append(problems, promlint.Problem{
			Metric: *mf.Name,
			Text:   fmt.Sprintf("name need to start with \"%s_%s_\"", operatorName, subOperatorName),
		})
	}

	// Check "_timestamp_seconds" suffix for non-counter metrics
	if *mf.Type != dto.MetricType_COUNTER && strings.HasSuffix(*mf.Name, "_timestamp_seconds") {
		problems = append(problems, promlint.Problem{
			Metric: *mf.Name,
			Text:   "non-counter metric should not have \"_timestamp_seconds\" suffix",
		})
	}

	// If promlint fails on a "total" suffix, check also for "_timestamp_seconds" suffix. If it exists, do not fail
	var newProblems []promlint.Problem
	for _, problem := range problems {
		if strings.Contains(problem.Text, "counter metrics should have \"_total\" suffix") {
			if !strings.HasSuffix(problem.Metric, "_timestamp_seconds") {
				problem.Text = "counter metrics should have \"_total\" or \"_timestamp_seconds\" suffix"
				newProblems = append(newProblems, problem)
			}
		} else {
			newProblems = append(newProblems, problem)
		}
	}

	// Sort the problems by metric name
	sort.Slice(newProblems, func(i, j int) bool {
		return newProblems[i].Metric < newProblems[j].Metric
	})

	return newProblems
}
