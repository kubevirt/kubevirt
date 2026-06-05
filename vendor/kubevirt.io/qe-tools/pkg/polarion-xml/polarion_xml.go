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

package polarion_xml

import (
	"encoding/xml"
	"fmt"
	"os"
)

type TestCases struct {
	Properties PolarionProperties `xml:"properties"`
	XMLName    xml.Name           `xml:"testcases"`
	TestCases  []TestCase         `xml:"testcase"`
	ProjectID  string             `xml:"project-id,attr"`
}

type TestCase struct {
	ID                      string                  `xml:"id,attr,omitempty"`
	Title                   Title                   `xml:"title"`
	Description             Description             `xml:"description"`
	TestCaseCustomFields    TestCaseCustomFields    `xml:"custom-fields"`
	TestCaseSteps           *TestCaseSteps          `xml:"test-steps,omitempty"`
	TestCaseLinkedWorkItems TestCaseLinkedWorkItems `xml:"linked-work-items"`
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

type TestCaseSteps struct {
	Steps []TestCaseStep `xml:"test-step"`
}

type TestCaseStep struct {
	StepColumn []TestCaseStepColumn `xml:"test-step-column"`
}

type TestCaseStepColumn struct {
	Content string `xml:",chardata"`
	ID      string `xml:"id,attr"`
}

type PolarionProperties struct {
	Property []PolarionProperty `xml:"property"`
}

type PolarionProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type TestCaseLinkedWorkItems struct {
	LinkedWorkItems []TestCaseLinkedWorkItem `xml:"linked-work-item"`
}

type TestCaseLinkedWorkItem struct {
	ID   string `xml:"workitem-id,attr"`
	Role string `xml:"role-id,attr"`
}

func GeneratePolarionXmlFile(outputFile string, testCases interface{}) {
	file, err := os.Create(outputFile)
	if err != nil {
		panic(fmt.Errorf("Failed to create Polarion report file: %s\n\t%s", outputFile, err.Error()))
	}
	defer file.Close()
	file.WriteString(xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("  ", "    ")
	err = encoder.Encode(testCases)
	if err != nil {
		panic(fmt.Errorf("Failed to generate Polarion report\n\t%s", err.Error()))
	}
}
