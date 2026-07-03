package envtest_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnvtest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envtest Tests")
}
