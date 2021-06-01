package log

import (
	"fmt"
	"os"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

func TestLogging(t *testing.T) {
	Log.SetIOWriter(ginkgo.GinkgoWriter)
	gomega.RegisterFailHandler(ginkgo.Fail)
	testsWrapped := os.Getenv("GO_TEST_WRAP")
	outputFile := os.Getenv("XML_OUTPUT_FILE")
	description := "log"

	// if run on bazel (XML_OUTPUT_FILE is not empty)
	// and rules_go is configured to not produce the junit xml
	// produce it here. Otherwise just run the default RunSpec
	if testsWrapped == "0" && outputFile != "" {
		testTarget := os.Getenv("TEST_TARGET")
		if testTarget != "" {
			description = testTarget
		}
		if config.GinkgoConfig.ParallelTotal > 1 {
			outputFile = fmt.Sprintf("%s-%d", outputFile, config.GinkgoConfig.ParallelNode)
		}

		ginkgo.RunSpecsWithDefaultAndCustomReporters(
			t,
			description,
			[]ginkgo.Reporter{
				reporters.NewJUnitReporter(outputFile),
			},
		)
	} else {
		ginkgo.RunSpecs(t, description)
	}
}
