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
	"io/fs"
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

var skipLeaves *regexp.Regexp

func init() {
	skipLeaves = regexp.MustCompile("By|BeforeEach|AfterEach")
}

func main() {
	workDir, err := os.Getwd()
	fatalIfErr("could not get work dir", err)

	_, err = os.Stat("_out/tests/ginkgo")
	if err != nil {
		fatalIfErr("error finding ginkgo binary", err)
	}

	err = filepath.WalkDir(filepath.Join(workDir, "tests"), func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_suite_test.go") {
			return nil
		}

		filename := path

		outlineCmd := exec.Command("_out/tests/ginkgo", "outline", "--format=json", filename)
		bytes, err := outlineCmd.Output()
		if err != nil {
			log.Debugf(fmt.Sprintf("%q: error fetching ginkgo outline output for: %v", filename), err)
			return nil
		}

		var nodes []*ginkgoNode
		json.Unmarshal(bytes, &nodes)

		testNames := expandTestNames("", nodes)
		testNamesUnique := map[string]struct{}{}
		for _, testName := range testNames {
			if _, exists := testNamesUnique[testName]; exists {
				return fmt.Errorf("%q: test name not unique: %q", filename, testName)
			}
			testNamesUnique[testName] = struct{}{}
		}

		return nil
	})

	fatalIfErr("failed to validate test files", err)
}

func expandTestNames(parentText string, nodes []*ginkgoNode) []string {
	var result []string
	for _, node := range nodes {
		if node.Text == "undefined" {
			continue
		}
		trimmedNodeTextWithParent := strings.Trim(fmt.Sprintf("%s %s", parentText, node.Text), " ")
		if len(node.Nodes) > 0 {
			result = append(result, expandTestNames(trimmedNodeTextWithParent, node.Nodes)...)
		} else {
			if skipLeaves.MatchString(node.Name) {
				continue
			}
			result = append(result, trimmedNodeTextWithParent)
		}
	}
	return result
}

func fatalIfErr(message string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
