package triple

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestTriple(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.triple_suite_test.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Certificate Test Suite", []Reporter{junitReporter})
}
