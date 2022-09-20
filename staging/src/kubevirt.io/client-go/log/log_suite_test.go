package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/gomega"

	v1reporter "kubevirt.io/client-go/reporter"
)

var afterSuiteReporters = []reporters.DeprecatedReporter{}

func TestLogging(t *testing.T) {
	Log.SetIOWriter(GinkgoWriter)
	gomega.RegisterFailHandler(Fail)
	testsWrapped := os.Getenv("GO_TEST_WRAP")
	outputFile := os.Getenv("XML_OUTPUT_FILE")
	_, description, _, _ := runtime.Caller(1)
	projectRoot := findRoot()
	description = strings.TrimPrefix(description, projectRoot)

	suiteConfig, _ := GinkgoConfiguration()

	// if run on bazel (XML_OUTPUT_FILE is not empty)
	// and rules_go is configured to not produce the junit xml
	// produce it here. Otherwise just run the default RunSpec
	if testsWrapped == "0" && outputFile != "" {
		testTarget := os.Getenv("TEST_TARGET")
		if testTarget != "" {
			description = testTarget
		}
		if suiteConfig.ParallelTotal > 1 {
			outputFile = fmt.Sprintf("%s-%d", outputFile, GinkgoParallelProcess())
		}

		afterSuiteReporters = append(afterSuiteReporters, v1reporter.NewV1JUnitReporter(outputFile))
	}

	RunSpecs(t, description)
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

var _ = ReportAfterSuite("TestLogging", func(report Report) {
	for _, reporter := range afterSuiteReporters {
		reporters.ReportViaDeprecatedReporter(reporter, report)
	}
})
