package regressions_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRegressions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envtest Regression Tests")
}
