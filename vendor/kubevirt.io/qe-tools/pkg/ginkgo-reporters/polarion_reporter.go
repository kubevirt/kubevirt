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

package ginkgo_reporters

import (
	"encoding/xml"
	"flag"
	"fmt"
	"strings"

	"kubevirt.io/qe-tools/pkg/polarion-xml"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
)

var Polarion = PolarionReporter{}

func init() {
	flag.BoolVar(&Polarion.Run, "polarion-execution", false, "Run Polarion reporter")
	flag.StringVar(&Polarion.ProjectId, "polarion-project-id", "", "Set Polarion project ID")
	flag.StringVar(&Polarion.Filename, "polarion-report-file", "polarion_results.xml", "Set Polarion report file path")
	flag.StringVar(&Polarion.PlannedIn, "polarion-custom-plannedin", "", "Set Polarion planned-in ID")
	flag.StringVar(&Polarion.Tier, "test-tier", "", "Set test tier number")
}

type PolarionTestSuite struct {
	XMLName    xml.Name           `xml:"testsuite"`
	Tests      int                `xml:"tests,attr"`
	Failures   int                `xml:"failures,attr"`
	Time       float64            `xml:"time,attr"`
	Properties PolarionProperties `xml:"properties"`
	TestCases  []PolarionTestCase `xml:"testcase"`
}

type PolarionTestCase struct {
	Name           string               `xml:"name,attr"`
	FailureMessage *JUnitFailureMessage `xml:"failure,omitempty"`
	Skipped        *JUnitSkipped        `xml:"skipped,omitempty"`
	SystemOut      string               `xml:"system-out,omitempty"`
}

type JUnitFailureMessage struct {
	Type    string `xml:"type,attr"`
	Message string `xml:",chardata"`
}

type JUnitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
}

type PolarionProperties struct {
	Property []PolarionProperty `xml:"property"`
}

type PolarionProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type PolarionReporter struct {
	Suite         PolarionTestSuite
	Run           bool
	Filename      string
	TestSuiteName string
	ProjectId     string
	PlannedIn     string
	Tier          string
}

func (reporter *PolarionReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {

	reporter.Suite = PolarionTestSuite{
		Properties: PolarionProperties{},
		TestCases:  []PolarionTestCase{},
	}

	properties := PolarionProperties{
		Property: []PolarionProperty{
			{
				Name:  "polarion-project-id",
				Value: reporter.ProjectId,
			},
			{
				Name:  "polarion-lookup-method",
				Value: "name",
			},
			{
				Name:  "polarion-custom-plannedin",
				Value: reporter.PlannedIn,
			},
			{
				Name:  "polarion-testrun-id",
				Value: reporter.PlannedIn + "_" + reporter.Tier,
			},
			{
				Name:  "polarion-custom-isautomated",
				Value: "True",
			},
		},
	}

	reporter.Suite.Properties = properties
	reporter.TestSuiteName = summary.SuiteDescription
}

func (reporter *PolarionReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (reporter *PolarionReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (reporter *PolarionReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
}

func failureMessage(failure types.SpecFailure) string {
	return fmt.Sprintf("%s\n%s\n%s", failure.ComponentCodeLocation.String(), failure.Message, failure.Location.String())
}

func (reporter *PolarionReporter) handleSetupSummary(name string, setupSummary *types.SetupSummary) {
	if setupSummary.State != types.SpecStatePassed {
		testCase := PolarionTestCase{
			Name: name,
		}

		testCase.FailureMessage = &JUnitFailureMessage{
			Type:    reporter.failureTypeForState(setupSummary.State),
			Message: failureMessage(setupSummary.Failure),
		}
		testCase.SystemOut = setupSummary.CapturedOutput
		reporter.Suite.TestCases = append(reporter.Suite.TestCases, testCase)
	}
}

func (reporter *PolarionReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	testName := fmt.Sprintf(
		"%s: %s",
		specSummary.ComponentTexts[1],
		strings.Join(specSummary.ComponentTexts[2:], " "),
	)
	testCase := PolarionTestCase{
		Name: testName,
	}
	if specSummary.State == types.SpecStateFailed || specSummary.State == types.SpecStateTimedOut || specSummary.State == types.SpecStatePanicked {
		testCase.FailureMessage = &JUnitFailureMessage{
			Type:    reporter.failureTypeForState(specSummary.State),
			Message: failureMessage(specSummary.Failure),
		}
		testCase.SystemOut = specSummary.CapturedOutput
	}
	if specSummary.State == types.SpecStateSkipped || specSummary.State == types.SpecStatePending {
		testCase.Skipped = &JUnitSkipped{}
	}
	reporter.Suite.TestCases = append(reporter.Suite.TestCases, testCase)
}

func (reporter *PolarionReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
	if reporter.ProjectId == "" {
		fmt.Println("Can not create Polarion report without project ID")
		return
	}
	if reporter.PlannedIn == "" {
		fmt.Println("Can not create Polarion report without planned-in ID")
		return
	}
	if reporter.Tier == "" {
		fmt.Println("Can not create Polarion report without tier ID")
		return
	}

	reporter.Suite.Tests = summary.NumberOfSpecsThatWillBeRun
	reporter.Suite.Time = summary.RunTime.Seconds()
	reporter.Suite.Failures = summary.NumberOfFailedSpecs

	// generate polarion test cases XML file
	polarion_xml.GeneratePolarionXmlFile(reporter.Filename, reporter.Suite)

}

func (reporter *PolarionReporter) failureTypeForState(state types.SpecState) string {
	switch state {
	case types.SpecStateFailed:
		return "Failure"
	case types.SpecStateTimedOut:
		return "Timeout"
	case types.SpecStatePanicked:
		return "Panic"
	default:
		return ""
	}
}
