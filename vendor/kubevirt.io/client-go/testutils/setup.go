package testutils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/onsi/gomega/format"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
	v1reporter "kubevirt.io/client-go/reporter"
)

var afterSuiteReporters = []reporters.DeprecatedReporter{}

// KubeVirtTestSuiteSetup is the default setup function for kubevirts unittests.
// If tests are executed through bazel, the provided description is ignored. Instead
// the TEST_TARGET environment variable will be used to synchronize the output
// with bazels test output and make test navigation and detection consistent.
func KubeVirtTestSuiteSetup(t *testing.T) {
	_, description, _, _ := runtime.Caller(1)
	projectRoot := findRoot()
	description = strings.TrimPrefix(description, projectRoot)
	// Redirect writes to ginkgo writer to keep tests quiet when
	// they succeed
	log.Log.SetIOWriter(GinkgoWriter)
	// setup the connection between ginkgo and gomega
	gomega.RegisterFailHandler(Fail)

	// See https://github.com/bazelbuild/rules_go/blob/197699822e081dad064835a09825448a3e4cc2a2/go/core.rst#go_test
	// for context.
	testsWrapped := os.Getenv("GO_TEST_WRAP")
	outputFile := os.Getenv("XML_OUTPUT_FILE")

	suiteConfig, _ := GinkgoConfiguration()
	format.TruncatedDiff = false
	format.MaxLength = 8192

	// if run on bazel (XML_OUTPUT_FILE is not empty)
	// and rules_go is configured to not produce the junit xml
	// produce it here. Otherwise just run the default RunSpec
	if testsWrapped == "0" && outputFile != "" {
		testTarget := os.Getenv("TEST_TARGET")
		if suiteConfig.ParallelTotal > 1 {
			outputFile = fmt.Sprintf("%s-%d", outputFile, GinkgoParallelProcess())
		}

		afterSuiteReporters = append(afterSuiteReporters, v1reporter.NewV1JUnitReporter(outputFile))

		RunSpecs(t, testTarget)
	} else {
		RunSpecs(t, description)
	}
}

func findRoot() string {
	_, current, _, _ := runtime.Caller(0)
	for {
		current = filepath.Dir(current)
		if current == "/" || current == "." {
			return current
		}
		if _, err := os.Stat(filepath.Join(current, "WORKSPACE")); err == nil {
			return strings.TrimSuffix(current, "/") + "/"
		} else if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			panic(err)
		}
	}
}

var _ = ReportAfterSuite("KubeVirtTest", func(report Report) {
	for _, reporter := range afterSuiteReporters {
		reporters.ReportViaDeprecatedReporter(reporter, report)
	}
})
