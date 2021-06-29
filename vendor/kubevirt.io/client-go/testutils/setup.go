package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
)

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
	log.Log.SetIOWriter(ginkgo.GinkgoWriter)
	// setup the connection between ginkgo and gomega
	gomega.RegisterFailHandler(ginkgo.Fail)

	// See https://github.com/bazelbuild/rules_go/blob/197699822e081dad064835a09825448a3e4cc2a2/go/core.rst#go_test
	// for context.
	testsWrapped := os.Getenv("GO_TEST_WRAP")
	outputFile := os.Getenv("XML_OUTPUT_FILE")

	// if run on bazel (XML_OUTPUT_FILE is not empty)
	// and rules_go is configured to not produce the junit xml
	// produce it here. Otherwise just run the default RunSpec
	if testsWrapped == "0" && outputFile != "" {
		testTarget := os.Getenv("TEST_TARGET")
		if config.GinkgoConfig.ParallelTotal > 1 {
			outputFile = fmt.Sprintf("%s-%d", outputFile, config.GinkgoConfig.ParallelNode)
		}

		ginkgo.RunSpecsWithDefaultAndCustomReporters(
			t,
			testTarget,
			[]ginkgo.Reporter{
				reporters.NewJUnitReporter(outputFile),
			},
		)
	} else {
		// Use the current filename as description for ginkgo
		ginkgo.RunSpecs(t, description)
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
		} else if os.IsNotExist(err) {
			continue
		} else if err != nil {
			panic(err)
		}
	}
}
