package longfile

import (
	_ "embed"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	exceptionDoc = "comma separated list of files with size greater than the default." +
		" each item in the list is with this format: <file name>:<number of lines>"
	maxFileLengthDoc     = "max allowed number of lines in a go file"
	maxTestFileLengthDoc = "max allowed number of lines in a go test file; test file is file within the " +
		"tests/ directory, or if its name ends with `_test.go`"
)

const (
	defaultMaxFileLength     = 1000
	defaultMaxTestFileLength = 1500
)

var Analyzer = newAnalyzer()

func newAnalyzer() *analysis.Analyzer {
	l := &longFileCfg{
		// todo: once in rules_go version v0.32.0 or higher, nogo should send the command
		//       line arguments (analyzer_flags field in nogo_configuration.json);
		//       then we can replace this initialization by: `exceptions: make(longFileExceptions)`
		// todo: try to reduce the size of each one of these files
		exceptions: longFileExceptions{
			"tests/infra_test.go":                1686,
			"tests/migration_test.go":            4382,
			"tests/operator_test.go":             3000,
			"tests/storage/restore.go":           1621,
			"tests/utils.go":                     2758,
			"tests/vm_test.go":                   2301,
			"tests/vmi_configuration_test.go":    3053,
			"tests/vmi_lifecycle_test.go":        1900,
			"tools/vms-generator/utils/utils.go": 1423,
		},
		maxFileLength:     defaultMaxFileLength,
		maxTestFileLength: defaultMaxTestFileLength,
	}
	a := &analysis.Analyzer{
		Name:             "longfile",
		Doc:              "detects if source code files are too long",
		Run:              l.checkPath,
		RunDespiteErrors: true,
	}

	a.Flags.Init("longfiles", flag.ExitOnError)
	a.Flags.Var(&l.exceptions, "exceptions", exceptionDoc)
	a.Flags.IntVar(&l.maxFileLength, "max-file-length", defaultMaxFileLength, maxFileLengthDoc)
	a.Flags.IntVar(&l.maxTestFileLength, "max-test-file-length", defaultMaxTestFileLength, maxTestFileLengthDoc)
	return a
}

type longFileExceptions map[string]int

// implement the flag.Value interface
func (e longFileExceptions) String() string {
	b := strings.Builder{}
	b.WriteRune('[')
	for k, v := range e {
		b.WriteString(fmt.Sprintf(`{%q: %d},`, k, v))
	}
	b.WriteRune(']')

	return b.String()
}

func (e longFileExceptions) Set(value string) error {
	items := strings.Split(value, ",")
	for _, item := range items {
		file := strings.Split(item, ":")
		if len(file) != 2 {
			return fmt.Errorf("can't parsr the file '%s'; it should be in format of <file name>:<number of lines>", item)
		}
		name := strings.TrimSpace(file[0])
		lines, err := strconv.Atoi(strings.TrimSpace(file[1]))
		if err != nil {
			return fmt.Errorf("can't parse number of lines from %s; %w", item, err)
		}
		e[name] = lines
	}

	return nil
}

type longFileCfg struct {
	exceptions        longFileExceptions
	maxFileLength     int
	maxTestFileLength int
}

func (l longFileCfg) checkPath(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		pos := pass.Fset.Position(file.End())
		if isGenerated(file, pos) {
			continue
		}

		fileName := pos.Filename
		if strings.Contains(fileName, "execroot/kubevirt/") {
			parts := strings.Split(pos.Filename, "execroot/kubevirt/")
			fileName = parts[len(parts)-1]
		}

		fileMax := l.maxAllowedFileLength(fileName)

		if pos.Line > fileMax {
			pass.Report(analysis.Diagnostic{
				Pos:     file.End(),
				Message: fmt.Sprintf("file has a length of %v which is more than %v lines; file name: %s", pos.Line, fileMax, pos.Filename),
			})
		}
	}
	return nil, nil
}

func (l longFileCfg) maxAllowedFileLength(fileName string) int {
	fileMax, exists := l.exceptions[fileName]
	if !exists {
		if strings.HasPrefix(fileName, "tests/") ||
			strings.Contains(fileName, "/tests/") ||
			strings.HasSuffix(fileName, "_test.go") {
			fileMax = l.maxTestFileLength
		} else {
			fileMax = l.maxFileLength
		}
	}

	return fileMax
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
