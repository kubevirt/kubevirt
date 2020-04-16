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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package nodelabeller

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Node-labeller config", func() {

	It("should return obsolete cpus", func() {
		var testCases = []struct {
			conf   Config
			result map[string]bool
		}{
			{
				conf: Config{
					ObsoleteCPUs: []string{"Conroe", "Haswell", "Penryn"},
				},
				result: map[string]bool{
					"Conroe":  true,
					"Haswell": true,
					"Penryn":  true,
				},
			},
			{
				conf: Config{
					ObsoleteCPUs: []string{},
				},
				result: map[string]bool{},
			},
		}
		for _, testCase := range testCases {
			m := testCase.conf.getObsoleteCPUMap()
			Expect(m).To(Equal(testCase.result), "obsolete cpus are not equal")
		}
	})

	It("should return min cpu", func() {
		var testCases = []struct {
			conf     Config
			provider string
			result   string
		}{
			{
				conf: Config{
					MinCPU: "Penryn",
				},
				result: "Penryn",
			},
			{
				conf: Config{
					MinCPU: "",
				},
				result: "",
			},
		}
		for _, testCase := range testCases {
			result := testCase.conf.getMinCPU()
			Expect(result).To(Equal(testCase.result), "minCPU is not equal")
		}
	})

})
