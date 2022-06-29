package longfile

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var max int
var exceptions map[string]int

var Analyzer = &analysis.Analyzer{
	Name: "longfile",
	Doc:  "detects if source code files are too long",
	Run:  checkPath,
}

func init() {
	max = 1000
	exceptions = map[string]int{
		"tests/utils.go":               3500,
		"tests/reporter/kubernetes.go": 2000,
	}
}

func checkPath(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		pos := pass.Fset.Position(file.End())
		if isGenerated(file, pos) {
			continue
		}

		parts := strings.Split(pos.Filename, "execroot/kubevirt/")
		filename := parts[len(parts)-1]

		fileMax, exists := exceptions[filename]
		if !exists {
			fileMax = max
		}

		if pos.Line > fileMax {
			pass.Report(analysis.Diagnostic{
				Pos:     file.End(),
				Message: fmt.Sprintf("file has a length of %v which is more than %v lines", pos.Line, fileMax),
			})
		}

	}
	return nil, nil
}

func isGenerated(file *ast.File, pos token.Position) bool {
	if strings.HasSuffix(pos.Filename, "_generated.go") {
		return true
	}
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "// Code generated ") && strings.HasSuffix(c.Text, " DO NOT EDIT.") {
				return true
			}
			if strings.HasPrefix(c.Text, "// Automatically generated ") && strings.HasSuffix(c.Text, " DO NOT EDIT!") {
				return true
			}
		}
	}

	return false
}
