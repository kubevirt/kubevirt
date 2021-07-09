package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/reporters"
	flag "github.com/spf13/pflag"

	"kubevirt.io/client-go/log"
)

func main() {

	if path := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); path != "" {
		if err := os.Chdir(path); err != nil {
			panic(err)
		}
	}

	var output string
	flag.StringVarP(&output, "output", "o", "-", "File to write the resulting junit file to, defaults to stdout (-)")
	flag.Parse()
	junitFiles := flag.Args()

	if len(junitFiles) == 0 {
		log.DefaultLogger().Critical("No JUnit files to merge provided.")
	}

	suites, err := loadJUnitFiles(junitFiles)
	if err != nil {
		log.DefaultLogger().Reason(err).Critical("Could not load JUnit files.")
	}

	result, err := mergeJUnitFiles(suites)
	if err != nil {
		log.DefaultLogger().Reason(err).Critical("Could not merge JUnit files")
	}

	writer, err := prepareOutput(output)
	if err != nil {
		log.DefaultLogger().Reason(err).Critical("Failed to prepare the output file")
	}

	err = writeJunitFile(writer, result)
	if err != nil {
		log.DefaultLogger().Reason(err).Critical("Failed to write the merged junit report")
	}
}

func loadJUnitFiles(fileGlobs []string) (suites []reporters.JUnitTestSuite, err error) {
	for _, fileglob := range fileGlobs {
		files, err := filepath.Glob(fileglob)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s: %v", file, err)
			}
			suite := reporters.JUnitTestSuite{}
			err = xml.NewDecoder(f).Decode(&suite)
			if err != nil {
				return nil, fmt.Errorf("failed to decode suite %s: %v", file, err)
			}
			suites = append(suites, suite)
		}
	}
	return suites, nil
}

func mergeJUnitFiles(suites []reporters.JUnitTestSuite) (result *reporters.JUnitTestSuite, err error) {
	result = &reporters.JUnitTestSuite{}
	ran := map[string]reporters.JUnitTestCase{}
	skipped := map[string]reporters.JUnitTestCase{}
	skippedList := []string{}

	// If tests ran in any of the suites, ensure it ends up in the ran-map and not in the skipped map.
	// If it only ever got skipped, keep it in the skip map
	for _, suite := range suites {
		for _, testcase := range suite.TestCases {
			if testcase.Skipped == nil {
				if _, exists := skipped[testcase.Name]; exists {
					delete(skipped, testcase.Name)
				}

				if _, exists := ran[testcase.Name]; exists {
					return nil, fmt.Errorf("test '%s' was executed multiple times", testcase.Name)
				}
				ran[testcase.Name] = testcase
				result.TestCases = append(result.TestCases, testcase)
			} else if testcase.Skipped != nil {
				if _, exists := ran[testcase.Name]; !exists {
					if _, exists := skipped[testcase.Name]; exists {
						testcase.Time = skipped[testcase.Name].Time + testcase.Time
					} else {
						skippedList = append(skippedList, testcase.Name)
					}
					skipped[testcase.Name] = testcase
				}
			}
		}
	}

	result.Name = "Merged Test Suite"
	for _, suite := range suites {
		result.Time += suite.Time
		result.Tests += suite.Tests
		result.Failures += suite.Failures
		result.Errors += suite.Errors
	}
	for _, casename := range skippedList {
		if _, exists := ran[casename]; exists {
			continue
		}
		if _, exists := skipped[casename]; !exists {
			panic("can't happen if we don't have a bug")
		}
		result.TestCases = append(result.TestCases, skipped[casename])
	}
	result.TestCases = append(result.TestCases)

	return result, nil
}

func prepareOutput(output string) (writer io.Writer, err error) {
	writer = os.Stdout
	if output != "-" && output != "" {
		writer, err = os.Create(output)
		if err != nil {
			return nil, err
		}
	}
	return writer, nil
}

func writeJunitFile(writer io.Writer, suite *reporters.JUnitTestSuite) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")
	err := encoder.Encode(suite)
	if err != nil {
		return err
	}
	return nil
}
