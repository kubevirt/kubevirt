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
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ginkgoMetadata holds useful bits of information for every entry in the outline
type ginkgoMetadata struct {
	// Name is the spec or container function name, e.g. `Describe` or `It`
	Name string `json:"name"`

	// Text is the `text` argument passed to specs, and some containers
	Text string `json:"text"`

	// Start is the position of first character of the spec or container block
	Start int `json:"start"`

	// End is the position of first character immediately after the spec or container block
	End int `json:"end"`

	Spec    bool `json:"spec"`
	Focused bool `json:"focused"`
	Pending bool `json:"pending"`
}

// ginkgoNode is used to construct the outline as a tree
type ginkgoNode struct {
	ginkgoMetadata
	Nodes []*ginkgoNode `json:"nodes"`
}

func main() {
	workDir, err := os.Getwd()
	fatalIfErr("could not get work dir", err)

	_, err = os.Stat("_out/tests/ginkgo")
	if err != nil {
		fatalIfErr("error finding ginkgo binary", err)
	}

	rfeIdMatcher := regexp.MustCompile("rfe\\_id\\:[0-9]+")

	testNamesToRFEId := map[string][]string{}
	err = filepath.WalkDir(filepath.Join(workDir, "tests"), func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_suite_test.go") {
			return nil
		}

		filename := path

		outlineCmd := exec.Command("_out/tests/ginkgo", "outline", "--format=json", filename)
		bytes, err := outlineCmd.Output()
		if err != nil {
			log.Debug(fmt.Sprintf("%q: error fetching ginkgo outline output for: %v", filename, err))
			return nil
		}

		var nodes []*ginkgoNode
		json.Unmarshal(bytes, &nodes)

		testNames := expandTestNames("", nodes)
		for _, testName := range testNames {
			rfeIdKey := "rfe_id:none"
			if rfeIdMatcher.MatchString(testName) {
				rfeIdKey = rfeIdMatcher.FindString(testName)
			}
			if _, exists := testNamesToRFEId[rfeIdKey]; !exists {
				testNamesToRFEId[rfeIdKey] = []string{}
			}
			testNamesToRFEId[rfeIdKey] = append(testNamesToRFEId[rfeIdKey], testName)
		}

		return nil
	})

	fatalIfErr("failed to validate test files", err)

	bytes, err := yaml.Marshal(testNamesToRFEId)
	fatalIfErr("failed to marshall map to yaml", err)
	testNamesWithRFEIdFile, err := ioutil.TempFile("", "testNamesWithRFEId-*.yaml")
	fatalIfErr("failed to open output file", err)
	os.WriteFile(testNamesWithRFEIdFile.Name(), bytes, fs.ModePerm)
	log.Infof("rfe_id test names written to file %q", testNamesWithRFEIdFile.Name())
}

func expandTestNames(parentText string, nodes []*ginkgoNode) []string {
	var result []string
	for _, node := range nodes {
		trimmedNodeTextWithParent := strings.Trim(fmt.Sprintf("%s %s", parentText, node.Text), " ")
		// node.Spec is only true for *It, *Specify, *Entry elements
		if node.Spec == true {
			// if the description inside a spec is itself referencing a const, then it will appear as "undefined" here
			// see tests/network/expose.go:207
			if node.Text != "undefined" {
				result = append(result, trimmedNodeTextWithParent)
			}
			continue
		}
		if len(node.Nodes) > 0 {
			result = append(result, expandTestNames(trimmedNodeTextWithParent, node.Nodes)...)
		}
	}
	return result
}

func fatalIfErr(message string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
