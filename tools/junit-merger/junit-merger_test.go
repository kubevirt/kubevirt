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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package main

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	. "github.com/onsi/gomega"
	"testing"
)

func TestJunitMerger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "junit-merger_test.go")
}

var _ = Describe("junit-merger", func() {

	When("merging", func() {

		It("fails on same test names", func() {
			suites := []reporters.JUnitTestSuite{
				{
					Name:       "SomeRidiculousSuite1",
					Package:    "",
					Tests:      0,
					Disabled:   0,
					Skipped:    0,
					Errors:     0,
					Failures:   0,
					Time:       0,
					Timestamp:  "",
					Properties: reporters.JUnitProperties{},
					TestCases: []reporters.JUnitTestCase{
						{
							Name:      "SomeRidiculousTestName",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped:   nil,
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
					},
				},
				{
					Name:       "SomeRidiculousSuite2",
					Package:    "",
					Tests:      0,
					Disabled:   0,
					Skipped:    0,
					Errors:     0,
					Failures:   0,
					Time:       0,
					Timestamp:  "",
					Properties: reporters.JUnitProperties{},
					TestCases: []reporters.JUnitTestCase{
						{
							Name:      "SomeRidiculousTestName",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped:   nil,
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
					},
				},
			}
			_, err := mergeJUnitFiles(suites)
			Expect(err).ToNot(BeNil())
		})

		It("doesn't fail on different test names", func() {
			suites := []reporters.JUnitTestSuite{
				{
					Name:       "SomeRidiculousSuite1",
					Package:    "",
					Tests:      0,
					Disabled:   0,
					Skipped:    0,
					Errors:     0,
					Failures:   0,
					Time:       0,
					Timestamp:  "",
					Properties: reporters.JUnitProperties{},
					TestCases: []reporters.JUnitTestCase{
						{
							Name:      "SomeRidiculousTestName1",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped:   nil,
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
					},
				},
				{
					Name:       "SomeRidiculousSuite2",
					Package:    "",
					Tests:      0,
					Disabled:   0,
					Skipped:    0,
					Errors:     0,
					Failures:   0,
					Time:       0,
					Timestamp:  "",
					Properties: reporters.JUnitProperties{},
					TestCases: []reporters.JUnitTestCase{
						{
							Name:      "SomeRidiculousTestName2",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped:   nil,
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
					},
				},
			}
			_, err := mergeJUnitFiles(suites)
			Expect(err).To(BeNil())
		})

		It("does keep different skipped tests if they are only seen once", func() {
			suites := []reporters.JUnitTestSuite{
				{
					Name:       "SomeRidiculousSuite1",
					Package:    "",
					Tests:      0,
					Disabled:   0,
					Skipped:    0,
					Errors:     0,
					Failures:   0,
					Time:       0,
					Timestamp:  "",
					Properties: reporters.JUnitProperties{},
					TestCases: []reporters.JUnitTestCase{
						{
							Name:      "SomeRidiculousTestNameSkipped1",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped: &reporters.JUnitSkipped{
								Message: "Skipped",
							},
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
					},
				},
				{
					Name:       "SomeRidiculousSuite2",
					Package:    "",
					Tests:      0,
					Disabled:   0,
					Skipped:    0,
					Errors:     0,
					Failures:   0,
					Time:       0,
					Timestamp:  "",
					Properties: reporters.JUnitProperties{},
					TestCases: []reporters.JUnitTestCase{
						{
							Name:      "SomeRidiculousTestNameSkipped2",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped: &reporters.JUnitSkipped{
								Message: "Skipped",
							},
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
					},
				},
			}
			result, err := mergeJUnitFiles(suites)
			Expect(err).To(BeNil())
			Expect(result).To(BeEquivalentTo(
				reporters.JUnitTestSuite{
					Name:       "Merged Test Suite",
					Package:    "",
					Tests:      0,
					Disabled:   0,
					Skipped:    0,
					Errors:     0,
					Failures:   0,
					Time:       0,
					Timestamp:  "",
					Properties: reporters.JUnitProperties{},
					TestCases: []reporters.JUnitTestCase{
						{
							Name:      "SomeRidiculousTestNameSkipped1",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped: &reporters.JUnitSkipped{
								Message: "Skipped",
							},
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
						{
							Name:      "SomeRidiculousTestNameSkipped2",
							Classname: "",
							Status:    "",
							Time:      0,
							Skipped: &reporters.JUnitSkipped{
								Message: "Skipped",
							},
							Error:     nil,
							Failure:   nil,
							SystemOut: "",
							SystemErr: "",
						},
					},
				}))
		})

	})

})
