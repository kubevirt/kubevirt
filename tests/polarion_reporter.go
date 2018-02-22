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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
)

var Polarion = PolarionReporter{}

func init() {
	flag.BoolVar(&Polarion.Run, "polarion", false, "Run Polarion reporter")
	flag.StringVar(&Polarion.projectId, "polarion-project-id", "", "Set the Polarion project ID")
	flag.StringVar(&Polarion.filename, "polarion-report-file", "polarion.xml", "Set the Polarion report file path")
}

type PolarionTestCases struct {
	XMLName   xml.Name           `xml:"testcases"`
	TestCases []PolarionTestCase `xml:"testcase"`
	ProjectID string             `xml:"project-id,attr"`
}

type PolarionTestCase struct {
	Title                Title                `xml:"title"`
	Description          Description          `xml:"description"`
	TestCaseCustomFields TestCaseCustomFields `xml:"custom-fields"`
}

type Title struct {
	Content string `xml:",chardata"`
}

type Description struct {
	Content string `xml:",chardata"`
}

type TestCaseCustomFields struct {
	CustomFields []TestCaseCustomField `xml:"custom-field"`
}

type TestCaseCustomField struct {
	Content string `xml:"content,attr"`
	ID      string `xml:"id,attr"`
}

type PolarionReporter struct {
	suite         PolarionTestCases
	Run           bool
	filename      string
	projectId     string
	testSuiteName string
}

func (reporter *PolarionReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
	reporter.suite = PolarionTestCases{
		TestCases: []PolarionTestCase{},
	}
	reporter.testSuiteName = summary.SuiteDescription
}

func (reporter *PolarionReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (reporter *PolarionReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (reporter *PolarionReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (reporter *PolarionReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	testName := fmt.Sprintf(
		"%s: %s",
		specSummary.ComponentTexts[1],
		strings.Join(specSummary.ComponentTexts[2:], " "),
	)
	testCase := PolarionTestCase{
		Title:       Title{Content: testName},
		Description: Description{Content: testName},
	}
	customFields := TestCaseCustomFields{}
	customFields.CustomFields = append(customFields.CustomFields, TestCaseCustomField{
		Content: "automated",
		ID:      "caseautomation",
	})
	testCase.TestCaseCustomFields = customFields

	reporter.suite.TestCases = append(reporter.suite.TestCases, testCase)
}

func (reporter *PolarionReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
	if reporter.projectId == "" {
		fmt.Println("Can not create Polarion report without project ID")
		return
	}
	reporter.suite.ProjectID = reporter.projectId
	file, err := os.Create(reporter.filename)
	if err != nil {
		fmt.Printf("Failed to create Polarion report file: %s\n\t%s", reporter.filename, err.Error())
		return
	}
	defer file.Close()
	file.WriteString(xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("  ", "    ")
	err = encoder.Encode(reporter.suite)
	if err != nil {
		fmt.Printf("Failed to generate Polarion report\n\t%s", err.Error())
	}
}
