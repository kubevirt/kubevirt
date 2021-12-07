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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package testsLabels

import (
	"go/ast"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "testsLabels",
	Doc:  "reports label errors on tests",
	Run:  run,
}

var labelRegex *regexp.Regexp
var validLabelRegex *regexp.Regexp

func init() {
	labelRegex = regexp.MustCompile("\\[[^\\[\\]]+\\]")
	validLabelRegex = regexp.MustCompile("\\[((test_id|rfe_id)\\:([0-9]+|TODO)|(vendor|crit|level|posneg|label):.*|sig-[a-z-]+|Serial|rook-ceph|QUARANTINE|arm64|release-blocker|Conformance|IPv[46]|outside_connectivity|verify-nonroot|small|Sysprep)\\]")
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {

			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			funcIdent, ok := callExpr.Fun.(*ast.Ident)
			if !ok {
				return true
			}

			if funcIdent.Name != "Describe" &&
				funcIdent.Name != "Context" &&
				funcIdent.Name != "When" &&
				funcIdent.Name != "It" &&
				funcIdent.Name != "Specify" {
				return true
			}

			funcText, ok := callExpr.Args[0].(*ast.BasicLit)
			if !ok {
				return true
			}

			for _, label := range labelRegex.FindAllString(funcText.Value, -1) {
				if validLabelRegex.MatchString(label) {
					continue
				}
				pass.Reportf(n.Pos(), "Invalid test label '%s' detected", label)
			}

			return true
		})
	}
	return nil, nil
}
